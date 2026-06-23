package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS ตั้ง header พื้นฐาน (dev: อนุญาตทุก origin)
// production ควรจำกัด origin ผ่าน config — ปรับ allowOrigin ตามต้องการ
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, "+RequestIDHeader)
		c.Header("Access-Control-Expose-Headers", RequestIDHeader)

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
