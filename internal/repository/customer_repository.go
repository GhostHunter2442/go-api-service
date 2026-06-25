// Package repository คือ data access layer — แยก service ออกจากรายละเอียดของ DB
package repository

import (
	"context"
	// "encoding/json"
	"errors"
	"fmt"

	"github.com/apidet/go-api-service/internal/model"
	"github.com/apidet/go-api-service/pkg/apperror"
	"gorm.io/gorm"
)

// CustomerRepository อ่านข้อมูลลูกค้า (read-only — ไม่แตะ schema เดิม)
type CustomerRepository interface {
	GetProfile(ctx context.Context, id uint) (*model.Customer, error)
	GetByPhone(ctx context.Context, phone string) (*model.Customer, error)
	List(ctx context.Context, limit, offset int) ([]model.Customer, error)
}

type customerRepository struct {
	db *gorm.DB
}

// NewCustomerRepository สร้าง implementation ที่ใช้ GORM
func NewCustomerRepository(db *gorm.DB) CustomerRepository {
	return &customerRepository{db: db}
}

// func (r *customerRepository) GetProfile(ctx context.Context, id uint) (*model.Customer, error) {
// 	var c model.Customer
// 	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, apperror.NotFound("customer not found")
// 		}
// 		return nil, fmt.Errorf("get customer by id: %w", err)
// 	}
// 	return &c, nil
// }

// GetProfile อ่านลูกค้ารายเดียวตาม customer_id (primary key) — read-only
func (r *customerRepository) GetProfile(ctx context.Context, id uint) (*model.Customer, error) {
	// ตัวแปรปลายทางสำหรับ scan ผลลัพธ์ลง struct (zero value ถ้าไม่เจอ)
	var customer model.Customer

	// สร้าง query บน connection เดิม โดยผูก ctx เข้าไปด้วย
	// WithContext: ให้ query ยกเลิก/timeout ตาม request context (กัน query ค้างเมื่อ client ตัดการเชื่อมต่อ)
	err := r.db.WithContext(ctx).
		// Select: ระบุเฉพาะคอลัมน์ที่ใช้ — ไม่ดึง field อ่อนไหว (password/token) และลด payload
		Select(
			"customer_id", "id_card", "customer_code", "customer_type",
			"phone_number", "firstname", "lastname",
			"gender", "date_of_birth", "email",
			"status", "total_point", "is_verify",
			"create_date", "update_date",
		).
		// First: ดึงแถวแรกที่ตรง primary key = id (เติม LIMIT 1); ถ้าไม่เจอคืน ErrRecordNotFound
		First(&customer, id).Error
	// b, _ := json.MarshalIndent(customer, "", "  ")
	// fmt.Printf("customer = %s\n", b)
	if err != nil {
		// แยกกรณี "ไม่พบข้อมูล" → แปลงเป็น AppError (404) ให้ ErrorHandler map เป็น HTTP 404
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("customer not found")
		}
		// error อื่น (เช่น DB ล่ม/connection หลุด) → wrap ไว้พร้อม context ให้ caller จัดการเป็น 500
		return nil, fmt.Errorf("get customer by id: %w", err)
	}

	// สำเร็จ — คืน pointer ของ customer ที่ scan มาแล้ว
	return &customer, nil
}

func (r *customerRepository) GetByPhone(ctx context.Context, phone string) (*model.Customer, error) {
	var c model.Customer
	if err := r.db.WithContext(ctx).Where("phone_number = ?", phone).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("customer not found")
		}
		return nil, fmt.Errorf("get customer by phone: %w", err)
	}
	return &c, nil
}

func (r *customerRepository) List(ctx context.Context, limit, offset int) ([]model.Customer, error) {
	var customers []model.Customer
	err := r.db.WithContext(ctx).
		Order("customer_id ASC").
		Limit(limit).
		Offset(offset).
		Find(&customers).Error
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	return customers, nil
}
