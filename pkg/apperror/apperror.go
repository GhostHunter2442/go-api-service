// Package apperror นิยาม error แบบ typed ที่ map ไปเป็น HTTP status ได้
// layer ใดก็ได้ return *AppError แล้วให้ error-handler middleware แปลงเป็น response กลาง
package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError คือ error ระดับ application พร้อม metadata สำหรับ transport
type AppError struct {
	Status  int    // HTTP status
	Code    string // machine-readable code เช่น "NOT_FOUND"
	Message string // ข้อความที่ปลอดภัยจะส่งให้ client
	Err     error  // error ต้นทาง (ไว้ log — ไม่ส่งออก client)
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap รองรับ errors.Is / errors.As
func (e *AppError) Unwrap() error { return e.Err }

// New สร้าง AppError แบบกำหนดเอง
func New(status int, code, message string, err error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Err: err}
}

// As ดึง *AppError ออกจาก error chain (false ถ้าไม่ใช่)
func As(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

// --- constructor ที่ใช้บ่อย ---

func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, "BAD_REQUEST", message, nil)
}

func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

func Forbidden(message string) *AppError {
	return New(http.StatusForbidden, "FORBIDDEN", message, nil)
}

func NotFound(message string) *AppError {
	return New(http.StatusNotFound, "NOT_FOUND", message, nil)
}

// Internal ห่อ error ต้นทางเป็น 500 (message generic ไม่รั่วรายละเอียดออก client)
func Internal(err error) *AppError {
	return New(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", err)
}
