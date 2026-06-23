// Package dto เก็บ struct สำหรับ bind request และ shape response
// แยกจาก model (DB) เพื่อไม่ผูก API contract เข้ากับ schema ตรงๆ
package dto

import (
	"time"

	"github.com/apidet/go-api-service/internal/model"
)

// ListCustomerQuery bind จาก query string ของ GET /customers
type ListCustomerQuery struct {
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
	Phone  string `form:"phone"`
}

// CustomerListItem เป็น response แบบย่อสำหรับ list (ไม่ดัมพ์ทุกคอลัมน์)
type CustomerListItem struct {
	CustomerID   uint    `json:"customer_id"`
	CustomerCode *string `json:"customer_code"`
	Firstname    *string `json:"firstname"`
	Lastname     *string `json:"lastname"`
	PhoneNumber  string  `json:"phone_number"`
	Status       *string `json:"status"`
	TotalPoint   *int    `json:"total_point"`
}

// NewCustomerListItem map จาก model → DTO ย่อ
func NewCustomerListItem(c model.Customer) CustomerListItem {
	return CustomerListItem{
		CustomerID:   c.CustomerID,
		CustomerCode: c.CustomerCode,
		Firstname:    c.Firstname,
		Lastname:     c.Lastname,
		PhoneNumber:  c.PhoneNumber,
		Status:       c.Status,
		TotalPoint:   c.TotalPoint,
	}
}

// NewCustomerList map ทั้ง slice
func NewCustomerList(cs []model.Customer) []CustomerListItem {
	items := make([]CustomerListItem, 0, len(cs))
	for _, c := range cs {
		items = append(items, NewCustomerListItem(c))
	}
	return items
}

// CustomerDetail เป็น response แบบเต็มสำหรับ GetByID
type CustomerDetail struct {
	CustomerID   uint       `json:"customer_id"`
	IDCard       *string    `json:"id_card"`
	CustomerCode *string    `json:"customer_code"`
	CustomerType *string    `json:"customer_type"`
	PhoneNumber  string     `json:"phone_number"`
	Firstname    *string    `json:"firstname"`
	Lastname     *string    `json:"lastname"`
	Gender       *string    `json:"gender"`
	DateOfBirth  *time.Time `json:"date_of_birth"`
	Email        *string    `json:"email"`
	Status       *string    `json:"status"`
	TotalPoint   *int       `json:"total_point"`
	IsVerify     *int       `json:"is_verify"`
	CreateDate   *time.Time `json:"create_date"`
	UpdateDate   *time.Time `json:"update_date"`
}

func NewCustomerDetail(c *model.Customer) CustomerDetail {
	return CustomerDetail{
		CustomerID:   c.CustomerID,
		IDCard:       c.IDCard,
		CustomerCode: c.CustomerCode,
		CustomerType: c.CustomerType,
		PhoneNumber:  c.PhoneNumber,
		Firstname:    c.Firstname,
		Lastname:     c.Lastname,
		Gender:       c.Gender,
		DateOfBirth:  c.DateOfBirth,
		Email:        c.Email,
		Status:       c.Status,
		TotalPoint:   c.TotalPoint,
		IsVerify:     c.IsVerify,
		CreateDate:   c.CreateDate,
		UpdateDate:   c.UpdateDate,
	}
}
