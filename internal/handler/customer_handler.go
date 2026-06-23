// Package handler คือ transport layer (Gin) — bind/validate request, เรียก service, shape response
// ไม่มี business logic; error ส่งผ่าน c.Error() ให้ ErrorHandler middleware จัดการรวมที่เดียว
package handler

import (
	"strconv"

	"github.com/apidet/go-api-service/internal/dto"
	"github.com/apidet/go-api-service/internal/service"
	"github.com/apidet/go-api-service/pkg/apperror"
	"github.com/apidet/go-api-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// CustomerHandler จัดการ HTTP สำหรับ resource customer (read-only)
type CustomerHandler struct {
	svc *service.CustomerService
}

// NewCustomerHandler inject service
func NewCustomerHandler(svc *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

// List ดึงลูกค้าแบบ paginate (?limit, ?offset) หรือค้นด้วย ?phone
//
//	@Summary	List customers
//	@Tags		customers
//	@Produce	json
//	@Param		limit	query	int		false	"จำนวนสูงสุด (default 50, max 200)"
//	@Param		offset	query	int		false	"ข้ามกี่ record"
//	@Param		phone	query	string	false	"ค้นด้วยเบอร์โทร (unique)"
//	@Success	200	{object}	response.Body
//	@Router		/api/v1/customers [get]
func (h *CustomerHandler) List(c *gin.Context) {
	var q dto.ListCustomerQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.Error(apperror.BadRequest("invalid query parameters"))
		return
	}

	// ?phone → lookup รายเดียว
	if q.Phone != "" {
		cust, err := h.svc.GetByPhone(c.Request.Context(), q.Phone)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(200, response.Success(cust))
		return
	}

	customers, err := h.svc.List(c.Request.Context(), q.Limit, q.Offset)
	if err != nil {
		c.Error(err)
		return
	}

	items := dto.NewCustomerList(customers)
	c.JSON(200, response.Paginated(items, response.PageMeta{
		Limit:  q.Limit,
		Offset: q.Offset,
		Count:  len(items),
	}))
}

// GetByID ดึงลูกค้าตาม customer_id
//
//	@Summary	Get customer by id
//	@Tags		customers
//	@Produce	json
//	@Param		id	path		int	true	"customer id"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/api/v1/customers/{id} [get]
func (h *CustomerHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("invalid id"))
		return
	}

	cust, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(200, response.Success(cust))
}
