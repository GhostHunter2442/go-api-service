// Command api คือ entrypoint ของ service — โหลด config, ตั้ง logger, ต่อ DB,
// wire dependency, รัน Gin HTTP server พร้อม graceful shutdown
//
//	@title			go-api-service API
//	@version		1.0
//	@description	REST API service (Gin + GORM + SQL Server) — layered architecture
//	@BasePath		/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				ใส่ "Bearer <access_token>" (มี Bearer นำหน้า เว้นวรรค 1 ที)
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/apidet/go-api-service/internal/config"
	"github.com/apidet/go-api-service/internal/database"
	"github.com/apidet/go-api-service/internal/handler"
	"github.com/apidet/go-api-service/internal/repository"
	"github.com/apidet/go-api-service/internal/server"
	"github.com/apidet/go-api-service/internal/service"
	"github.com/apidet/go-api-service/pkg/logger"
	"github.com/apidet/go-api-service/pkg/token"
)

func main() {
	cfg := config.Load()

	// 0) structured logger (slog) — ตั้งเป็น default ให้ทั้ง process
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(log)

	// 1) ต่อ SQL Server — fail fast ถ้าต่อไม่ติด
	db, err := database.NewSQLServer(cfg.DB)
	if err != nil {
		log.Error("connect db", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer database.Close(db)
	log.Info("connected to sql server")

	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		log.Error("connect redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer rdb.Close()
	log.Info("connected to redis")

	// 2) wire dependency: repository → service → handler
	customerRepo := repository.NewCustomerRepository(db)
	customerSvc := service.NewCustomerService(customerRepo)

	tokenMgr := token.NewManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL, cfg.Auth.Issuer)
	pwStore := repository.NewPasswordStore(rdb)
	refreshStore := repository.NewRefreshStore(rdb)
	authSvc := service.NewAuthService(customerRepo, tokenMgr, pwStore, refreshStore, cfg.Auth.Pepper, cfg.Auth.RefreshTTL)

	handlers := server.Handlers{
		Health:   handler.NewHealthHandler(db),
		Customer: handler.NewCustomerHandler(customerSvc),
		Auth:     handler.NewAuthHandler(authSvc, cfg.Auth.AccessTTL, cfg.Auth.RefreshTTL, cfg.IsProduction()),
	}

	// 3) สร้าง Gin server (middleware stack + routes อยู่ใน package server)
	srv := server.New(cfg, log, handlers, tokenMgr)

	// 4) graceful shutdown ด้วย signal.NotifyContext
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("server listening", slog.String("addr", srv.Addr), slog.String("env", cfg.Env))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("server stopped")
}
