package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/apidet/go-api-service/pkg/apperror"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit จำกัด perMinute request/นาที ต่อ IP (burst = พุ่งได้กี่ครั้งติดกัน)
func RateLimit(perMinute, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*clientLimiter)
	r := rate.Every(time.Minute / time.Duration(perMinute))

	go func() { // cleanup IP ที่เงียบไป
		for {
			time.Sleep(3 * time.Minute)
			mu.Lock()
			for ip, cl := range clients {
				if time.Since(cl.lastSeen) > 5*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		cl, ok := clients[ip]
		if !ok {
			cl = &clientLimiter{limiter: rate.NewLimiter(r, burst)}
			clients[ip] = cl
		}
		cl.lastSeen = time.Now()
		allowed := cl.limiter.Allow()
		mu.Unlock()
		if !allowed {
			c.Header("Retry-After", "60")
			c.Error(apperror.New(http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, try again later", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}
