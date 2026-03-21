//go:build wireinject
// +build wireinject

package wire

import (
	"cpamgt/internal/config"
	"cpamgt/internal/handler"
	"cpamgt/internal/repository"
	"cpamgt/internal/router"
	"cpamgt/internal/server"
	task2 "cpamgt/internal/server/task"
	"cpamgt/internal/service"
	"cpamgt/internal/task"
	"cpamgt/pkg/app"
	"cpamgt/pkg/log"
	"cpamgt/pkg/server/http"

	"github.com/google/wire"
)

var repositorySet = wire.NewSet(
	repository.NewDB,
	repository.NewRepository,
	repository.NewTransaction,
	repository.NewTokenAccountRepo,
)

var serviceSet = wire.NewSet(
	service.NewService,
	service.NewTokenAccountService,
	service.NewCpaAccountService,
)

var handlerSet = wire.NewSet(
	handler.NewHandler,
	handler.NewHealthHandler,
	handler.NewTaskManageHandler,
	handler.NewTokenAccountHandler,
	handler.NewCpaAccountHandler,
)

var serverSet = wire.NewSet(server.NewHTTPServer)

var taskSet = wire.NewSet(task.NewMockTask, task.NewCodexCheckTask, task.NewCpaTask)

func NewTaskServer(
	logger *log.Logger,
	mockTask *task.MockTask,
	codexCheckTask *task.CodexCheckTask,
	caTask *task.CpaTask,
) *task2.TaskServer {
	taskList := []task2.Task{
		mockTask,
		codexCheckTask,
		caTask,
	}
	return task2.NewTaskServer(logger, taskList...)
}

var taskExportSet = wire.NewSet(task2.GetTaskManager)

// build App
func newApp(
	httpServer *http.Server,
	taskServer *task2.TaskServer,
	// task *server.Task,
) *app.App {
	return app.NewApp(
		app.WithServer(httpServer, taskServer),
		app.WithName("demo-server"),
	)
}

func NewWire(*config.Config, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		serviceSet,
		taskSet,
		NewTaskServer,
		taskExportSet,
		handlerSet,
		serverSet,
		wire.Struct(new(router.RouterDeps), "*"),
		newApp,
	))
}
