//go:build wireinject
// +build wireinject

package wire

import (
	"cpamgt/internal/config"
	"cpamgt/internal/pkg/log"
	"cpamgt/internal/repository"
	"cpamgt/internal/server"
	"cpamgt/pkg/app"

	"github.com/google/wire"
)

var repositorySet = wire.NewSet(repository.NewDB)

var serverSet = wire.NewSet(server.NewMigrate)

func newApp(
	migrateServer *server.Migrate,
) *app.App {
	return app.NewApp(app.WithServer(migrateServer), app.WithName("migration"))
}

func NewWire(*config.Config, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		serverSet,
		newApp,
	))
}
