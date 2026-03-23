package router

import (
	"cpamgt/internal/config"
	"cpamgt/internal/handler"
	"cpamgt/internal/pkg/log"

	"github.com/gin-gonic/gin"
)

type RouterDeps struct {
	Conf         *config.Config
	Logger       *log.Logger
	Health       *handler.HealthHandler
	TaskMgt      *handler.TaskManageHandler
	TokenAccount *handler.TokenAccountHandler
	CpaAccount   *handler.CpaAccountHandler
}

func (r *RouterDeps) SetupRouter(group *gin.RouterGroup) {
	SetHealthRoutes(group, r)
	SetTokenAccountRoutes(group, r)
	SetTaskManageRoutes(group, r)
	SetCpaAccountRoutes(group, r)
}
