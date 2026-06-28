// Package handler คือ transport layer (Gin) — bind/validate request, เรียก service, shape response
// ไม่มี business logic; error ส่งผ่าน c.Error() ให้ ErrorHandler middleware จัดการรวมที่เดียว
package handler

import (
	"github.com/apidet/go-api-service/internal/appctx"
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
//	@Security	BearerAuth
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

// GetProfile ดึงโปรไฟล์ของลูกค้าที่ login อยู่ — อ่าน customer_id จาก token (context)
// ไม่ต้องรับ path param; middleware.Auth set ค่าไว้ให้แล้วหลัง verify token
//
//	@Summary	Get my profile
//	@Tags		customers
//	@Produce	json
//	@Success	200	{object}	response.Body
//	@Failure	401	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Security	BearerAuth
//	@Router		/api/v1/customers/profile [get]
func (h *CustomerHandler) GetProfile(c *gin.Context) {
	id, _ := appctx.CustomerID(c.Request.Context()) // จาก token; Auth middleware รับประกันว่ามีค่าแล้ว (ละ ok ได้)

	cust, err := h.svc.GetProfile(c.Request.Context(), uint(id))
	if err != nil {
		c.Error(err)
		return
	}
	// c.JSON(200, response.Success(cust))
	c.JSON(200, response.Success(dto.NewCustomerDetail(cust))) // ← map ผ่าน DTO
}

// UpdateProfile แก้ไขโปรไฟล์ของลูกค้าที่ login อยู่ — id จาก token, แก้ได้เฉพาะ field ที่อนุญาต
//
//	@Summary	Update my profile
//	@Tags		customers
//	@Accept		json
//	@Produce	json
//	@Param		body	body		dto.UpdateProfileRequest	true	"field ที่ต้องการแก้ (ส่งเฉพาะที่จะแก้)"
//	@Success	200	{object}	response.Body
//	@Failure	400	{object}	response.Body
//	@Failure	401	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Security	BearerAuth
//	@Router		/api/v1/customers/profile [patch]
func (h *CustomerHandler) UpdateProfile(c *gin.Context) {
	id, _ := appctx.CustomerID(c.Request.Context()) // จาก token; Auth middleware รับประกันว่ามีค่าแล้ว (ละ ok ได้)

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("invalid request body"))
		return
	}

	cust, err := h.svc.UpdateProfile(c.Request.Context(), id, req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(200, response.Success(dto.NewCustomerDetail(cust)))
}
