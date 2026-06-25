package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrRefreshNotFound = token ไม่อยู่ใน store (หมดอายุ / ถูก consume ไปแล้ว / ปลอม)
var ErrRefreshNotFound = errors.New("refresh token not found")

type refreshEntry struct {
	CustomerID uint   `json:"customer_id"`
	FamilyID   string `json:"family_id"`
}

type RefreshStore interface {
	Save(ctx context.Context, token string, customerID uint, familyID string, ttl time.Duration) error
	Consume(ctx context.Context, token string) (customerID uint, familyID string, err error)
	RevokeAll(ctx context.Context, customerID uint) error
}

type redisRefreshStore struct{ rdb *redis.Client }

func NewRefreshStore(rdb *redis.Client) RefreshStore { return &redisRefreshStore{rdb: rdb} }

// เก็บ "hash ของ token" ไม่เก็บ token ดิบ → Redis รั่วก็ใช้ token ไม่ได้
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func refreshKey(h string) string    { return "refresh:" + h }
func custRefreshKey(id uint) string { return fmt.Sprintf("cust_refresh:%d", id) }

func (s *redisRefreshStore) Save(ctx context.Context, token string, customerID uint, familyID string, ttl time.Duration) error {
	h := hashToken(token)
	b, err := json.Marshal(refreshEntry{CustomerID: customerID, FamilyID: familyID})
	if err != nil {
		return err
	}
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, refreshKey(h), b, ttl)
	pipe.SAdd(ctx, custRefreshKey(customerID), h) // index ไว้ revoke ทั้งหมด
	pipe.Expire(ctx, custRefreshKey(customerID), ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("save refresh: %w", err)
	}
	return nil
}

// Consume อ่าน+ลบ atomically (GETDEL) → token ใช้ครั้งเดียว (rotation); ใช้ซ้ำ = ไม่เจอ = reject
func (s *redisRefreshStore) Consume(ctx context.Context, token string) (uint, string, error) {
	h := hashToken(token)
	raw, err := s.rdb.GetDel(ctx, refreshKey(h)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, "", ErrRefreshNotFound
	}
	if err != nil {
		return 0, "", fmt.Errorf("consume refresh: %w", err)
	}
	var e refreshEntry
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return 0, "", fmt.Errorf("unmarshal refresh: %w", err)
	}
	s.rdb.SRem(ctx, custRefreshKey(e.CustomerID), h)
	return e.CustomerID, e.FamilyID, nil
}

func (s *redisRefreshStore) RevokeAll(ctx context.Context, customerID uint) error {
	setKey := custRefreshKey(customerID)
	hashes, err := s.rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		return fmt.Errorf("revoke all (smembers): %w", err)
	}
	pipe := s.rdb.TxPipeline()
	for _, h := range hashes {
		pipe.Del(ctx, refreshKey(h))
	}
	pipe.Del(ctx, setKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("revoke all: %w", err)
	}
	return nil
}
