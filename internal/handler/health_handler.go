package handler

import (
	"net/http"

	"github.com/apidet/go-api-service/pkg/apperror"
	"github.com/apidet/go-api-service/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler รวม endpoint สำหรับ liveness / readiness
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler สร้าง handler โดยรับ *gorm.DB ไว้ ping
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Liveness ตอบว่า process ยังมีชีวิต (ไม่แตะ DB)
//
//	@Summary	Liveness probe
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	response.Body
//	@Router		/healthz [get]
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, response.Success(gin.H{"status": "ok"}))
}

// Readiness ping SQL Server เพื่อยืนยันว่าพร้อมรับ traffic
//
//	@Summary	Readiness probe (ping DB)
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	response.Body
//	@Failure	503	{object}	response.Body
//	@Router		/readyz [get]
func (h *HealthHandler) Readiness(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil {
		c.Error(apperror.New(http.StatusServiceUnavailable, "DB_UNAVAILABLE", "database unavailable", err))
		return
	}
	if err := sqlDB.PingContext(c.Request.Context()); err != nil {
		c.Error(apperror.New(http.StatusServiceUnavailable, "DB_UNAVAILABLE", "database ping failed", err))
		return
	}
	c.JSON(http.StatusOK, response.Success(gin.H{"status": "ready"}))
}
