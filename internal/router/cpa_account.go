package router

import (
	"cpamgt/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetCpaAccountRoutes(r *gin.RouterGroup, deps *RouterDeps) {
	group := r.Group("/cpa-account")
	group.Use(middleware.BaseAuthMiddleware(deps.Conf))
	{
		group.GET("/config", deps.CpaAccount.GetCpaConfig)
		group.POST("/config", deps.CpaAccount.SetCpaConfig)
	}
}
