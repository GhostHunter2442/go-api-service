// Package config โหลดค่า config จาก environment variables เป็น struct เดียว
// ใช้ที่จุด startup เท่านั้น — layer อื่นรับเป็น argument ไม่อ่าน env เอง (testable)
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config คือค่า config ทั้งหมดของ service
type Config struct {
	Env    string // "development" | "production"
	HTTP   HTTPConfig
	DB     DBConfig
	Log    LogConfig
	Auth   AuthConfig
	Redis  RedisConfig
	Ticket TicketConfig
}

// TicketConfig ค่าเชื่อม external API ของ ticket (ไม่ใช่ DB)
// BaseURL ใช้ PAWN_URL (host ของ pawnshop API) — path ของแต่ละ function ต่อในโค้ด
type TicketConfig struct {
	BaseURL string        // base host จาก env PAWN_URL เช่น https://pawnshop-api-dev.../
	APIKey  string        // ส่งผ่าน header (อย่า hardcode — อ่านจาก env)
	Timeout time.Duration // timeout ต่อ request
}

// LogConfig ค่าเกี่ยวกับ structured logging
type LogConfig struct {
	Level  string // debug | info | warn | error
	Format string // json | text
}

// IsProduction true เมื่อรันโหมด production (ใช้ตั้ง gin ReleaseMode ฯลฯ)
func (c Config) IsProduction() bool { return c.Env == "production" }

// HTTPConfig ค่าเกี่ยวกับ HTTP server
type HTTPConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DBConfig ค่าเกี่ยวกับการต่อ SQL Server
type DBConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Name         string
	Encrypt      string // "disable" | "true" | "false" (dev มักใช้ "disable")
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLife  time.Duration
}

// เพิ่มใน Config struct:
//   Auth  AuthConfig
//   Redis RedisConfig

type AuthConfig struct {
	JWTSecret  string
	Pepper     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Issuer     string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// Load อ่านค่าจาก env (มี default สำหรับ dev) — เรียกครั้งเดียวตอน main
//
// dev: ถ้ามีไฟล์ .env จะ load เข้า process ให้อัตโนมัติ
// production: ไม่มี .env ก็ข้าม ใช้ env จริงของ host/container ตามปกติ
func Load() Config {
	_ = godotenv.Load() // best-effort: ไม่มีไฟล์ก็ไม่เป็นไร

	return Config{
		Env: getEnv("APP_ENV", "development"),
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		HTTP: HTTPConfig{
			Port:            getEnv("HTTP_PORT", "8080"),
			ReadTimeout:     getEnvDuration("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout:    getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout: getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		DB: DBConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnvInt("DB_PORT", 1433),
			User:         getEnv("DB_USER", "sa"),
			Password:     getEnv("DB_PASSWORD", ""),
			Name:         getEnv("DB_NAME", "master"),
			Encrypt:      getEnv("DB_ENCRYPT", "disable"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLife:  getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Auth: AuthConfig{
			JWTSecret:  getEnv("JWT_SECRET", ""),
			Pepper:     getEnv("PEPPER", ""),
			AccessTTL:  getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: getEnvDuration("JWT_REFRESH_TTL", 720*time.Hour),
			Issuer:     getEnv("JWT_ISSUER", "go-api-service"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Ticket: TicketConfig{
			BaseURL: getEnv("PAWN_URL", "http://localhost"),
			APIKey:  getEnv("TICKET_API_KEY", ""),
			Timeout: getEnvDuration("TICKET_API_TIMEOUT", 10*time.Second),
		},
	}
}

// DSN สร้าง connection string รูปแบบ sqlserver:// สำหรับ go-mssqldb / GORM
// ใช้ url.URL เพื่อ escape username/password ที่มีอักขระพิเศษให้ถูกต้อง
func (d DBConfig) DSN() string {
	q := url.Values{}
	q.Add("database", d.Name)
	q.Add("encrypt", d.Encrypt)

	u := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(d.User, d.Password),
		Host:     fmt.Sprintf("%s:%d", d.Host, d.Port),
		RawQuery: q.Encode(),
	}
	return u.String()
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
