package service

import (
	"context"
	"cpamgt/internal/pkg/log"
	"cpamgt/internal/repository"
)

type Service struct {
	logger *log.Logger
	//sid    *sid.Sid
	//jwt    *jwt.JWT
	tm repository.Transaction
}

func NewService(logger *log.Logger, tm repository.Transaction) *Service {
	return &Service{
		logger: logger,
		tm:     tm,
	}
}

// Log avoid using slog.Logger directly
func (s *Service) Log(ctx context.Context) *log.Logger {
	return s.logger.FromContext(ctx)
}
