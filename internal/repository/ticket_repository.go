package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/apidet/go-api-service/pkg/apperror"
	"github.com/apidet/go-api-service/pkg/httpclient"
)

// virtualSoapPath คือ endpoint กลางของ internal API (function-RPC แบบ virtual SOAP)
// ต่อท้าย base host (PAWN_URL); ทุก operation ยิงมา path เดียว แล้วแยกด้วย field "function"
const virtualSoapPath = "/api/internal-api/virtualsoap"

// TicketRepository เข้าถึงข้อมูล ticket ของลูกค้า
// impl จริงยิง HTTP ไป external API (ไม่ใช่ DB) — แต่ interface ปิดบังเอาไว้
// service/handler จึงไม่ต้องรู้ว่าข้อมูลมาจากไหน (mock ใน test ได้)
type TicketRepository interface {
	// GetActiveByCard ดึง ticket ที่ active จากเลขบัตร (card_number = id_card ของลูกค้า)
	// คืน []json.RawMessage = ส่งต่อ ticket แต่ละก้อนจาก curh1 ดิบๆ (ทุก field ไม่ตกหล่น)
	GetActiveByCard(ctx context.Context, cardNumber string) ([]json.RawMessage, error)
}

// ticketClient เป็น impl ที่คุยกับ external API ผ่าน httpclient กลาง
type ticketClient struct {
	api *httpclient.Client
}

// NewTicketRepository สร้าง client ที่ยิงไป external API
// httpclient.Client ถือ base host (PAWN_URL) + connection pool/retry/timeout ไว้ภายใน
// สร้างครั้งเดียวตอน wire; ทุก function ยิง POST ที่ virtualSoapPath เดียวกัน
func NewTicketRepository(api *httpclient.Client) TicketRepository {
	return &ticketClient{api: api}
}

// virtualSoapRequest คือ envelope มาตรฐานของ internal API:
//
//	{ "function": "<ชื่อ function>", "data": { ... } }
type virtualSoapRequest struct {
	Function string `json:"function"`
	Data     any    `json:"data"`
}

// getActiveTicketData คือ payload ของ function "getactiveticket01"
type getActiveTicketData struct {
	CardNumber string `json:"card_number"`
}

// getActiveTicketResponse คือรูปแบบ response จริงของ virtualsoap:
//
//	{ "tstatus": { "fstatus": "success" }, "curh1": [ {ticket...}, ... ] }
//
// - tstatus.fstatus = สถานะของ call ("success" = สำเร็จ)
// - curh1 = list ของ ticket ที่ active (เก็บเป็น RawMessage = ไม่ parse field ภายใน ส่งต่อทั้งก้อน)
type getActiveTicketResponse struct {
	TStatus struct {
		FStatus string `json:"fstatus"`
	} `json:"tstatus"`
	Tickets []json.RawMessage `json:"curh1"`
}

// GetActiveByCard ยิง function "getactiveticket01" ไปที่ virtualsoap
//
// request body:
//
//	{ "function":"getactiveticket01", "data":{ "card_number":"<id_card>" } }
//
// การ map error ภายนอก → apperror ของระบบเรา:
//   - 404 จากปลายทาง → ถือว่าไม่มี ticket (คืน list ว่าง ไม่ใช่ error)
//   - status อื่นที่ผิดปกติ / network error → apperror.Internal (ไม่รั่วรายละเอียดออก client)
func (c *ticketClient) GetActiveByCard(ctx context.Context, cardNumber string) ([]json.RawMessage, error) {
	body := virtualSoapRequest{
		Function: "getactiveticket01",
		Data:     getActiveTicketData{CardNumber: cardNumber},
	}

	// log body ที่จะยิงออก (debug เท่านั้น — มี card_number = id_card ซึ่งเป็น PII อย่าเปิด debug บน prod)
	// if raw, err := json.Marshal(body); err == nil {
	// 	slog.Default().DebugContext(ctx, "virtualsoap request",
	// 		slog.String("function", body.Function),
	// 		slog.String("body", string(raw)),
	// 	)
	// }

	var out getActiveTicketResponse
	err := c.api.Do(ctx, httpclient.Request{
		Method: http.MethodPost,
		Path:   virtualSoapPath,
		Body:   body,
		Out:    &out,
	})
	if err != nil {
		if httpclient.StatusCode(err) == http.StatusNotFound {
			return []json.RawMessage{}, nil
		}
		return nil, apperror.Internal(fmt.Errorf("getactiveticket01 (card=%s): %w", cardNumber, err))
	}

	// ปลายทางตอบ 2xx แต่ fstatus ไม่ใช่ success → ถือว่าไม่มีตั๋ว (log ไว้สังเกต ไม่ 500 ให้ client)
	if !strings.EqualFold(out.TStatus.FStatus, "success") {
		slog.Default().WarnContext(ctx, "virtualsoap non-success",
			slog.String("function", "getactiveticket01"),
			slog.String("fstatus", out.TStatus.FStatus),
		)
		return []json.RawMessage{}, nil
	}
	if out.Tickets == nil {
		return []json.RawMessage{}, nil // กัน data:null ให้เป็น [] เสมอ
	}
	return out.Tickets, nil
}
