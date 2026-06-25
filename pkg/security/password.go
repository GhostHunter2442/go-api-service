// Package security จัดการ verify/hash password — แยก legacy (ของเดิม) ออกจาก argon2id (ของใหม่)
package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/alexedwards/argon2id"
)

// VerifyLegacyPassword ตรงกับ PHP เดิม: hash_hmac('SHA256', password, phone_number)
//
//	key = phone_number, message = password, output = lowercase hex
//
// ใช้แค่ตอน bootstrap (login ครั้งแรก) ก่อน migrate เป็น argon2id
func VerifyLegacyPassword(phone, password, storedHash string) bool {
	mac := hmac.New(sha256.New, []byte(phone))
	mac.Write([]byte(password))
	computed := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(computed), []byte(storedHash)) // constant-time
}

// withPepper ผสม pepper (secret ฝั่ง server) ก่อน hash — store รั่วก็ crack ไม่ได้ถ้าไม่มี pepper
func withPepper(password, pepper string) string {
	m := hmac.New(sha256.New, []byte(pepper))
	m.Write([]byte(password))
	return base64.RawStdEncoding.EncodeToString(m.Sum(nil))
}

// HashPassword สร้าง argon2id hash (salt สุ่มต่อคน + params ฝังในสตริง)
func HashPassword(password, pepper string) (string, error) {
	return argon2id.CreateHash(withPepper(password, pepper), argon2id.DefaultParams)
}

// VerifyPassword เทียบ password กับ argon2id hash
func VerifyPassword(password, pepper, encoded string) bool {
	ok, err := argon2id.ComparePasswordAndHash(withPepper(password, pepper), encoded)
	return err == nil && ok
}
