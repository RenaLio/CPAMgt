package task

import (
	"context"
	"cpamgt/pkg/log"
	"time"
)

type MockTask struct {
	logger *log.Logger
}

func NewMockTask(logger *log.Logger) *MockTask {
	return &MockTask{logger: logger}
}

func (m *MockTask) Name() string {
	return MockTaskName
}

func (m *MockTask) Run(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	m.logger.Info("mock task runOnce")
	<-ctx.Done()
	return nil
}

//func (m *MockTask) Run(ctx context.Context) error {
//	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
//	defer cancel()
//	m.logger.Info("mock task runOnce")
//	select {
//	case <-ctx.Done():
//		return nil
//	}
//}
