package handler

import (
	"github.com/apidet/go-api-service/internal/appctx"
	"github.com/apidet/go-api-service/internal/service"
	"github.com/apidet/go-api-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// TicketHandler จัดการ HTTP สำหรับ resource ticket
type TicketHandler struct {
	svc *service.TicketService
}

// NewTicketHandler inject service
func NewTicketHandler(svc *service.TicketService) *TicketHandler {
	return &TicketHandler{svc: svc}
}

// GetActiveTicket ดึง ticket ที่ active ของลูกค้าที่ login อยู่ (id จาก token)
//
//	@Summary	Get my active tickets
//	@Tags		tickets
//	@Produce	json
//	@Success	200	{object}	response.Body
//	@Failure	401	{object}	response.Body
//	@Security	BearerAuth
//	@Router		/api/v1/tickets/active [get]
func (h *TicketHandler) GetActiveTicket(c *gin.Context) {
	id, _ := appctx.CustomerID(c.Request.Context()) // จาก token; Auth middleware รับประกันว่ามีค่าแล้ว

	tickets, err := h.svc.GetActive(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(200, response.Success(tickets)) // ส่งต่อ ticket ดิบทั้งหมดจาก curh1 (ทุก field)
}
