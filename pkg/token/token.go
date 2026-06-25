// Package token ออก/ตรวจ JWT access token (HS256)
package token

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Manager ถือ secret + นโยบายอายุ token
type Manager struct {
	secret    []byte
	accessTTL time.Duration
	issuer    string
}

func NewManager(secret string, accessTTL time.Duration, issuer string) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, issuer: issuer}
}

// IssueAccess สร้าง access token ใส่ customer_id ใน sub
func (m *Manager) IssueAccess(customerID uint) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(uint64(customerID), 10),
		Issuer:    m.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// VerifyAccess ตรวจ token → คืน customer_id
func (m *Manager) VerifyAccess(raw string) (uint, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}), // ปิดช่อง alg-confusion / none
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return 0, errors.New("invalid subject")
	}
	return uint(id), nil
}
