package router

import (
	"cpamgt/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetHealthRoutes(group *gin.RouterGroup, deps *RouterDeps) {
	group.GET("/ping", deps.Health.Ping)
	auth := group.Use(middleware.BaseAuthMiddleware(deps.Conf))
	auth.GET("/health", deps.Health.GetHealth)
}
