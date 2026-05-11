package router

import (
	"cpamgt/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetTaskManageRoutes(r *gin.RouterGroup, deps *RouterDeps) {
	g := r.Group("/tasks")
	g.Use(middleware.BaseAuthMiddleware(deps.Conf))
	{
		g.GET("", deps.TaskMgt.ListTasks)
		g.GET("/:name", deps.TaskMgt.GetTask)
		g.PATCH("/:name", deps.TaskMgt.UpdateNamedTaskConfig)
	}
}
