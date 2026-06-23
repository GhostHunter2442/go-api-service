// Package logger ตั้งค่า structured logger (slog) ให้ทั้ง service
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New สร้าง *slog.Logger ตาม level + format
//
//	level:  debug | info | warn | error
//	format: json (production) | text (dev อ่านง่าย)
func New(level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var h slog.Handler
	if strings.ToLower(format) == "text" {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
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
