// Package service คือ business logic layer — ไม่รู้จัก HTTP และไม่ผูกกับ GORM ตรงๆ
package service

import (
	"context"

	"github.com/apidet/go-api-service/internal/dto"
	"github.com/apidet/go-api-service/internal/model"
	"github.com/apidet/go-api-service/internal/repository"
	"github.com/apidet/go-api-service/pkg/apperror"
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

// GetProfile ดึงลูกค้าตาม customer_id
func (s *CustomerService) GetProfile(ctx context.Context, id uint) (*model.Customer, error) {
	return s.repo.GetProfile(ctx, id)
}

// UpdateProfile อัปเดตโปรไฟล์ของลูกค้าตาม customer_id (เฉพาะ field ที่ส่งมา)
func (s *CustomerService) UpdateProfile(ctx context.Context, id uint, req dto.UpdateProfileRequest) (*model.Customer, error) {
	fields := req.ToUpdateMap()
	if len(fields) == 0 {
		return nil, apperror.BadRequest("no fields to update")
	}
	return s.repo.UpdateProfile(ctx, id, fields)
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
