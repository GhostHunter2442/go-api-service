package middleware

import (
	"strings"

	"github.com/apidet/go-api-service/internal/appctx"
	"github.com/apidet/go-api-service/pkg/apperror"
	"github.com/apidet/go-api-service/pkg/token"
	"github.com/gin-gonic/gin"
)

func Auth(tm *token.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.Error(apperror.Unauthorized("missing bearer token"))
			c.Abort()
			return
		}
		id, err := tm.VerifyAccess(strings.TrimSpace(strings.TrimPrefix(h, "Bearer ")))
		if err != nil {
			c.Error(apperror.Unauthorized("invalid or expired token"))
			c.Abort()
			return
		}
		// invariant: token ที่ valid ต้องมี customer_id > 0 — ตรวจที่ขอบครั้งเดียว
		// handler ที่อยู่หลัง middleware นี้จึงเชื่อได้ว่ามี id เสมอ ไม่ต้องเช็คซ้ำ
		if id == 0 {
			c.Error(apperror.Unauthorized("missing customer in token"))
			c.Abort()
			return
		}
		// ฝัง customer_id ลง request context มาตรฐาน (ผ่าน appctx — unexported key)
		// แทนที่ c.Request เดิมด้วยตัวที่ผูก context ใหม่ → layer ล่านอ่านผ่าน c.Request.Context() ได้
		c.Request = c.Request.WithContext(appctx.WithCustomerID(c.Request.Context(), id))
		c.Next()
	}
}
