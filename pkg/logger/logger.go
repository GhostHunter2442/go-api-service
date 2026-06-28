// Package logger ตั้งค่า structured logger (slog) ให้ทั้ง service
package logger

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

// New สร้าง *slog.Logger ตาม level + format
//
//	level:  debug | info | warn | error
//	format: json (production — เครื่องอ่าน/รวม log ได้) | text (dev — สี + อ่านง่าย)
func New(level, format string) *slog.Logger {
	lv := parseLevel(level)

	var h slog.Handler
	if strings.ToLower(format) == "text" {
		// dev: colored + จัดรูปแบบให้คนอ่านง่าย (เวลา · level สี · msg · key=value)
		h = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      lv,
			TimeFormat: time.TimeOnly,                      // 15:04:05 (ตัดวันที่ออก ให้สั้น)
			NoColor:    !isatty(os.Stdout) || noColorEnv(), // ปิดสีถ้าไม่ใช่ terminal (เช่น pipe ไป file/CI)
		})
	} else {
		// production: JSON ให้ log aggregator (Loki/ELK) parse ได้
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
	}
	return slog.New(h)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// isatty true ถ้า output เป็น terminal จริง (มีสีได้) — ถ้าถูก redirect ไปไฟล์/pipe จะ false
func isatty(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// noColorEnv รองรับ convention NO_COLOR (https://no-color.org) — ตั้ง env แล้วปิดสีได้
func noColorEnv() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}
