package server

import (
	"context"
	"cpamgt/internal/model"
	"cpamgt/internal/pkg/log"

	"gorm.io/gorm"
)

type Migrate struct {
	db     *gorm.DB
	logger *log.Logger
}

func NewMigrate(db *gorm.DB, logger *log.Logger) *Migrate {
	return &Migrate{
		db:     db,
		logger: logger,
	}
}
func (m *Migrate) Start(ctx context.Context) error {
	if err := m.db.AutoMigrate(
		new(model.TokenAccount),
	); err != nil {
		m.logger.Error("AutoMigrate error", "err", err)
		return err
	}
	m.logger.Info("AutoMigrate success")
	//os.Exit(0)
	return nil
}
func (m *Migrate) Stop(ctx context.Context) error {
	return nil
}
