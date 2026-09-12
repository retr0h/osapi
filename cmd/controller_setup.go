// Copyright (c) 2024 John Dewey

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

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/avfs/avfs/vfs/osfs"
	"github.com/labstack/echo/v5"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	natsclient "github.com/osapi-io/nats-client/pkg/client"

	agentPKI "github.com/osapi-io/osapi/internal/agent/pki"
	"github.com/osapi-io/osapi/internal/audit"
	"github.com/osapi-io/osapi/internal/cli"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller"
	"github.com/osapi-io/osapi/internal/controller/api"
	agentAPI "github.com/osapi-io/osapi/internal/controller/api/agent"
	auditAPI "github.com/osapi-io/osapi/internal/controller/api/audit"
	factsAPI "github.com/osapi-io/osapi/internal/controller/api/facts"
	"github.com/osapi-io/osapi/internal/controller/api/file"
	"github.com/osapi-io/osapi/internal/controller/api/health"
	jobAPI "github.com/osapi-io/osapi/internal/controller/api/job"
	nodeAPI "github.com/osapi-io/osapi/internal/controller/api/node"
	certificateAPI "github.com/osapi-io/osapi/internal/controller/api/node/certificate"
	commandAPI "github.com/osapi-io/osapi/internal/controller/api/node/command"
	dockerAPI "github.com/osapi-io/osapi/internal/controller/api/node/docker"
	nodeFileAPI "github.com/osapi-io/osapi/internal/controller/api/node/file"
	hostnameAPI "github.com/osapi-io/osapi/internal/controller/api/node/hostname"
	logAPI "github.com/osapi-io/osapi/internal/controller/api/node/log"
	networkAPI "github.com/osapi-io/osapi/internal/controller/api/node/network"
	ntpAPI "github.com/osapi-io/osapi/internal/controller/api/node/ntp"
	packageAPI "github.com/osapi-io/osapi/internal/controller/api/node/package"
	powerAPI "github.com/osapi-io/osapi/internal/controller/api/node/power"
	processAPI "github.com/osapi-io/osapi/internal/controller/api/node/process"
	scheduleAPI "github.com/osapi-io/osapi/internal/controller/api/node/schedule"
	serviceAPI "github.com/osapi-io/osapi/internal/controller/api/node/service"
	sysctlAPI "github.com/osapi-io/osapi/internal/controller/api/node/sysctl"
	timezoneAPI "github.com/osapi-io/osapi/internal/controller/api/node/timezone"
	userAPI "github.com/osapi-io/osapi/internal/controller/api/node/user"
	uihandler "github.com/osapi-io/osapi/internal/controller/api/ui"
	"github.com/osapi-io/osapi/internal/controller/enrollment"
	"github.com/osapi-io/osapi/internal/controller/notify"
	"github.com/osapi-io/osapi/internal/job"
	jobclient "github.com/osapi-io/osapi/internal/job/client"
	"github.com/osapi-io/osapi/internal/telemetry/metrics"
	"github.com/osapi-io/osapi/internal/telemetry/process"
	"github.com/osapi-io/osapi/internal/validation"
	uiassets "github.com/osapi-io/osapi/ui"
)

// ServerManager responsible for Server operations.
type ServerManager interface {
	cli.Lifecycle
	// RegisterHandlers registers a list of handlers with the Echo instance.
	RegisterHandlers(handlers []func(e *echo.Echo))
}

// NATSClient defines the NATS operations needed by cmd setup, runtime, and
// metrics functions. Each function that receives a NATSClient uses only a
// subset of these methods; the full interface is defined here so the
// natsBundle can store a single value that satisfies all consumers.
type NATSClient interface {
	// Connection management
	Connect() error
	Close()

	// Stream operations
	CreateOrUpdateStreamWithConfig(
		ctx context.Context,
		streamConfig jetstream.StreamConfig,
	) error
	GetStreamInfo(
		ctx context.Context,
		streamName string,
	) (*jetstream.StreamInfo, error)

	// KV operations
	CreateOrUpdateKVBucket(
		ctx context.Context,
		bucketName string,
	) (jetstream.KeyValue, error)
	CreateOrUpdateKVBucketWithConfig(
		ctx context.Context,
		config jetstream.KeyValueConfig,
	) (jetstream.KeyValue, error)

	// Object Store operations
	CreateOrUpdateObjectStore(
		ctx context.Context,
		cfg jetstream.ObjectStoreConfig,
	) (jetstream.ObjectStore, error)
	ObjectStore(
		ctx context.Context,
		name string,
	) (jetstream.ObjectStore, error)

	// Message publishing
	Publish(
		ctx context.Context,
		subject string,
		data []byte,
	) error

	// Message consumption
	ConsumeMessages(
		ctx context.Context,
		streamName string,
		consumerName string,
		handler natsclient.JetStreamMessageHandler,
		opts *natsclient.ConsumeOptions,
	) error
	CreateOrUpdateConsumerWithConfig(
		ctx context.Context,
		streamName string,
		consumerConfig jetstream.ConsumerConfig,
	) error

	// KV convenience operations
	KVPut(
		bucket string,
		key string,
		value []byte,
	) error

	// Core NATS operations (non-JetStream)
	Subscribe(
		subject string,
		handler nats.MsgHandler,
	) (*nats.Subscription, error)
	PublishCore(
		subject string,
		data []byte,
	) error

	// Connection inspection
	ConnectedURL() string
	ConnectedServerVersion() string

	// JetStream handle access
	KeyValue(
		ctx context.Context,
		bucket string,
	) (jetstream.KeyValue, error)
	Stream(
		ctx context.Context,
		name string,
	) (jetstream.Stream, error)
}

// natsBundle holds the NATS connection, job client, and KV handles created
// by connectNATSBundle.
type natsBundle struct {
	nc            NATSClient
	jobClient     jobclient.JobClient
	jobsKV        jetstream.KeyValue
	registryKV    jetstream.KeyValue
	factsKV       jetstream.KeyValue
	fileStateKV   jetstream.KeyValue
	objStore      file.ObjectStoreManager
	checker       health.Checker
	subComponents map[string]job.SubComponentInfo
}

// setupController connects to NATS, creates the API server with all handlers,
// and returns the server manager and NATS bundle. It is used by the standalone
// API server start and combined start commands. Extra api.Option values are
// appended after the audit-store option (e.g., WithMeterProvider).
func setupController(
	ctx context.Context,
	log *slog.Logger,
	connCfg config.NATSConnection,
	extraOpts ...api.Option,
) (ServerManager, *natsBundle) {
	namespace := connCfg.Namespace
	streamName := job.ApplyNamespaceToInfraName(namespace, appConfig.NATS.Stream.Name)

	b := connectNATSBundle(ctx, log, connCfg, namespace, streamName)

	validation.RegisterTargetValidator(
		func(ctx context.Context) ([]validation.AgentTarget, error) {
			agents, err := b.jobClient.ListAgents(ctx)
			if err != nil {
				return nil, err
			}
			targets := make([]validation.AgentTarget, 0, len(agents))
			for _, a := range agents {
				targets = append(targets, validation.AgentTarget{
					MachineID: a.MachineID,
					Hostname:  a.Hostname,
					State:     a.State,
					Labels:    a.Labels,
				})
			}
			return targets, nil
		},
	)

	checker := newHealthChecker(b.nc, b.jobsKV)
	b.checker = checker
	auditStore, serverOpts := createAuditStore(ctx, log, b.nc, namespace)
	kvBuckets := configuredKVBuckets(namespace)
	objBuckets := configuredObjectBuckets(namespace)
	metricsProvider := newMetricsProvider(
		b.nc,
		streamName,
		kvBuckets,
		objBuckets,
		b.jobClient,
		b.registryKV,
	)

	// --- PKI enrollment watcher (conditional) ---
	var enrollmentWatcher *enrollment.Watcher
	if appConfig.Controller.PKI.Enabled {
		enrollmentWatcher = setupEnrollmentWatcher(ctx, log, b, namespace)
	}

	serverOpts = append(serverOpts, extraOpts...)
	sm := api.New(appConfig, log, serverOpts...)
	registerControllerHandlers(
		log, sm, b.jobClient, checker, metricsProvider,
		auditStore, b.objStore, b.fileStateKV, enrollmentWatcher,
	)

	b.subComponents = buildControllerSubComponents()
	startControllerHeartbeat(ctx, log, b.registryKV, b.subComponents)
	startConditionWatcher(ctx, log, b.registryKV)

	return sm, b
}

func connectNATSBundle(
	ctx context.Context,
	log *slog.Logger,
	connCfg config.NATSConnection,
	namespace string,
	streamName string,
) *natsBundle {
	concreteNC := natsclient.New(log, &natsclient.Options{
		Host: connCfg.Host,
		Port: connCfg.Port,
		Auth: cli.BuildNATSAuthOptions(connCfg.Auth),
		Name: connCfg.ClientName,
	})

	if err := concreteNC.Connect(); err != nil {
		cli.LogFatal(log, "failed to connect to NATS", err)
	}

	var nc NATSClient = concreteNC

	jobKVConfig := cli.BuildJobKVConfig(namespace, appConfig.NATS.KV)
	jobsKV, err := nc.CreateOrUpdateKVBucketWithConfig(ctx, jobKVConfig)
	if err != nil {
		cli.LogFatal(log, "failed to create KV bucket", err)
	}

	registryKVConfig := cli.BuildRegistryKVConfig(namespace, appConfig.NATS.Registry)
	registryKV, err := nc.CreateOrUpdateKVBucketWithConfig(ctx, registryKVConfig)
	if err != nil {
		cli.LogFatal(log, "failed to create registry KV bucket", err)
	}

	var factsKV jetstream.KeyValue
	if appConfig.NATS.Facts.Bucket != "" {
		factsKVConfig := cli.BuildFactsKVConfig(namespace, appConfig.NATS.Facts)
		factsKV, err = nc.CreateOrUpdateKVBucketWithConfig(ctx, factsKVConfig)
		if err != nil {
			cli.LogFatal(log, "failed to create facts KV bucket", err)
		}
	}

	var stateKV jetstream.KeyValue
	if appConfig.NATS.State.Bucket != "" {
		stateKVConfig := cli.BuildStateKVConfig(namespace, appConfig.NATS.State)
		stateKV, err = nc.CreateOrUpdateKVBucketWithConfig(ctx, stateKVConfig)
		if err != nil {
			cli.LogFatal(log, "failed to create state KV bucket", err)
		}
	}

	// Create or update Object Store bucket for file management API
	var objStore file.ObjectStoreManager
	if appConfig.NATS.Objects.Bucket != "" {
		objStoreConfig := cli.BuildObjectStoreConfig(namespace, appConfig.NATS.Objects)
		var objErr error
		objStore, objErr = nc.CreateOrUpdateObjectStore(ctx, objStoreConfig)
		if objErr != nil {
			log.Warn("Object Store not available, file endpoints disabled",
				slog.String("bucket", objStoreConfig.Bucket),
				slog.String("error", objErr.Error()))
		}
	}

	// Look up the file-state KV bucket for stale deployment checks.
	var fileStateKV jetstream.KeyValue
	if appConfig.NATS.FileState.Bucket != "" {
		fileStateBucket := job.ApplyNamespaceToInfraName(
			namespace,
			appConfig.NATS.FileState.Bucket,
		)
		var kvErr error
		fileStateKV, kvErr = nc.KeyValue(ctx, fileStateBucket)
		if kvErr != nil {
			log.Warn("file state KV not available",
				slog.String("error", kvErr.Error()))
		}
	}

	jobTimeout, _ := time.ParseDuration(appConfig.Controller.API.JobTimeout)

	jc, err := jobclient.New(log, nc, &jobclient.Options{
		Timeout:        jobTimeout,
		KVBucket:       jobsKV,
		RegistryKV:     registryKV,
		FactsKV:        factsKV,
		StateKV:        stateKV,
		StreamName:     streamName,
		TargetResolver: validation.ResolveTarget,
	})
	if err != nil {
		cli.LogFatal(log, "failed to create job client", err)
	}

	return &natsBundle{
		nc:          nc,
		jobClient:   jc,
		jobsKV:      jobsKV,
		registryKV:  registryKV,
		factsKV:     factsKV,
		fileStateKV: fileStateKV,
		objStore:    objStore,
	}
}

func newHealthChecker(
	nc NATSClient,
	jobsKV jetstream.KeyValue,
) *health.NATSChecker {
	return &health.NATSChecker{
		NATSCheck: func() error {
			if nc.ConnectedURL() == "" {
				return fmt.Errorf("nats not connected")
			}

			return nil
		},
		KVCheck: func() error {
			_, err := jobsKV.Status(context.Background())
			if err != nil {
				return fmt.Errorf("kv bucket not accessible: %w", err)
			}

			return nil
		},
	}
}

// configuredKVBuckets returns the namespaced names of all KV buckets
// declared in osapi.yaml. Only non-empty bucket configs are included.
func configuredKVBuckets(
	namespace string,
) []string {
	var buckets []string
	for _, info := range appConfig.NATS.AllKVBuckets() {
		if info.Bucket != "" {
			buckets = append(buckets, job.ApplyNamespaceToInfraName(namespace, info.Bucket))
		}
	}

	return buckets
}

// configuredObjectBuckets returns the namespaced names of all Object Store
// buckets declared in osapi.yaml.
func configuredObjectBuckets(
	namespace string,
) []string {
	var buckets []string
	for _, info := range appConfig.NATS.AllObjectStoreBuckets() {
		if info.Bucket != "" {
			buckets = append(
				buckets,
				job.ApplyNamespaceToInfraName(namespace, info.Bucket),
			)
		}
	}

	return buckets
}

func newMetricsProvider(
	nc NATSClient,
	streamName string,
	kvBuckets []string,
	objBuckets []string,
	jc jobclient.JobClient,
	registryKV jetstream.KeyValue,
) *health.ClosureMetricsProvider {
	return &health.ClosureMetricsProvider{
		NATSInfoFn: func(_ context.Context) (*health.NATSMetrics, error) {
			url := nc.ConnectedURL()
			if url == "" {
				return nil, fmt.Errorf("NATS client unavailable")
			}

			return &health.NATSMetrics{
				URL:     url,
				Version: nc.ConnectedServerVersion(),
			}, nil
		},
		StreamInfoFn: func(fnCtx context.Context) ([]health.StreamMetrics, error) {
			info, err := nc.GetStreamInfo(fnCtx, streamName)
			if err != nil {
				return nil, fmt.Errorf("stream info: %w", err)
			}

			return []health.StreamMetrics{
				{
					Name:      streamName,
					Messages:  info.State.Msgs,
					Bytes:     info.State.Bytes,
					Consumers: info.State.Consumers,
				},
			}, nil
		},
		KVInfoFn: func(fnCtx context.Context) ([]health.KVMetrics, error) {
			results := make([]health.KVMetrics, 0, len(kvBuckets))
			for _, name := range kvBuckets {
				kv, err := nc.KeyValue(fnCtx, name)
				if err != nil {
					continue
				}
				status, err := kv.Status(fnCtx)
				if err != nil {
					continue
				}
				results = append(results, health.KVMetrics{
					Name:  status.Bucket(),
					Keys:  int(status.Values()),
					Bytes: status.Bytes(),
				})
			}

			return results, nil
		},
		ObjectStoreInfoFn: func(fnCtx context.Context) ([]health.ObjectStoreMetrics, error) {
			results := make([]health.ObjectStoreMetrics, 0, len(objBuckets))
			for _, name := range objBuckets {
				obj, err := nc.ObjectStore(fnCtx, name)
				if err != nil {
					continue
				}
				status, err := obj.Status(fnCtx)
				if err != nil {
					continue
				}
				results = append(results, health.ObjectStoreMetrics{
					Name: status.Bucket(),
					Size: status.Size(),
				})
			}

			return results, nil
		},
		JobStatsFn: func(fnCtx context.Context) (*health.JobMetrics, error) {
			stats, err := jc.GetQueueSummary(fnCtx)
			if err != nil {
				return nil, fmt.Errorf("job stats: %w", err)
			}

			return &health.JobMetrics{
				Total:       stats.TotalJobs,
				Unprocessed: stats.StatusCounts["submitted"],
				Processing:  stats.StatusCounts["processing"],
				Completed:   stats.StatusCounts["completed"],
				Failed:      stats.StatusCounts["failed"],
				DLQ:         stats.DLQCount,
			}, nil
		},
		AgentStatsFn: func(fnCtx context.Context) (*health.AgentMetrics, error) {
			agents, err := jc.ListAgents(fnCtx)
			if err != nil {
				return nil, fmt.Errorf("agent stats: %w", err)
			}

			details := make([]health.AgentDetail, 0, len(agents))
			for _, a := range agents {
				labels := ""
				for k, v := range a.Labels {
					if labels != "" {
						labels += ", "
					}
					labels += k + "=" + v
				}
				registered := cli.FormatAge(time.Since(a.RegisteredAt)) + " ago"
				details = append(details, health.AgentDetail{
					Hostname:   a.Hostname,
					Labels:     labels,
					Registered: registered,
				})
			}

			return &health.AgentMetrics{
				Total:  len(agents),
				Ready:  len(agents), // presence in KV = Ready
				Agents: details,
			}, nil
		},
		ComponentRegistryFn: func(fnCtx context.Context) ([]health.ComponentEntry, error) {
			if registryKV == nil {
				return nil, fmt.Errorf("registry KV not configured")
			}

			keys, err := registryKV.Keys(fnCtx)
			if err != nil {
				if strings.Contains(err.Error(), "no keys found") {
					return []health.ComponentEntry{}, nil
				}
				return nil, fmt.Errorf("list registry keys: %w", err)
			}

			entries := make([]health.ComponentEntry, 0, len(keys))
			for _, key := range keys {
				parts := strings.SplitN(key, ".", 2)
				if len(parts) != 2 {
					continue
				}
				prefix := parts[0]

				kvEntry, err := registryKV.Get(fnCtx, key)
				if err != nil {
					continue
				}

				var entry health.ComponentEntry

				switch prefix {
				case "agents":
					var reg job.AgentRegistration
					if err := json.Unmarshal(kvEntry.Value(), &reg); err != nil {
						continue
					}
					entry = health.ComponentEntry{
						Type:     "agent",
						Hostname: reg.Hostname,
						Status:   "Ready",
						Age:      cli.FormatAge(time.Since(reg.StartedAt)),
					}
					for _, c := range reg.Conditions {
						if c.Status {
							entry.Conditions = append(entry.Conditions, c.Type)
						}
					}
					if reg.Process != nil {
						entry.CPUPercent = reg.Process.CPUPercent
						entry.MemBytes = reg.Process.RSSBytes
					}
					entry.SubComponents = convertSubComponents(reg.SubComponents)
				case "controller", "nats":
					var reg job.ComponentRegistration
					if err := json.Unmarshal(kvEntry.Value(), &reg); err != nil {
						continue
					}
					entry = health.ComponentEntry{
						Type:     reg.Type,
						Hostname: reg.Hostname,
						Status:   "Ready",
						Age:      cli.FormatAge(time.Since(reg.StartedAt)),
					}
					for _, c := range reg.Conditions {
						if c.Status {
							entry.Conditions = append(entry.Conditions, c.Type)
						}
					}
					if reg.Process != nil {
						entry.CPUPercent = reg.Process.CPUPercent
						entry.MemBytes = reg.Process.RSSBytes
					}
					entry.SubComponents = convertSubComponents(reg.SubComponents)
				default:
					continue
				}

				entries = append(entries, entry)
			}

			sort.Slice(entries, func(i, j int) bool {
				if entries[i].Type != entries[j].Type {
					return entries[i].Type < entries[j].Type
				}
				return entries[i].Hostname < entries[j].Hostname
			})

			return entries, nil
		},
	}
}

// convertSubComponents converts job.SubComponentInfo map to health.SubComponentInfo map.
func convertSubComponents(
	src map[string]job.SubComponentInfo,
) map[string]health.SubComponentInfo {
	if len(src) == 0 {
		return nil
	}

	result := make(map[string]health.SubComponentInfo, len(src))
	for k, v := range src {
		result[k] = health.SubComponentInfo{
			Status:  v.Status,
			Address: v.Address,
		}
	}

	return result
}

// subsystemStatuses converts a sub-components map into SubsystemStatus
// entries for registering osapi_subsystem_up gauges on a metrics server.
func subsystemStatuses(
	scs map[string]job.SubComponentInfo,
) []metrics.SubsystemStatus {
	result := make([]metrics.SubsystemStatus, 0, len(scs))
	for name, info := range scs {
		status := info.Status
		result = append(result, metrics.SubsystemStatus{
			Name:     name,
			StatusFn: func() string { return status },
		})
	}

	return result
}

func createAuditStore(
	ctx context.Context,
	log *slog.Logger,
	nc NATSClient,
	namespace string,
) (audit.Store, []api.Option) {
	if appConfig.NATS.Audit.Stream == "" {
		return nil, nil
	}

	auditStreamConfig := cli.BuildAuditStreamConfig(
		namespace,
		appConfig.NATS.Audit,
	)
	if err := nc.CreateOrUpdateStreamWithConfig(
		ctx,
		auditStreamConfig,
	); err != nil {
		cli.LogFatal(log, "failed to create audit stream", err)
	}

	streamName := job.ApplyNamespaceToInfraName(
		namespace,
		appConfig.NATS.Audit.Stream,
	)
	stream, err := nc.Stream(ctx, streamName)
	if err != nil {
		cli.LogFatal(log, "failed to get audit stream", err)
	}

	subject := job.ApplyNamespaceToSubjects(
		namespace,
		appConfig.NATS.Audit.Subject,
	)

	store := audit.NewStreamStore(log, stream, nc, subject)

	return store, []api.Option{api.WithAuditStore(store)}
}

func registerControllerHandlers(
	log *slog.Logger,
	sm ServerManager,
	jc jobclient.JobClient,
	checker health.Checker,
	metricsProvider health.MetricsProvider,
	auditStore audit.Store,
	objStore file.ObjectStoreManager,
	fileStateKV file.StateKeyValue,
	enrollmentWatcher *enrollment.Watcher,
) {
	startTime := time.Now()
	signingKey := appConfig.Controller.API.Security.SigningKey

	// Build custom roles map from config.
	var customRoles map[string][]string
	if cfgRoles := appConfig.Controller.API.Security.Roles; len(cfgRoles) > 0 {
		customRoles = make(map[string][]string, len(cfgRoles))
		for name, role := range cfgRoles {
			customRoles[name] = role.Permissions
		}
	}

	httpAddr := func(host string, port int) string {
		return fmt.Sprintf("http://%s:%d", host, port)
	}

	enabledOrDisabled := func(enabled bool) string {
		if enabled {
			return "ok"
		}

		return "disabled"
	}

	subComponents := map[string]health.SubComponentInfo{
		"controller.api": {
			Status:  "ok",
			Address: httpAddr("0.0.0.0", appConfig.Controller.API.Port),
		},
		"controller.heartbeat": {Status: "ok"},
		"controller.metrics": {
			Status:  enabledOrDisabled(appConfig.Controller.Metrics.Enabled),
			Address: httpAddr(appConfig.Controller.Metrics.Host, appConfig.Controller.Metrics.Port),
		},
		"controller.notifier": {
			Status: enabledOrDisabled(appConfig.Controller.Notifications.Enabled),
		},
		"controller.tracing": {Status: enabledOrDisabled(appConfig.Telemetry.Tracing.Enabled)},
	}

	handlers := make([]func(e *echo.Echo), 0, 16)
	handlers = append(
		handlers,
		agentAPI.Handler(log, jc, signingKey, customRoles, enrollmentWatcher)...,
	)
	handlers = append(handlers, nodeAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, jobAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(
		handlers,
		health.Handler(
			log,
			checker,
			startTime,
			"0.1.0",
			metricsProvider,
			subComponents,
			signingKey,
			customRoles,
		)...,
	)
	handlers = append(handlers, hostnameAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, dockerAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, scheduleAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, sysctlAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, ntpAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, powerAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, timezoneAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, networkAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, commandAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, processAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, userAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, packageAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, certificateAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, serviceAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, logAPI.Handler(log, jc, signingKey, customRoles)...)
	handlers = append(handlers, nodeFileAPI.Handler(log, jc, signingKey, customRoles)...)
	if auditStore != nil {
		handlers = append(handlers, auditAPI.Handler(log, auditStore, signingKey, customRoles)...)
	}
	if objStore != nil {
		handlers = append(
			handlers,
			file.Handler(log, objStore, fileStateKV, signingKey, customRoles)...,
		)
	}
	handlers = append(handlers, factsAPI.Handler(log, signingKey, customRoles)...)

	if appConfig.Controller.UI.UIEnabled() {
		// The embedded assets are rooted at dist/ inside the ui package.
		// fs.Sub with a static valid path cannot fail in practice, but we
		// propagate the error rather than panic to be safe.
		distFS, err := fs.Sub(uiassets.Assets, "dist")
		if err != nil {
			log.Error(
				"failed to open embedded UI assets",
				slog.String("error", err.Error()),
			)
		} else {
			handlers = append(handlers, uihandler.Handler(distFS)...)
			log.Info("embedded UI enabled", slog.String("path", "/"))
		}
	}

	sm.RegisterHandlers(handlers)
}

// setupEnrollmentWatcher creates the enrollment KV bucket, the controller PKI
// manager, and the enrollment watcher. It starts the watcher in a background
// goroutine and returns it so the agent handler can list/accept/reject agents.
func setupEnrollmentWatcher(
	ctx context.Context,
	log *slog.Logger,
	b *natsBundle,
	namespace string,
) *enrollment.Watcher {
	enrollLog := log

	// Create enrollment KV bucket.
	enrollmentKVConfig := cli.BuildEnrollmentKVConfig(namespace, appConfig.NATS.Enrollment)
	enrollmentKV, err := b.nc.CreateOrUpdateKVBucketWithConfig(ctx, enrollmentKVConfig)
	if err != nil {
		enrollLog.Warn(
			"failed to create enrollment KV bucket, enrollment disabled",
			slog.String("error", err.Error()),
		)
		return nil
	}

	// Create controller PKI manager and load or generate keypair.
	pkiFS := osfs.New()
	pkiManager := agentPKI.New(pkiFS, appConfig.Controller.PKI.KeyDir, "controller")
	if err := pkiManager.LoadOrGenerate(); err != nil {
		cli.LogFatal(enrollLog, "failed to load or generate controller keypair", err)
	}

	enrollLog.Info(
		"controller PKI initialized",
		slog.String("fingerprint", pkiManager.Fingerprint()),
		slog.String("key_dir", appConfig.Controller.PKI.KeyDir),
	)

	// Create and start enrollment watcher.
	watcher := enrollment.NewWatcher(
		enrollLog,
		b.nc,
		enrollmentKV,
		pkiManager,
		appConfig.Controller.PKI.AutoAccept,
		namespace,
	)

	go func() {
		if err := watcher.Start(ctx); err != nil {
			enrollLog.Warn(
				"enrollment watcher stopped with error",
				slog.String("error", err.Error()),
			)
		}
	}()

	return watcher
}

// startConditionWatcher creates a LogNotifier and a Watcher, then starts the
// watcher in a background goroutine when notifications are enabled. The watcher
// monitors the registry KV bucket for condition transitions.
func startConditionWatcher(
	ctx context.Context,
	log *slog.Logger,
	registryKV jetstream.KeyValue,
) {
	if !appConfig.Controller.Notifications.Enabled {
		return
	}

	if registryKV == nil {
		return
	}

	var renotifyInterval time.Duration
	if appConfig.Controller.Notifications.RenotifyInterval != "" {
		d, err := time.ParseDuration(appConfig.Controller.Notifications.RenotifyInterval)
		if err != nil {
			log.Warn(
				"invalid renotify_interval, using default (0 = disabled)",
				slog.String("value", appConfig.Controller.Notifications.RenotifyInterval),
				slog.String("error", err.Error()),
			)
		} else {
			renotifyInterval = d
		}
	}

	notifier := notify.NewLogNotifier(log)
	watcher := notify.NewWatcher(registryKV, notifier, log, renotifyInterval)

	go func() {
		if err := watcher.Start(ctx); err != nil {
			log.Warn(
				"condition watcher stopped with error",
				slog.String("error", err.Error()),
			)
		}
	}()
}

// buildControllerSubComponents creates the sub-component map for the controller.
func buildControllerSubComponents() map[string]job.SubComponentInfo {
	httpAddr := func(host string, port int) string {
		return fmt.Sprintf("http://%s:%d", host, port)
	}

	enabledOrDisabled := func(enabled bool) string {
		if enabled {
			return "ok"
		}

		return "disabled"
	}

	return map[string]job.SubComponentInfo{
		"controller.api": {
			Status:  "ok",
			Address: httpAddr("0.0.0.0", appConfig.Controller.API.Port),
		},
		"controller.heartbeat": {Status: "ok"},
		"controller.metrics": {
			Status:  enabledOrDisabled(appConfig.Controller.Metrics.Enabled),
			Address: httpAddr(appConfig.Controller.Metrics.Host, appConfig.Controller.Metrics.Port),
		},
		"controller.pki": {
			Status: enabledOrDisabled(appConfig.Controller.PKI.Enabled),
		},
		"controller.notifier": {
			Status: enabledOrDisabled(appConfig.Controller.Notifications.Enabled),
		},
		"controller.tracing": {Status: enabledOrDisabled(appConfig.Telemetry.Tracing.Enabled)},
		"controller.ui": {
			Status:  enabledOrDisabled(appConfig.Controller.UI.UIEnabled()),
			Address: httpAddr("0.0.0.0", appConfig.Controller.API.Port),
		},
	}
}

// startControllerHeartbeat resolves the local hostname, creates a ComponentHeartbeat
// for the API server, and starts it in a background goroutine. The heartbeat
// deregisters when ctx is cancelled.
func startControllerHeartbeat(
	ctx context.Context,
	log *slog.Logger,
	registryKV jetstream.KeyValue,
	subComponents map[string]job.SubComponentInfo,
) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Warn(
			"failed to resolve hostname for controller heartbeat, using 'unknown'",
			slog.String("error", err.Error()),
		)
		hostname = "unknown"
	}

	hb := controller.NewComponentHeartbeat(
		log,
		registryKV,
		hostname,
		"0.1.0",
		"controller",
		process.New(),
		10*time.Second,
		process.ConditionThresholds{
			MemoryPressureBytes: appConfig.Agent.ProcessConditions.MemoryPressureBytes,
			HighCPUPercent:      appConfig.Agent.ProcessConditions.HighCPUPercent,
		},
		subComponents,
	)

	go hb.Start(ctx)
}
