package service

import (
	"context"
	v1 "cpamgt/api/v1"
)

type CpaAccountServiceConfig = v1.CpaAccountServiceConfig

type CpaAccountService interface {
	GetCpaConfig(ctx context.Context) (*CpaAccountServiceConfig, error)
	SetCpaConfig(ctx context.Context, config *CpaAccountServiceConfig) error
}

type cpaAccountService struct {
	*Service
	config *CpaAccountServiceConfig
}

func NewCpaAccountService(svc *Service) CpaAccountService {
	return &cpaAccountService{Service: svc}
}

func (s *cpaAccountService) GetCpaConfig(ctx context.Context) (*CpaAccountServiceConfig, error) {
	if s.config == nil {
		return nil, v1.ErrNotFound
	}
	return s.config, nil
}

func (s *cpaAccountService) SetCpaConfig(ctx context.Context, config *CpaAccountServiceConfig) error {
	s.config = config
	return nil
}

func validateCpaConfig(config *CpaAccountServiceConfig) error {
	if config.CpaUrl == "" {
		return v1.ErrBadRequest
	}
	if config.CpaToken == "" {
		return v1.ErrBadRequest
	}
	return nil
}
