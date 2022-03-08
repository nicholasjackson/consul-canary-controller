package mocks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/stretchr/testify/mock"
)

type MonitorMock struct {
	mock.Mock
}

func (r *MonitorMock) Configure(c json.RawMessage) error {
	args := r.Called(c)

	return args.Error(0)
}

func (r *MonitorMock) Check(ctx context.Context, interval time.Duration) error {
	args := r.Called(ctx, interval)

	return args.Error(0)
}
