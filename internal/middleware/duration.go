package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func DurationMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Header("X-Server-Time", strconv.FormatInt(start.UnixMilli(), 10))
		c.Next()

	}
}
