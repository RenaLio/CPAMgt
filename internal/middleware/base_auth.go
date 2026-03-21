package middleware

import (
	v1 "cpamgt/api/v1"
	"cpamgt/internal/config"
	"strings"

	"github.com/gin-gonic/gin"
)

func BaseAuthMiddleware(conf *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		token = strings.TrimSpace(token)
		token = strings.TrimPrefix(token, "Bearer ")
		if token != conf.Security.Secret {
			v1.HandleError(c, v1.ErrUnauthorized, "unauthorized")
		}
		c.Next()
	}
}
