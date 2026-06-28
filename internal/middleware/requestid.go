// Package middleware รวม gin middleware กลางของ service
package middleware

import (
	"github.com/apidet/go-api-service/pkg/httpclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDKey คีย์ใน gin.Context
	RequestIDKey = "request_id"
	// RequestIDHeader ชื่อ header ทั้งขาเข้า/ขาออก
	RequestIDHeader = "X-Request-ID"
)

// RequestID แนบ request id ให้ทุก request (ใช้ของที่ client ส่งมา ถ้าไม่มีก็ gen ใหม่)
// แล้ว echo กลับใน response header เพื่อ trace ข้าม service ได้
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(RequestIDKey, id)
		c.Header(RequestIDHeader, id)
		// ฝังลง request context ด้วย เพื่อให้ outbound call (httpclient) แนบ X-Request-ID ต่อได้
		c.Request = c.Request.WithContext(httpclient.WithRequestID(c.Request.Context(), id))
		c.Next()
	}
}

// GetRequestID ดึง request id จาก context (ว่างถ้าไม่มี)
func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(RequestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
