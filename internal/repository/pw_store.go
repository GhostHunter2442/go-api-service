package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ShadowHash = argon2 hash ที่ migrate แล้ว + snapshot ของ legacy hash (ไว้ตรวจว่าระบบเก่า reset password ไหม)
type ShadowHash struct {
	Argon2         string `json:"argon2"`
	LegacySnapshot string `json:"legacy"`
}

type PasswordStore interface {
	Get(ctx context.Context, customerID uint) (*ShadowHash, error)
	Set(ctx context.Context, customerID uint, argon2, legacySnapshot string) error
}

type redisPasswordStore struct{ rdb *redis.Client }

func NewPasswordStore(rdb *redis.Client) PasswordStore { return &redisPasswordStore{rdb: rdb} }

func pwKey(id uint) string { return fmt.Sprintf("pwhash:%d", id) }

func (s *redisPasswordStore) Get(ctx context.Context, customerID uint) (*ShadowHash, error) {
	raw, err := s.rdb.Get(ctx, pwKey(customerID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // ยังไม่เคย migrate
	}
	if err != nil {
		return nil, fmt.Errorf("get shadow hash: %w", err)
	}
	var sh ShadowHash
	if err := json.Unmarshal([]byte(raw), &sh); err != nil {
		return nil, fmt.Errorf("unmarshal shadow hash: %w", err)
	}
	return &sh, nil
}

func (s *redisPasswordStore) Set(ctx context.Context, customerID uint, argon2, legacySnapshot string) error {
	b, err := json.Marshal(ShadowHash{Argon2: argon2, LegacySnapshot: legacySnapshot})
	if err != nil {
		return err
	}
	// TTL 0 = ถาวรจนถูกเขียนทับ (ถ้า Redis หาย ก็ fall back legacy + migrate ใหม่)
	if err := s.rdb.Set(ctx, pwKey(customerID), b, 0).Err(); err != nil {
		return fmt.Errorf("set shadow hash: %w", err)
	}
	return nil
}
