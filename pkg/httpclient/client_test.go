package httpclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fastRetry = retry แบบ backoff สั้นๆ ให้ test ไม่ช้า
func fastRetry() RetryConfig {
	return RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}
}

func TestGet_DecodesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tickets" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"id":7,"status":"active"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, WithRetry(fastRetry()))
	var out struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	}
	if err := c.Get(context.Background(), "/tickets", &out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if out.ID != 7 || out.Status != "active" {
		t.Fatalf("decoded = %+v", out)
	}
}

func TestRetry_On503ThenSuccess(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	c := New(srv.URL, WithRetry(fastRetry()))
	if err := c.Get(context.Background(), "/x", nil); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestNoRetry_OnPOST(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := New(srv.URL, WithRetry(fastRetry()))
	err := c.Post(context.Background(), "/x", map[string]string{"a": "b"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("POST should not retry (non-idempotent), got %d calls", got)
	}
}

func TestAPIError_StatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":"nope"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, WithRetry(fastRetry()))
	err := c.Get(context.Background(), "/missing", nil)
	if StatusCode(err) != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (err=%v)", StatusCode(err), err)
	}
	ae, ok := AsAPIError(err)
	if !ok || ae.Body == "" {
		t.Fatalf("expected APIError with body, got %v", err)
	}
}

func TestPropagatesRequestID(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Request-ID")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	c := New(srv.URL, WithRetry(fastRetry()))
	ctx := WithRequestID(context.Background(), "req-123")
	if err := c.Get(ctx, "/x", nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "req-123" {
		t.Fatalf("X-Request-ID = %q, want req-123", got)
	}
}

func TestContextCancel_NotRetried(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // ยกเลิกก่อนยิง

	c := New(srv.URL, WithRetry(fastRetry()))
	if err := c.Get(ctx, "/x", nil); err == nil {
		t.Fatal("expected context error")
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("canceled context should not hit server, got %d calls", got)
	}
}
