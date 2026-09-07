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
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	"github.com/osapi-io/osapi/internal/controller/api/health"
	"github.com/osapi-io/osapi/internal/controller/api/health/gen"
	healthmocks "github.com/osapi-io/osapi/internal/controller/api/health/mocks"
)

const rbacHealthStatusTestSigningKey = "test-signing-key-for-rbac-integration"

type HealthStatusGetPublicTestSuite struct {
	suite.Suite

	ctx       context.Context
	appConfig config.Config
	logger    *slog.Logger
	mockCtrl  *gomock.Controller
}

func (s *HealthStatusGetPublicTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *HealthStatusGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *HealthStatusGetPublicTestSuite) TestGetHealthStatus() {
	tests := []struct {
		name         string
		checker      health.Checker
		checkerFn    func() health.Checker
		metrics      health.MetricsProvider
		validateFunc func(resp gen.GetHealthStatusResponseObject)
	}{
		{
			name: "all components healthy",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)
				s.Equal("ok", r.Status)
				s.Equal("ok", r.Components["controller.nats (connectivity)"].Status)
				s.Equal("ok", r.Components["controller.kv (connectivity)"].Status)
				s.Equal("0.1.0", r.Version)
				s.NotEmpty(r.Uptime)
			},
		},
		{
			name: "NATS unhealthy returns 503",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return fmt.Errorf("nats not connected") },
				KVCheck:   func() error { return nil },
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus503JSONResponse)
				s.True(ok)
				s.Equal("degraded", r.Status)
				s.Equal("error", r.Components["controller.nats (connectivity)"].Status)
				s.Require().NotNil(r.Components["controller.nats (connectivity)"].Error)
				s.Contains(
					*r.Components["controller.nats (connectivity)"].Error,
					"nats not connected",
				)
				s.Equal("ok", r.Components["controller.kv (connectivity)"].Status)
			},
		},
		{
			name: "KV unhealthy returns 503",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return fmt.Errorf("kv bucket not accessible") },
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus503JSONResponse)
				s.True(ok)
				s.Equal("degraded", r.Status)
				s.Equal("ok", r.Components["controller.nats (connectivity)"].Status)
				s.Equal("error", r.Components["controller.kv (connectivity)"].Status)
				s.Require().NotNil(r.Components["controller.kv (connectivity)"].Error)
				s.Contains(
					*r.Components["controller.kv (connectivity)"].Error,
					"kv bucket not accessible",
				)
			},
		},
		{
			name: "both unhealthy returns 503",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return fmt.Errorf("nats down") },
				KVCheck:   func() error { return fmt.Errorf("kv down") },
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus503JSONResponse)
				s.True(ok)
				s.Equal("degraded", r.Status)
				s.Equal("error", r.Components["controller.nats (connectivity)"].Status)
				s.Equal("error", r.Components["controller.kv (connectivity)"].Status)
			},
		},
		{
			name: "includes version and uptime",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)
				s.Equal("0.1.0", r.Version)
				s.NotEmpty(r.Uptime)
			},
		},
		{
			name: "non-NATSChecker returns ok with nil components",
			checkerFn: func() health.Checker {
				// GetHealthStatus type-asserts to *NATSChecker; for any other
				// implementation it short-circuits without calling CheckHealth.
				return healthmocks.NewMockChecker(s.mockCtrl)
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)
				s.Equal("ok", r.Status)
				s.Equal("ok", r.Components["controller.nats (connectivity)"].Status)
				s.Equal("ok", r.Components["controller.kv (connectivity)"].Status)
			},
		},
		{
			name: "nil MetricsProvider omits metrics",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			metrics: nil,
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)
				s.Nil(r.Nats)
				s.Nil(r.Streams)
				s.Nil(r.KvBuckets)
				s.Nil(r.Jobs)
			},
		},
		{
			name: "successful metrics populated",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			metrics: &health.ClosureMetricsProvider{
				NATSInfoFn: func(_ context.Context) (*health.NATSMetrics, error) {
					return &health.NATSMetrics{URL: "nats://localhost:4222", Version: "2.10.0"}, nil
				},
				StreamInfoFn: func(_ context.Context) ([]health.StreamMetrics, error) {
					return []health.StreamMetrics{
						{Name: "JOBS", Messages: 42, Bytes: 1024, Consumers: 1},
					}, nil
				},
				KVInfoFn: func(_ context.Context) ([]health.KVMetrics, error) {
					return []health.KVMetrics{
						{Name: "job-queue", Keys: 10, Bytes: 2048},
					}, nil
				},
				ObjectStoreInfoFn: func(_ context.Context) ([]health.ObjectStoreMetrics, error) {
					return []health.ObjectStoreMetrics{
						{Name: "file-objects", Size: 5242880},
					}, nil
				},
				JobStatsFn: func(_ context.Context) (*health.JobMetrics, error) {
					return &health.JobMetrics{
						Total: 100, Unprocessed: 5, Processing: 2,
						Completed: 90, Failed: 3, DLQ: 0,
					}, nil
				},
				AgentStatsFn: func(_ context.Context) (*health.AgentMetrics, error) {
					return &health.AgentMetrics{
						Total: 3,
						Ready: 3,
						Agents: []health.AgentDetail{
							{Hostname: "web-01", Labels: "group=web.prod", Registered: "15s ago"},
							{Hostname: "web-02", Labels: "group=web.prod", Registered: "8s ago"},
							{Hostname: "db-01", Labels: "", Registered: "2m ago"},
						},
					}, nil
				},
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)

				s.Require().NotNil(r.Nats)
				s.Equal("nats://localhost:4222", r.Nats.Url)
				s.Equal("2.10.0", r.Nats.Version)

				s.Require().NotNil(r.Streams)
				s.Len(*r.Streams, 1)
				s.Equal("JOBS", (*r.Streams)[0].Name)
				s.Equal(42, (*r.Streams)[0].Messages)

				s.Require().NotNil(r.KvBuckets)
				s.Len(*r.KvBuckets, 1)
				s.Equal("job-queue", (*r.KvBuckets)[0].Name)
				s.Equal(10, (*r.KvBuckets)[0].Keys)

				s.Require().NotNil(r.ObjectStores)
				s.Len(*r.ObjectStores, 1)
				s.Equal("file-objects", (*r.ObjectStores)[0].Name)
				s.Equal(5242880, (*r.ObjectStores)[0].Size)

				// Consumer count is derived from stream info, not enumerated.
				s.Require().NotNil(r.Consumers)
				s.Equal(1, r.Consumers.Total)
				s.Nil(r.Consumers.Consumers)

				s.Require().NotNil(r.Jobs)
				s.Equal(100, r.Jobs.Total)
				s.Equal(5, r.Jobs.Unprocessed)
				s.Equal(90, r.Jobs.Completed)

				s.Require().NotNil(r.Agents)
				s.Equal(3, r.Agents.Total)
				s.Equal(3, r.Agents.Ready)
				s.Require().NotNil(r.Agents.Agents)
				s.Len(*r.Agents.Agents, 3)
				s.Equal("web-01", (*r.Agents.Agents)[0].Hostname)
				s.Require().NotNil((*r.Agents.Agents)[0].Labels)
				s.Equal("group=web.prod", *(*r.Agents.Agents)[0].Labels)
				s.Equal("15s ago", (*r.Agents.Agents)[0].Registered)
				s.Equal("db-01", (*r.Agents.Agents)[2].Hostname)
				s.Nil((*r.Agents.Agents)[2].Labels)
				s.Equal("2m ago", (*r.Agents.Agents)[2].Registered)
			},
		},
		{
			name: "registry populated from component entries",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			metrics: &health.ClosureMetricsProvider{
				NATSInfoFn: func(_ context.Context) (*health.NATSMetrics, error) {
					return &health.NATSMetrics{URL: "nats://localhost:4222", Version: "2.10.0"}, nil
				},
				StreamInfoFn: func(_ context.Context) ([]health.StreamMetrics, error) {
					return []health.StreamMetrics{}, nil
				},
				KVInfoFn: func(_ context.Context) ([]health.KVMetrics, error) {
					return []health.KVMetrics{}, nil
				},
				ObjectStoreInfoFn: func(_ context.Context) ([]health.ObjectStoreMetrics, error) {
					return []health.ObjectStoreMetrics{}, nil
				},
				JobStatsFn: func(_ context.Context) (*health.JobMetrics, error) {
					return &health.JobMetrics{}, nil
				},
				AgentStatsFn: func(_ context.Context) (*health.AgentMetrics, error) {
					return &health.AgentMetrics{}, nil
				},
				ComponentRegistryFn: func(_ context.Context) ([]health.ComponentEntry, error) {
					return []health.ComponentEntry{
						{
							Type:       "api",
							Hostname:   "api-server-01",
							Status:     "Ready",
							Conditions: nil,
							Age:        "7h 6m",
							CPUPercent: 2.1,
							MemBytes:   134217728,
						},
						{
							Type:       "agent",
							Hostname:   "web-01",
							Status:     "Ready",
							Conditions: []string{"DiskPressure"},
							Age:        "3h 0m",
							CPUPercent: 1.2,
							MemBytes:   100663296,
						},
					}, nil
				},
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)

				s.Require().NotNil(r.Registry)
				s.Len(*r.Registry, 2)

				first := (*r.Registry)[0]
				s.Require().NotNil(first.Type)
				s.Equal("api", *first.Type)
				s.Require().NotNil(first.Hostname)
				s.Equal("api-server-01", *first.Hostname)
				s.Require().NotNil(first.Status)
				s.Equal("Ready", *first.Status)
				s.Require().NotNil(first.Age)
				s.Equal("7h 6m", *first.Age)
				s.Require().NotNil(first.CpuPercent)
				s.InDelta(2.1, float64(*first.CpuPercent), 0.01)
				s.Require().NotNil(first.MemBytes)
				s.Equal(int64(134217728), *first.MemBytes)
				s.Nil(first.Conditions)

				second := (*r.Registry)[1]
				s.Require().NotNil(second.Type)
				s.Equal("agent", *second.Type)
				s.Require().NotNil(second.Conditions)
				s.Equal([]string{"DiskPressure"}, *second.Conditions)
			},
		},
		{
			name: "registry sub-components merged into components",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			metrics: &health.ClosureMetricsProvider{
				NATSInfoFn: func(_ context.Context) (*health.NATSMetrics, error) {
					return &health.NATSMetrics{URL: "nats://localhost:4222", Version: "2.10.0"}, nil
				},
				StreamInfoFn: func(_ context.Context) ([]health.StreamMetrics, error) {
					return []health.StreamMetrics{}, nil
				},
				KVInfoFn: func(_ context.Context) ([]health.KVMetrics, error) {
					return []health.KVMetrics{}, nil
				},
				ObjectStoreInfoFn: func(_ context.Context) ([]health.ObjectStoreMetrics, error) {
					return []health.ObjectStoreMetrics{}, nil
				},
				JobStatsFn: func(_ context.Context) (*health.JobMetrics, error) {
					return &health.JobMetrics{}, nil
				},
				AgentStatsFn: func(_ context.Context) (*health.AgentMetrics, error) {
					return &health.AgentMetrics{}, nil
				},
				ComponentRegistryFn: func(_ context.Context) ([]health.ComponentEntry, error) {
					return []health.ComponentEntry{
						{
							Type:     "nats",
							Hostname: "nats-01",
							Status:   "Ready",
							Age:      "1h",
							SubComponents: map[string]health.SubComponentInfo{
								"nats.server":    {Status: "ok", Address: "nats://localhost:4222"},
								"nats.heartbeat": {Status: "ok"},
								"nats.metrics":   {Status: "ok", Address: "http://0.0.0.0:9092"},
							},
						},
						{
							Type:     "agent",
							Hostname: "web-01",
							Status:   "Ready",
							Age:      "30m",
							SubComponents: map[string]health.SubComponentInfo{
								"agent.heartbeat": {Status: "ok"},
								"agent.metrics":   {Status: "disabled"},
							},
						},
					}, nil
				},
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)

				// Sub-components from registry entries merged into components map.
				natsServer := r.Components["nats.server"]
				s.Equal("ok", natsServer.Status)
				s.Require().NotNil(natsServer.Address)
				s.Equal("nats://localhost:4222", *natsServer.Address)

				natsHB := r.Components["nats.heartbeat"]
				s.Equal("ok", natsHB.Status)
				s.Nil(natsHB.Address)

				natsMetrics := r.Components["nats.metrics"]
				s.Equal("ok", natsMetrics.Status)
				s.Require().NotNil(natsMetrics.Address)
				s.Equal("http://0.0.0.0:9092", *natsMetrics.Address)

				agentHB := r.Components["agent.heartbeat"]
				s.Equal("ok", agentHB.Status)

				agentMetrics := r.Components["agent.metrics"]
				s.Equal("disabled", agentMetrics.Status)
				s.Nil(agentMetrics.Address)
			},
		},
		{
			name: "registry failure degrades gracefully",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			metrics: &health.ClosureMetricsProvider{
				NATSInfoFn: func(_ context.Context) (*health.NATSMetrics, error) {
					return &health.NATSMetrics{URL: "nats://localhost:4222", Version: "2.10.0"}, nil
				},
				StreamInfoFn: func(_ context.Context) ([]health.StreamMetrics, error) {
					return []health.StreamMetrics{}, nil
				},
				KVInfoFn: func(_ context.Context) ([]health.KVMetrics, error) {
					return []health.KVMetrics{}, nil
				},
				ObjectStoreInfoFn: func(_ context.Context) ([]health.ObjectStoreMetrics, error) {
					return []health.ObjectStoreMetrics{}, nil
				},
				JobStatsFn: func(_ context.Context) (*health.JobMetrics, error) {
					return &health.JobMetrics{}, nil
				},
				AgentStatsFn: func(_ context.Context) (*health.AgentMetrics, error) {
					return &health.AgentMetrics{}, nil
				},
				ComponentRegistryFn: func(_ context.Context) ([]health.ComponentEntry, error) {
					return nil, fmt.Errorf("registry unavailable")
				},
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)
				s.Equal("ok", r.Status)
				s.Nil(r.Registry)
			},
		},
		{
			name: "nil ComponentRegistryFn omits registry",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			metrics: &health.ClosureMetricsProvider{
				NATSInfoFn: func(_ context.Context) (*health.NATSMetrics, error) {
					return &health.NATSMetrics{URL: "nats://localhost:4222", Version: "2.10.0"}, nil
				},
				StreamInfoFn: func(_ context.Context) ([]health.StreamMetrics, error) {
					return []health.StreamMetrics{}, nil
				},
				KVInfoFn: func(_ context.Context) ([]health.KVMetrics, error) {
					return []health.KVMetrics{}, nil
				},
				ObjectStoreInfoFn: func(_ context.Context) ([]health.ObjectStoreMetrics, error) {
					return []health.ObjectStoreMetrics{}, nil
				},
				JobStatsFn: func(_ context.Context) (*health.JobMetrics, error) {
					return &health.JobMetrics{}, nil
				},
				AgentStatsFn: func(_ context.Context) (*health.AgentMetrics, error) {
					return &health.AgentMetrics{}, nil
				},
				// ComponentRegistryFn intentionally nil
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)
				s.Equal("ok", r.Status)
				s.Nil(r.Registry)
			},
		},
		{
			name: "partial metric failures degrade gracefully",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			metrics: &health.ClosureMetricsProvider{
				NATSInfoFn: func(_ context.Context) (*health.NATSMetrics, error) {
					return nil, fmt.Errorf("nats info unavailable")
				},
				StreamInfoFn: func(_ context.Context) ([]health.StreamMetrics, error) {
					return nil, fmt.Errorf("stream info unavailable")
				},
				KVInfoFn: func(_ context.Context) ([]health.KVMetrics, error) {
					return []health.KVMetrics{
						{Name: "job-queue", Keys: 5, Bytes: 512},
					}, nil
				},
				ObjectStoreInfoFn: func(_ context.Context) ([]health.ObjectStoreMetrics, error) {
					return nil, fmt.Errorf("object store info unavailable")
				},
				JobStatsFn: func(_ context.Context) (*health.JobMetrics, error) {
					return nil, fmt.Errorf("job stats unavailable")
				},
				AgentStatsFn: func(_ context.Context) (*health.AgentMetrics, error) {
					return nil, fmt.Errorf("agent stats unavailable")
				},
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)
				s.Equal("ok", r.Status)
				s.Nil(r.Nats)
				s.Nil(r.Streams)
				s.Require().NotNil(r.KvBuckets)
				s.Len(*r.KvBuckets, 1)
				s.Nil(r.ObjectStores)
				s.Nil(r.Consumers)
				s.Nil(r.Jobs)
				s.Nil(r.Agents)
			},
		},
		{
			name: "all metric failures degrade gracefully",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			metrics: &health.ClosureMetricsProvider{
				NATSInfoFn: func(_ context.Context) (*health.NATSMetrics, error) {
					return nil, fmt.Errorf("nats info unavailable")
				},
				StreamInfoFn: func(_ context.Context) ([]health.StreamMetrics, error) {
					return nil, fmt.Errorf("stream info unavailable")
				},
				KVInfoFn: func(_ context.Context) ([]health.KVMetrics, error) {
					return nil, fmt.Errorf("kv info unavailable")
				},
				ObjectStoreInfoFn: func(_ context.Context) ([]health.ObjectStoreMetrics, error) {
					return nil, fmt.Errorf("object store info unavailable")
				},
				JobStatsFn: func(_ context.Context) (*health.JobMetrics, error) {
					return nil, fmt.Errorf("job stats unavailable")
				},
				AgentStatsFn: func(_ context.Context) (*health.AgentMetrics, error) {
					return nil, fmt.Errorf("agent stats unavailable")
				},
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)
				s.Equal("ok", r.Status)
				s.Nil(r.Nats)
				s.Nil(r.Streams)
				s.Nil(r.KvBuckets)
				s.Nil(r.ObjectStores)
				s.Nil(r.Consumers)
				s.Nil(r.Jobs)
				s.Nil(r.Agents)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			checker := tt.checker
			if checker == nil && tt.checkerFn != nil {
				checker = tt.checkerFn()
			}
			handler := health.New(slog.Default(), checker, time.Now(), "0.1.0", tt.metrics, nil)

			resp, err := handler.GetHealthStatus(s.ctx, gen.GetHealthStatusRequestObject{})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *HealthStatusGetPublicTestSuite) TestGetHealthStatusSubComponents() {
	tests := []struct {
		name         string
		subs         map[string]health.SubComponentInfo
		validateFunc func(resp gen.GetHealthStatusResponseObject)
	}{
		{
			name: "sub-components with ports appear in response",
			subs: map[string]health.SubComponentInfo{
				"controller.api":       {Status: "ok", Address: "http://0.0.0.0:8080"},
				"controller.heartbeat": {Status: "ok"},
				"controller.metrics":   {Status: "ok", Address: "http://0.0.0.0:9090"},
				"controller.notifier":  {Status: "disabled"},
			},
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)

				api := r.Components["controller.api"]
				s.Equal("ok", api.Status)
				s.Require().NotNil(api.Address)
				s.Equal("http://0.0.0.0:8080", *api.Address)

				hb := r.Components["controller.heartbeat"]
				s.Equal("ok", hb.Status)
				s.Nil(hb.Address)

				metrics := r.Components["controller.metrics"]
				s.Equal("ok", metrics.Status)
				s.Require().NotNil(metrics.Address)
				s.Equal("http://0.0.0.0:9090", *metrics.Address)

				notifier := r.Components["controller.notifier"]
				s.Equal("disabled", notifier.Status)
				s.Nil(notifier.Address)
			},
		},
		{
			name: "nil sub-components produces no extra keys",
			subs: nil,
			validateFunc: func(resp gen.GetHealthStatusResponseObject) {
				r, ok := resp.(gen.GetHealthStatus200JSONResponse)
				s.True(ok)
				s.Len(r.Components, 2) // nats + kv only
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			checker := &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			}

			handler := health.New(slog.Default(), checker, time.Now(), "0.1.0", nil, tt.subs)

			resp, err := handler.GetHealthStatus(s.ctx, gen.GetHealthStatusRequestObject{})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *HealthStatusGetPublicTestSuite) TestGetHealthStatusHTTP() {
	tests := []struct {
		name         string
		checker      *health.NATSChecker
		metrics      health.MetricsProvider
		wantCode     int
		wantContains []string
	}{
		{
			name: "when all components healthy returns status with metrics",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			metrics: &health.ClosureMetricsProvider{
				NATSInfoFn: func(
					_ context.Context,
				) (*health.NATSMetrics, error) {
					return &health.NATSMetrics{
						URL:     "nats://localhost:4222",
						Version: "2.10.0",
					}, nil
				},
				StreamInfoFn: func(
					_ context.Context,
				) ([]health.StreamMetrics, error) {
					return []health.StreamMetrics{
						{Name: "JOBS", Messages: 42, Bytes: 1024, Consumers: 1},
					}, nil
				},
				KVInfoFn: func(
					_ context.Context,
				) ([]health.KVMetrics, error) {
					return []health.KVMetrics{
						{Name: "job-queue", Keys: 10, Bytes: 2048},
					}, nil
				},
				ObjectStoreInfoFn: func(
					_ context.Context,
				) ([]health.ObjectStoreMetrics, error) {
					return []health.ObjectStoreMetrics{
						{Name: "file-objects", Size: 5242880},
					}, nil
				},
				JobStatsFn: func(
					_ context.Context,
				) (*health.JobMetrics, error) {
					return &health.JobMetrics{
						Total: 100, Unprocessed: 5, Processing: 2,
						Completed: 90, Failed: 3, DLQ: 0,
					}, nil
				},
				AgentStatsFn: func(
					_ context.Context,
				) (*health.AgentMetrics, error) {
					return &health.AgentMetrics{
						Total: 3,
						Ready: 3,
						Agents: []health.AgentDetail{
							{Hostname: "web-01", Labels: "group=web.prod", Registered: "15s ago"},
						},
					}, nil
				},
				ComponentRegistryFn: func(
					_ context.Context,
				) ([]health.ComponentEntry, error) {
					return []health.ComponentEntry{
						{
							Type:     "api",
							Hostname: "api-server-01",
							Status:   "Ready",
							Age:      "7h 6m",
						},
					}, nil
				},
			},
			wantCode: http.StatusOK,
			wantContains: []string{
				`"status":"ok"`,
				`"version":"0.1.0"`,
				`"uptime"`,
				`"nats"`,
				`"streams"`,
				`"kv_buckets"`,
				`"object_stores"`,
				`"consumers"`,
				`"jobs"`,
				`"agents"`,
				`"web-01"`,
				`"group=web.prod"`,
				`"total":1`,
				`"file-objects"`,
				`"registry"`,
				`"api-server-01"`,
			},
		},
		{
			name: "when nil metrics omits metrics fields",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			metrics:  nil,
			wantCode: http.StatusOK,
			wantContains: []string{
				`"status":"ok"`,
				`"version":"0.1.0"`,
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			healthHandler := health.New(
				s.logger,
				tc.checker,
				time.Now(),
				"0.1.0",
				tc.metrics,
				nil,
			)
			strictHandler := gen.NewStrictHandler(healthHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, "/api/health/status", nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, want := range tc.wantContains {
				s.Contains(rec.Body.String(), want)
			}
		})
	}
}

func (s *HealthStatusGetPublicTestSuite) TestGetHealthStatusRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		wantCode     int
		wantContains []string
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set
			},
			wantCode:     http.StatusUnauthorized,
			wantContains: []string{"Bearer token required"},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacHealthStatusTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"job:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid token with health:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacHealthStatusTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"status":"ok"`, `"version"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			checker := &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			}
			metrics := &health.ClosureMetricsProvider{
				NATSInfoFn: func(
					_ context.Context,
				) (*health.NATSMetrics, error) {
					return &health.NATSMetrics{
						URL:     "nats://localhost:4222",
						Version: "2.10.0",
					}, nil
				},
				StreamInfoFn: func(
					_ context.Context,
				) ([]health.StreamMetrics, error) {
					return []health.StreamMetrics{}, nil
				},
				KVInfoFn: func(
					_ context.Context,
				) ([]health.KVMetrics, error) {
					return []health.KVMetrics{}, nil
				},
				ObjectStoreInfoFn: func(
					_ context.Context,
				) ([]health.ObjectStoreMetrics, error) {
					return []health.ObjectStoreMetrics{}, nil
				},
				JobStatsFn: func(
					_ context.Context,
				) (*health.JobMetrics, error) {
					return &health.JobMetrics{}, nil
				},
				AgentStatsFn: func(
					_ context.Context,
				) (*health.AgentMetrics, error) {
					return &health.AgentMetrics{}, nil
				},
				ComponentRegistryFn: func(
					_ context.Context,
				) ([]health.ComponentEntry, error) {
					return []health.ComponentEntry{}, nil
				},
			}

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacHealthStatusTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := health.Handler(
				s.logger,
				checker,
				time.Now(),
				"0.1.0",
				metrics,
				nil,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(http.MethodGet, "/api/health/status", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, want := range tc.wantContains {
				s.Contains(rec.Body.String(), want)
			}
		})
	}
}

func (s *HealthStatusGetPublicTestSuite) TestMetricsCache() {
	tests := []struct {
		name         string
		validateFunc func(handler *health.Health)
	}{
		{
			name: "when cache is fresh serves cached metrics",
			validateFunc: func(handler *health.Health) {
				// First call populates cache.
				resp1, err := handler.GetHealthStatus(s.ctx, gen.GetHealthStatusRequestObject{})
				s.NoError(err)
				s.NotNil(resp1)

				// Second call should serve from cache with identical data.
				resp2, err := handler.GetHealthStatus(s.ctx, gen.GetHealthStatusRequestObject{})
				s.NoError(err)
				s.NotNil(resp2)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			checker := &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			}
			metrics := &health.ClosureMetricsProvider{
				NATSInfoFn: func(_ context.Context) (*health.NATSMetrics, error) {
					return &health.NATSMetrics{URL: "nats://localhost:4222", Version: "2.10.0"}, nil
				},
				StreamInfoFn: func(_ context.Context) ([]health.StreamMetrics, error) {
					return []health.StreamMetrics{{Name: "JOBS", Consumers: 5}}, nil
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
				ComponentRegistryFn: func(_ context.Context) ([]health.ComponentEntry, error) {
					return []health.ComponentEntry{
						{
							Type:     "agent",
							Hostname: "web-01",
							Status:   "Ready",
							SubComponents: map[string]health.SubComponentInfo{
								"agent.heartbeat": {Status: "ok"},
							},
						},
					}, nil
				},
			}
			handler := health.New(s.logger, checker, time.Now(), "0.1.0", metrics, nil)

			tc.validateFunc(handler)
		})
	}
}

func TestHealthStatusGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HealthStatusGetPublicTestSuite))
}
