package router

import (
	"cpamgt/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetTokenAccountRoutes(r *gin.RouterGroup, deps *RouterDeps) {
	g := r.Group("/token-account")
	g.Use(middleware.BaseAuthMiddleware(deps.Conf))
	{
		g.POST("", deps.TokenAccount.CreateTokenAccount)
		g.POST("/batch", deps.TokenAccount.CreateTokenAccountBatch)
		g.GET("/:id", deps.TokenAccount.GetTokenAccountByID)
		g.GET("", deps.TokenAccount.ListTokenAccounts)
	}
}
