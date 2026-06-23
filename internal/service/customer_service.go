// Package service คือ business logic layer — ไม่รู้จัก HTTP และไม่ผูกกับ GORM ตรงๆ
package service

import (
	"context"

	"github.com/apidet/go-api-service/internal/model"
	"github.com/apidet/go-api-service/internal/repository"
)

const (
	defaultListLimit = 50
	maxListLimit     = 200
)

// CustomerService รวม use case ที่เกี่ยวกับลูกค้า
type CustomerService struct {
	repo repository.CustomerRepository
}

// NewCustomerService inject repository เข้ามา (dependency inversion)
func NewCustomerService(repo repository.CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
}

// GetByID ดึงลูกค้าตาม customer_id
func (s *CustomerService) GetByID(ctx context.Context, id uint) (*model.Customer, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByPhone ดึงลูกค้าตามเบอร์โทร (unique)
func (s *CustomerService) GetByPhone(ctx context.Context, phone string) (*model.Customer, error) {
	return s.repo.GetByPhone(ctx, phone)
}

// List ดึงลูกค้าแบบ paginate — clamp limit กัน query ทั้งตาราง
func (s *CustomerService) List(ctx context.Context, limit, offset int) ([]model.Customer, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}
