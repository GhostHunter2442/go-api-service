// Package database จัดการการเชื่อมต่อ SQL Server ผ่าน GORM
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/apidet/go-api-service/internal/config"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewSQLServer สร้าง *gorm.DB ที่ต่อกับ SQL Server พร้อมตั้งค่า connection pool
// และ ping ตรวจสอบว่าต่อติดจริงก่อน return (fail fast ตอน startup)
//
// ฟังก์ชันนี้คือจุดเดียวที่เปิด connection — layer อื่นรับ *gorm.DB ไปใช้ต่อ
func NewSQLServer(cfg config.DBConfig) (*gorm.DB, error) {
	db, err := gorm.Open(sqlserver.Open(cfg.DSN()), &gorm.Config{
		// ปิด default transaction ต่อ statement → throughput ดีกว่า
		SkipDefaultTransaction: true,
		// PrepareStmt cache prepared statement ลด round-trip
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlserver: %w", err)
	}

	// ตั้งค่า connection pool ผ่าน *sql.DB ที่อยู่ข้างใต้ GORM
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLife)

	// ping เพื่อยืนยันว่าต่อ DB ได้จริง ภายใน timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping sqlserver: %w", err)
	}

	return db, nil
}

// Close ปิด connection pool — เรียกตอน shutdown
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
