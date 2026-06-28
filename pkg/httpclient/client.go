// Package httpclient เป็น client กลางสำหรับเรียก external REST API
//
// ทำไมต้องมี wrapper แทนที่จะใช้ net/http ตรงๆ ทุกที่:
//   - รวม cross-cutting concern ไว้ที่เดียว: timeout, retry+backoff, connection pool,
//     logging, propagate request-id, จำกัดขนาด body, drain/close ให้ครบ (reuse connection)
//   - repository แค่บอก method/path/body/out — ไม่ต้องเขียน boilerplate ของ net/http ซ้ำ
//   - error เป็น typed (*APIError) → caller map เป็น apperror ได้เอง
//
// ทิศการใช้งาน: repository ถือ *Client (inject ตอน wire ใน main) แล้วเรียก
//
//	var out dto.X
//	err := c.Get(ctx, "/path", &out)
//	err := c.Post(ctx, "/path", reqBody, &out)
package httpclient

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ค่า default — override ได้ผ่าน Option ตอน New
const (
	defaultTimeout      = 10 * time.Second
	defaultMaxBodyBytes = 5 << 20 // 5 MiB — กัน external ตอบใหญ่จน OOM
	defaultUserAgent    = "go-api-service"
)

// Request คือคำขอ 1 ครั้งแบบ declarative — ฟังก์ชันกลาง Do รับ struct นี้ตัวเดียว
type Request struct {
	Method  string            // GET/POST/PUT/PATCH/DELETE
	Path    string            // path ต่อท้าย baseURL (เช่น "/tickets/active")
	Query   url.Values        // query string (optional)
	Headers map[string]string // header เฉพาะ request นี้ (ทับ default ได้)
	Body    any               // ถ้าไม่ใช่ nil → marshal เป็น JSON ส่งไป
	Out     any               // ถ้าไม่ใช่ nil → decode JSON ของ response 2xx ลงตัวนี้
}

// Client เป็น external API client ที่ thread-safe (ใช้ร่วมกันทุก goroutine ได้)
// สร้างครั้งเดียวตอน startup แล้ว reuse — อย่าสร้างใหม่ต่อ request (จะเสีย connection pool)
type Client struct {
	base      *url.URL
	http      *http.Client
	log       *slog.Logger
	headers   map[string]string
	userAgent string
	maxBody   int64
	retry     RetryConfig
}

// RetryConfig คุมพฤติกรรม retry — retry เฉพาะ "transient" (network error / 429 / 5xx)
// และเฉพาะ method ที่ idempotent (GET/PUT/DELETE/HEAD) เว้นแต่เปิด RetryNonIdempotent
type RetryConfig struct {
	MaxAttempts        int           // จำนวนครั้งรวม (1 = ไม่ retry); <=0 จะถูกตั้งเป็น 1
	BaseDelay          time.Duration // backoff ตั้งต้น (exponential: base, 2*base, 4*base...)
	MaxDelay           time.Duration // เพดาน backoff ต่อครั้ง
	RetryNonIdempotent bool          // เปิดถ้า external API การันตี POST/PATCH ซ้ำได้ปลอดภัย
}

// DefaultRetry ค่าที่เหมาะกับ production ทั่วไป (3 ครั้ง, backoff 200ms→…สูงสุด 2s)
func DefaultRetry() RetryConfig {
	return RetryConfig{MaxAttempts: 3, BaseDelay: 200 * time.Millisecond, MaxDelay: 2 * time.Second}
}

// Option ปรับแต่ง Client ตอนสร้าง (functional options)
type Option func(*Client)

// WithHTTPClient ใส่ *http.Client เอง (เช่นใส่ mTLS/proxy) — ถ้าไม่ใส่จะสร้างตัว default ที่ tune แล้ว
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// WithTimeout ตั้ง timeout รวมต่อ request (ครอบ dial+TLS+อ่าน body); ใช้กับ default client เท่านั้น
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if c.http != nil {
			c.http.Timeout = d
		}
	}
}

// WithLogger ใส่ slog logger (ไม่ใส่ → ใช้ slog.Default)
func WithLogger(l *slog.Logger) Option { return func(c *Client) { c.log = l } }

// WithHeader เพิ่ม default header ที่ติดไปทุก request (เช่น Authorization / X-Api-Key)
func WithHeader(k, v string) Option {
	return func(c *Client) { c.headers[k] = v }
}

// WithUserAgent ตั้งค่า User-Agent
func WithUserAgent(ua string) Option { return func(c *Client) { c.userAgent = ua } }

// WithMaxBodyBytes จำกัดขนาด response body ที่จะอ่าน (กัน OOM)
func WithMaxBodyBytes(n int64) Option { return func(c *Client) { c.maxBody = n } }

// WithRetry ตั้งค่า retry policy
func WithRetry(r RetryConfig) Option { return func(c *Client) { c.retry = r } }

// New สร้าง Client ใหม่ — baseURL เช่น "https://api.partner.com/v2"
// ถ้า baseURL parse ไม่ได้จะ panic ตอน startup (fail fast — config ผิดต้องรู้ทันที)
func New(baseURL string, opts ...Option) *Client {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		panic(fmt.Sprintf("httpclient: invalid base url %q: %v", baseURL, err))
	}

	c := &Client{
		base:      u,
		http:      &http.Client{Timeout: defaultTimeout, Transport: defaultTransport()},
		log:       slog.Default(),
		headers:   map[string]string{},
		userAgent: defaultUserAgent,
		maxBody:   defaultMaxBodyBytes,
		retry:     DefaultRetry(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// defaultTransport คือ http.Transport ที่ tune connection pool ไว้สำหรับ production
// (ค่า default ของ stdlib มี MaxIdleConnsPerHost=2 ซึ่งน้อยไปเวลายิงปลายทางเดียวถี่ๆ)
func defaultTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100, // สำคัญ: ปลายทางเดียวจึงดันให้สูง เพื่อ reuse keep-alive
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// --- ฟังก์ชันกลาง ---

// Do คือ entry point เดียวที่ยิง request จริง — Get/Post/... ทั้งหมดเรียกผ่านตัวนี้
// คืน *APIError เมื่อ status นอก 2xx; คืน error ห่อ (network/timeout/decode) ในกรณีอื่น
func (c *Client) Do(ctx context.Context, r Request) error {
	full := c.resolve(r.Path, r.Query)

	// buffer body ไว้เป็น []byte เพื่อสร้าง reader ใหม่ได้ทุกครั้งที่ retry
	var bodyBytes []byte
	if r.Body != nil {
		b, err := json.Marshal(r.Body)
		if err != nil {
			return fmt.Errorf("httpclient: marshal request body: %w", err)
		}
		bodyBytes = b
	}

	attempts := c.retry.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	idempotent := isIdempotent(r.Method) || c.retry.RetryNonIdempotent

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		// เคารพ context ก่อนเริ่มแต่ละครั้ง (อาจถูก cancel ระหว่างรอ backoff)
		if err := ctx.Err(); err != nil {
			return err
		}

		retryAfter, err := c.attempt(ctx, r, full, bodyBytes, attempt)
		if err == nil {
			return nil
		}
		lastErr = err

		// ตัดสินใจ retry: ต้อง idempotent + transient + ยังเหลือ attempt
		if attempt == attempts || !idempotent || !isRetryable(err) {
			break
		}

		delay := c.backoff(attempt, retryAfter)
		c.log.WarnContext(ctx, "external api retry",
			slog.String("method", r.Method), slog.String("url", full),
			slog.Int("attempt", attempt), slog.Duration("backoff", delay),
			slog.String("error", err.Error()),
		)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}

// attempt ยิง 1 ครั้ง คืน (retryAfter, err) — retryAfter > 0 เมื่อ server ขอ (429/503 + Retry-After)
func (c *Client) attempt(ctx context.Context, r Request, full string, body []byte, attempt int) (time.Duration, error) {
	start := time.Now()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, full, reader)
	if err != nil {
		return 0, fmt.Errorf("httpclient: build request: %w", err)
	}
	c.setHeaders(ctx, req, r, body != nil)

	resp, err := c.http.Do(req)
	if err != nil {
		// network/timeout/cancel — ห่อไว้ (transient → ให้ retry ตัดสินใจจาก isRetryable)
		return 0, &transportError{err: err}
	}
	defer drainClose(resp.Body)

	// อ่าน body แบบจำกัดขนาด (กัน external ตอบใหญ่เกิน)
	data, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBody))
	if err != nil {
		return 0, fmt.Errorf("httpclient: read response body: %w", err)
	}

	c.log.LogAttrs(ctx, slog.LevelDebug, "external api call",
		slog.String("method", r.Method), slog.String("url", full),
		slog.Int("status", resp.StatusCode), slog.Int("attempt", attempt),
		slog.Duration("latency", time.Since(start)),
		slog.Int("resp_bytes", len(data)),
		slog.String("response", snippet(data, 1024)), // ดู body ที่ปลายทางตอบกลับ (debug เท่านั้น)
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return retryAfter, &APIError{
			Method:     r.Method,
			URL:        full,
			StatusCode: resp.StatusCode,
			Body:       snippet(data, 512),
		}
	}

	if r.Out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, r.Out); err != nil {
			return 0, fmt.Errorf("httpclient: decode response: %w", err)
		}
	}
	return 0, nil
}

func (c *Client) setHeaders(ctx context.Context, req *http.Request, r Request, hasBody bool) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.headers { // default ของ client
		req.Header.Set(k, v)
	}
	for k, v := range r.Headers { // เฉพาะ request นี้ (ทับ default ได้)
		req.Header.Set(k, v)
	}
	// propagate request id ข้าม service เพื่อ trace ปลายทางต่อได้ (ถ้ามีใน context)
	if rid := requestIDFromContext(ctx); rid != "" {
		req.Header.Set("X-Request-ID", rid)
	}
}

// resolve รวม baseURL + path + query เป็น absolute URL
// path ว่าง → ใช้ base URL ตามเดิม (กรณี base เป็น URL เต็มของ endpoint อยู่แล้ว)
func (c *Client) resolve(path string, q url.Values) string {
	u := *c.base
	if p := strings.TrimLeft(path, "/"); p != "" {
		u.Path = strings.TrimRight(u.Path, "/") + "/" + p
	}
	if len(q) > 0 {
		u.RawQuery = q.Encode()
	}
	return u.String()
}

// backoff คำนวณเวลารอแบบ exponential + jitter; ถ้า server ส่ง Retry-After มาให้ใช้ค่านั้นเป็นพื้น
func (c *Client) backoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return capDelay(retryAfter, c.retry.MaxDelay)
	}
	base := c.retry.BaseDelay
	if base <= 0 {
		base = 200 * time.Millisecond
	}
	// 2^(attempt-1) * base
	d := base << (attempt - 1)
	d = capDelay(d, c.retry.MaxDelay)
	return d/2 + jitter(d/2) // full jitter ครึ่งหนึ่ง กัน thundering herd
}

// --- wrapper ตาม method (ergonomic) ---

// Get ยิง GET แล้ว decode response ลง out (out=nil ได้ถ้าไม่สนใจ body)
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.Do(ctx, Request{Method: http.MethodGet, Path: path, Out: out})
}

// Post ยิง POST พร้อม body (JSON) แล้ว decode response ลง out
func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	return c.Do(ctx, Request{Method: http.MethodPost, Path: path, Body: body, Out: out})
}

// Put ยิง PUT พร้อม body แล้ว decode response ลง out
func (c *Client) Put(ctx context.Context, path string, body, out any) error {
	return c.Do(ctx, Request{Method: http.MethodPut, Path: path, Body: body, Out: out})
}

// Patch ยิง PATCH พร้อม body แล้ว decode response ลง out
func (c *Client) Patch(ctx context.Context, path string, body, out any) error {
	return c.Do(ctx, Request{Method: http.MethodPatch, Path: path, Body: body, Out: out})
}

// Delete ยิง DELETE (out=nil ได้)
func (c *Client) Delete(ctx context.Context, path string, out any) error {
	return c.Do(ctx, Request{Method: http.MethodDelete, Path: path, Out: out})
}

// --- helper ภายใน ---

// transportError ห่อ error ระดับ network/transport เพื่อแยกออกจาก *APIError
type transportError struct{ err error }

func (e *transportError) Error() string { return e.err.Error() }
func (e *transportError) Unwrap() error { return e.err }

// isRetryable: transient เท่านั้น — network error, 429, หรือ 5xx (ยกเว้น 501 Not Implemented)
func isRetryable(err error) bool {
	// context ถูก cancel/timeout จาก caller → ไม่ retry (เคารพ deadline เดิม)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var te *transportError
	if errors.As(err, &te) {
		return true
	}
	if ae, ok := AsAPIError(err); ok {
		if ae.StatusCode == http.StatusTooManyRequests {
			return true
		}
		return ae.StatusCode >= 500 && ae.StatusCode != http.StatusNotImplemented
	}
	return false
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
}

// drainClose อ่าน body ที่เหลือทิ้งแล้วปิด — ให้ Transport reuse keep-alive connection ได้
func drainClose(rc io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(rc, 4<<10))
	_ = rc.Close()
}

func snippet(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

func capDelay(d, max time.Duration) time.Duration {
	if max > 0 && d > max {
		return max
	}
	return d
}

// jitter คืนค่าสุ่มในช่วง [0, d) ด้วย crypto/rand (ไม่ต้อง seed, ปลอดภัยข้าม goroutine)
func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(d)))
	if err != nil {
		return d / 2 // fallback แบบ deterministic
	}
	return time.Duration(n.Int64())
}

// parseRetryAfter รองรับทั้งแบบ "วินาที" (เช่น "120") และ HTTP-date
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}
