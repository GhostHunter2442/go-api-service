package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/apidet/go-api-service/internal/repository"
	"github.com/apidet/go-api-service/pkg/apperror"
	"github.com/apidet/go-api-service/pkg/security"
	"github.com/apidet/go-api-service/pkg/token"
)

type AuthService struct {
	repo       repository.CustomerRepository
	tokens     *token.Manager
	pwStore    repository.PasswordStore
	refresh    repository.RefreshStore
	pepper     string
	refreshTTL time.Duration
	dummyHash  string // ไว้กิน timing ตอน user ไม่มี กัน enumeration
}

func NewAuthService(
	repo repository.CustomerRepository,
	tokens *token.Manager,
	pwStore repository.PasswordStore,
	refresh repository.RefreshStore,
	pepper string,
	refreshTTL time.Duration,
) *AuthService {
	dummy, _ := security.HashPassword("timing-equalizer", pepper)
	return &AuthService{
		repo: repo, tokens: tokens, pwStore: pwStore, refresh: refresh,
		pepper: pepper, refreshTTL: refreshTTL, dummyHash: dummy,
	}
}

// TokenPair ผลลัพธ์ login/refresh
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

var errInvalidCredentials = apperror.Unauthorized("invalid credentials")

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Login: verify (argon2id shadow → legacy fallback + migrate) แล้วออก token pair
func (s *AuthService) Login(ctx context.Context, phone, password string) (*TokenPair, error) {
	cust, err := s.repo.GetByPhone(ctx, phone)
	if err != nil {
		security.VerifyPassword(password, s.pepper, s.dummyHash) // กิน timing
		return nil, errInvalidCredentials
	}
	if cust.Status == nil || *cust.Status != "active" || cust.Password == nil {
		security.VerifyPassword(password, s.pepper, s.dummyHash)
		return nil, errInvalidCredentials
	}

	shadow, err := s.pwStore.Get(ctx, cust.CustomerID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	// ระบบเดิม reset password → legacy เปลี่ยน → ทิ้ง shadow แล้ว bootstrap ใหม่
	if shadow != nil && shadow.LegacySnapshot != *cust.Password {
		shadow = nil
	}

	var ok bool
	if shadow != nil {
		ok = security.VerifyPassword(password, s.pepper, shadow.Argon2) // เส้นทางปกติ
	} else {
		ok = security.VerifyLegacyPassword(phone, password, *cust.Password) // bootstrap
		if ok {
			if h, herr := security.HashPassword(password, s.pepper); herr == nil {
				_ = s.pwStore.Set(ctx, cust.CustomerID, h, *cust.Password) // migrate (best-effort)
			}
		}
	}
	if !ok {
		return nil, errInvalidCredentials
	}
	return s.issuePair(ctx, cust.CustomerID, "")
}

// Refresh: consume เก่า → ออกคู่ใหม่ (rotation)
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	customerID, familyID, err := s.refresh.Consume(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshNotFound) {
			return nil, apperror.Unauthorized("invalid refresh token")
		}
		return nil, apperror.Internal(err)
	}
	return s.issuePair(ctx, customerID, familyID)
}

// Logout: เพิกถอนทุก session ของ user นี้
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	customerID, _, err := s.refresh.Consume(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshNotFound) {
			return nil // logout ซ้ำ/หมดอายุ ถือว่าสำเร็จ
		}
		return apperror.Internal(err)
	}
	return s.refresh.RevokeAll(ctx, customerID)
}

func (s *AuthService) issuePair(ctx context.Context, customerID uint, familyID string) (*TokenPair, error) {
	access, err := s.tokens.IssueAccess(customerID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	refreshTok, err := randomToken()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if familyID == "" {
		if familyID, err = randomToken(); err != nil {
			return nil, apperror.Internal(err)
		}
	}
	if err := s.refresh.Save(ctx, refreshTok, customerID, familyID, s.refreshTTL); err != nil {
		return nil, apperror.Internal(err)
	}
	return &TokenPair{AccessToken: access, RefreshToken: refreshTok}, nil
}
