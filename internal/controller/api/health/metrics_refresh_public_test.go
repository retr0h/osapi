// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package health_test

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/controller/api/health"
	"github.com/osapi-io/osapi/internal/controller/api/health/gen"
)

type MetricsRefreshPublicTestSuite struct {
	suite.Suite

	ctx    context.Context
	logger *slog.Logger
}

func (s *MetricsRefreshPublicTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *MetricsRefreshPublicTestSuite) SetupSubTest() {
	s.SetupTest()
}

func (s *MetricsRefreshPublicTestSuite) newMetricsProvider(
	callCount *atomic.Int32,
) *health.ClosureMetricsProvider {
	return &health.ClosureMetricsProvider{
		NATSInfoFn: func(_ context.Context) (*health.NATSMetrics, error) {
			callCount.Add(1)

			return &health.NATSMetrics{URL: "nats://localhost:4222"}, nil
		},
		StreamInfoFn: func(_ context.Context) ([]health.StreamMetrics, error) {
			return nil, nil
		},
		KVInfoFn: func(_ context.Context) ([]health.KVMetrics, error) {
			return nil, nil
		},
		ObjectStoreInfoFn: func(_ context.Context) ([]health.ObjectStoreMetrics, error) {
			return nil, nil
		},
		JobStatsFn: func(_ context.Context) (*health.JobMetrics, error) {
			return nil, nil
		},
		AgentStatsFn: func(_ context.Context) (*health.AgentMetrics, error) {
			return nil, nil
		},
		ComponentRegistryFn: nil,
	}
}

func (s *MetricsRefreshPublicTestSuite) TestStartMetricsRefresh() {
	tests := []struct {
		name         string
		validateFunc func()
	}{
		{
			name: "when metrics is nil does not start goroutine",
			validateFunc: func() {
				handler := health.New(s.logger, nil, time.Now(), "0.1.0", nil, nil)
				// Should not panic or start goroutine.
				handler.StartMetricsRefresh(s.ctx)
			},
		},
		{
			name: "when metrics provided populates cache on initial fetch",
			validateFunc: func() {
				var callCount atomic.Int32
				metrics := s.newMetricsProvider(&callCount)

				ctx, cancel := context.WithCancel(s.ctx)
				defer cancel()

				handler := health.New(s.logger, nil, time.Now(), "0.1.0", metrics, nil)
				handler.StartMetricsRefresh(ctx)

				// Wait for initial fetch.
				time.Sleep(50 * time.Millisecond)
				s.GreaterOrEqual(callCount.Load(), int32(1))

				// Verify cache is populated via GetHealthStatus.
				resp, err := handler.GetHealthStatus(ctx, gen.GetHealthStatusRequestObject{})
				s.NoError(err)
				s.NotNil(resp)
			},
		},
		{
			name: "when ticker fires refreshes cache",
			validateFunc: func() {
				var callCount atomic.Int32
				metrics := s.newMetricsProvider(&callCount)

				ctx, cancel := context.WithCancel(s.ctx)
				defer cancel()

				handler := health.New(s.logger, nil, time.Now(), "0.1.0", metrics, nil)
				handler.MetricsRefreshInterval = 20 * time.Millisecond
				handler.StartMetricsRefresh(ctx)

				// Wait for initial + at least one ticker refresh.
				time.Sleep(100 * time.Millisecond)
				s.GreaterOrEqual(callCount.Load(), int32(2))
			},
		},
		{
			name: "when context cancelled stops refresh",
			validateFunc: func() {
				var callCount atomic.Int32
				metrics := s.newMetricsProvider(&callCount)

				ctx, cancel := context.WithCancel(s.ctx)
				handler := health.New(s.logger, nil, time.Now(), "0.1.0", metrics, nil)
				handler.MetricsRefreshInterval = 20 * time.Millisecond
				handler.StartMetricsRefresh(ctx)

				// Wait for goroutine to enter the select loop.
				time.Sleep(100 * time.Millisecond)
				cancel()
				time.Sleep(50 * time.Millisecond)

				// Record count after cancel.
				countAfterCancel := callCount.Load()
				time.Sleep(100 * time.Millisecond)

				// No more calls after cancel.
				s.Equal(countAfterCancel, callCount.Load())
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc()
		})
	}
}

func TestMetricsRefreshPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MetricsRefreshPublicTestSuite))
}
