package httpclient

import (
	"errors"
	"fmt"
)

// APIError คือ error เมื่อ external API ตอบ status นอกช่วง 2xx
// พก status + body snippet ไว้ให้ caller (repository) ตัดสินใจ map เป็น apperror ได้เอง
// (เช่น 404 ภายนอก → apperror.NotFound, ที่เหลือ → apperror.Internal)
type APIError struct {
	Method     string // HTTP method ที่ยิงไป
	URL        string // ปลายทาง (ไว้ log/debug — ระวัง query ที่มีของลับ)
	StatusCode int    // status ที่ได้กลับมา
	Body       string // เนื้อหา response (ตัดให้สั้นแล้ว) ไว้ดูสาเหตุ
}

func (e *APIError) Error() string {
	return fmt.Sprintf("external api %s %s: status %d: %s", e.Method, e.URL, e.StatusCode, e.Body)
}

// AsAPIError ดึง *APIError ออกจาก error chain (false ถ้าไม่ใช่)
func AsAPIError(err error) (*APIError, bool) {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

// StatusCode คืน status จาก error ถ้าเป็น *APIError, ไม่งั้นคืน 0
// ใช้สั้นๆ ใน repository: if httpclient.StatusCode(err) == 404 { ... }
func StatusCode(err error) int {
	if ae, ok := AsAPIError(err); ok {
		return ae.StatusCode
	}
	return 0
}
