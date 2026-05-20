package middlewares

import (
	"sync"
	"time"

	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

var limiter = &rateLimiter{
	requests: make(map[string][]time.Time),
	limit:    2000, // Maximum 2000 requests per minute
	window:   time.Minute,
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		limiter.mu.Lock()
		defer limiter.mu.Unlock()

		now := time.Now()
		requests := limiter.requests[ip]

		var validRequests []time.Time
		for _, t := range requests {
			if now.Sub(t) < limiter.window {
				validRequests = append(validRequests, t)
			}
		}

		if len(validRequests) >= limiter.limit {
			response.Error(c, 429, "RATE_LIMIT_EXCEEDED", "Too many requests, please try again later")
			c.Abort()
			return
		}

		validRequests = append(validRequests, now)
		limiter.requests[ip] = validRequests

		c.Next()
	}
}
