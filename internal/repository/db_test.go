package repository

import (
	"context"
	"cpamgt/internal/config"
	"cpamgt/internal/model"
	"cpamgt/internal/pkg/log"
	"errors"
	"log/slog"
	"os"
	"testing"

	"gorm.io/gorm"
)

var conf config.Config
var logger *log.Logger
var db *gorm.DB

func TestMain(m *testing.M) {
	// local test for nested transaction
	conf.Data.DB.User.Driver = "mysql"
	conf.Data.DB.User.DSN = "root:1234567890.@tcp(localhost:3306)/default_db_x?charset=utf8mb4&parseTime=True&loc=Local"
	logger = &log.Logger{
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
	db = NewDB(&conf, logger)
	db.AutoMigrate(&model.TokenAccount{})

	code := m.Run()
	os.Exit(code)
}

// compare nested transaction
// 1. create token account a1
// 2. create token account a2
// 3. throw error
// 4. expect a1 a2 are not created
func TestNestedTransaction(t *testing.T) {
	repo := NewRepository(logger, db)
	slog.Info("test nested transaction")
	tokenRepo := TokenAccountRepo{repo}
	ctx := context.Background()
	tm := NewTransaction(repo)
	err := tm.Transaction(ctx, func(ctx context.Context) error {
		logger.Info("start transaction")
		err := tokenRepo.Create(ctx, &model.TokenAccount{
			IDToken: "a1",
		})
		if err != nil {
			logger.Error("create token account failed", "err", err)
			return err
		}
		e := tm.Transaction(ctx, func(ctx context.Context) error {
			tokenRepo.Create(ctx, &model.TokenAccount{
				IDToken: "a2",
			})
			return nil
		})
		if e != nil {
			logger.Error("create token account failed", "err", e)
		}
		_ = e
		return errors.New("test error")
		//return nil
	})
	if err != nil {
		t.Fatal(err)
	}

}
