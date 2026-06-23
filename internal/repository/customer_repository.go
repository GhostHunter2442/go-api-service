// Package repository คือ data access layer — แยก service ออกจากรายละเอียดของ DB
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/apidet/go-api-service/internal/model"
	"github.com/apidet/go-api-service/pkg/apperror"
	"gorm.io/gorm"
)

// CustomerRepository อ่านข้อมูลลูกค้า (read-only — ไม่แตะ schema เดิม)
type CustomerRepository interface {
	GetByID(ctx context.Context, id uint) (*model.Customer, error)
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

func (r *customerRepository) GetByID(ctx context.Context, id uint) (*model.Customer, error) {
	var c model.Customer
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("customer not found")
		}
		return nil, fmt.Errorf("get customer by id: %w", err)
	}
	return &c, nil
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
