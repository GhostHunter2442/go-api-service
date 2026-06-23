package middleware

import (
	"log/slog"
	"net/http"

	"github.com/apidet/go-api-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// Recovery ดัก panic ทุกตัว → log + ตอบ 500 ในรูป envelope กลาง (ไม่ให้ server ตาย)
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		log.Error("panic recovered",
			slog.Any("error", err),
			slog.String("request_id", GetRequestID(c)),
			slog.String("path", c.Request.URL.Path),
		)
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			response.Failure("INTERNAL_ERROR", "internal server error", GetRequestID(c)),
		)
	})
}
