package middleware

import (
	"strings"

	"github.com/apidet/go-api-service/pkg/apperror"
	"github.com/apidet/go-api-service/pkg/token"
	"github.com/gin-gonic/gin"
)

// ContextCustomerID key สำหรับเก็บ customer_id ใน gin.Context (handler อ่านด้วย c.GetUint)
const ContextCustomerID = "customer_id"

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
		c.Set(ContextCustomerID, id)
		c.Next()
	}
}
