package middleware

import (
	"cpamgt/pkg/log"
	"time"

	"github.com/gin-gonic/gin"
)

func LoggingMiddleware(logger *log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		logger.FromContext(c.Request.Context()).Info(
			"http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(startedAt).String(),
			"ip", c.ClientIP(),
		)

	}
}
