package middleware

import (
	"log/slog"

	"github.com/apidet/go-api-service/pkg/apperror"
	"github.com/apidet/go-api-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// ErrorHandler รวมการแปลง error → HTTP response ไว้ที่เดียว
//
// handler แค่ `c.Error(err)` แล้ว return — ไม่ต้องเขียน JSON error เอง
//   - ถ้าเป็น *AppError → ใช้ status/code/message ของมัน
//   - error อื่นๆ → 500 (log error ต้นทาง, ไม่รั่วรายละเอียดออก client)
func ErrorHandler(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}
		// ถ้า handler เขียน response ไปแล้ว ไม่ทับ
		if c.Writer.Written() {
			return
		}

		err := c.Errors.Last().Err
		appErr, ok := apperror.As(err)
		if !ok {
			appErr = apperror.Internal(err)
		}

		if appErr.Status >= 500 {
			log.Error("request error",
				slog.String("request_id", GetRequestID(c)),
				slog.String("path", c.Request.URL.Path),
				slog.String("error", appErr.Error()),
			)
		}

		c.AbortWithStatusJSON(
			appErr.Status,
			response.Failure(appErr.Code, appErr.Message, GetRequestID(c)),
		)
	}
}
