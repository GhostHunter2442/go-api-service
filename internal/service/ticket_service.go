package service

import (
	"context"
	"encoding/json"

	"github.com/apidet/go-api-service/internal/repository"
	"github.com/apidet/go-api-service/pkg/apperror"
)

// TicketService รวม use case ที่เกี่ยวกับ ticket
//
// ticket อยู่ที่ external API ซึ่งระบุลูกค้าด้วย card_number (= id_card)
// แต่ token รู้แค่ customer_id → service จึง lookup customer จาก DB เพื่อเอา id_card ก่อนยิงต่อ
type TicketService struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
}

// NewTicketService inject ทั้ง ticket repo (external API) และ customer repo (DB) สำหรับ resolve id_card
func NewTicketService(tickets repository.TicketRepository, customers repository.CustomerRepository) *TicketService {
	return &TicketService{tickets: tickets, customers: customers}
}

// GetActive คืน ticket ที่ยัง active ของลูกค้าตาม customer_id (จาก token)
//
// ขั้นตอน: customer_id → ดึง customer จาก DB → ใช้ id_card เป็น card_number ยิง external API
func (s *TicketService) GetActive(ctx context.Context, customerID uint) ([]json.RawMessage, error) {
	cust, err := s.customers.GetProfile(ctx, customerID)
	if err != nil {
		return nil, err // GetProfile คืน apperror.NotFound/Internal ให้อยู่แล้ว
	}
	if cust.IDCard == nil || *cust.IDCard == "" {
		return nil, apperror.New(422, "NO_ID_CARD", "customer has no id card on file", nil)
	}
	return s.tickets.GetActiveByCard(ctx, *cust.IDCard)
}
