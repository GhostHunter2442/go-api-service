// Package server ประกอบ Gin engine + middleware stack + http.Server
package server

import (
	"log/slog"
	"net/http"

	"github.com/apidet/go-api-service/internal/config"
	"github.com/apidet/go-api-service/internal/handler"
	"github.com/apidet/go-api-service/internal/middleware"
	"github.com/apidet/go-api-service/pkg/token"
	"github.com/gin-gonic/gin"
)

// Handlers รวม handler ทั้งหมดที่ wire มาแล้วจาก main
type Handlers struct {
	Health   *handler.HealthHandler
	Customer *handler.CustomerHandler
	Auth     *handler.AuthHandler
	Ticket   *handler.TicketHandler
}

// New สร้าง *http.Server ที่ใช้ Gin engine พร้อม middleware stack
//
// ลำดับ middleware (สำคัญ):
//
//	Recovery → RequestID → Logger → CORS → ErrorHandler → routes
//
// Recovery นอกสุด (กัน panic ทุกชั้น), Logger เห็น status สุดท้ายที่ ErrorHandler เขียน
func New(cfg config.Config, log *slog.Logger, h Handlers, tm *token.Manager) *http.Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()
	engine.Use(
		middleware.Recovery(log),
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.CORS(),
		middleware.ErrorHandler(log),
	)

	registerRoutes(engine, h, tm)

	return &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      engine,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}
}
