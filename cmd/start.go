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

package cmd

import (
	"context"
	"sync"

	"github.com/spf13/cobra"

	"github.com/osapi-io/osapi/internal/cli"
	"github.com/osapi-io/osapi/internal/controller/api"
	"github.com/osapi-io/osapi/internal/job"
	jobclient "github.com/osapi-io/osapi/internal/job/client"
	"github.com/osapi-io/osapi/internal/telemetry/metrics"
	"github.com/osapi-io/osapi/internal/telemetry/tracing"
)

// compositeLifecycle manages multiple Lifecycle components, starting them
// sequentially and stopping them concurrently.
type compositeLifecycle struct {
	components []cli.Lifecycle
}

func (c *compositeLifecycle) Start() {
	for _, comp := range c.components {
		comp.Start()
	}
}

func (c *compositeLifecycle) Stop(
	ctx context.Context,
) {
	var wg sync.WaitGroup
	for _, comp := range c.components {
		wg.Add(1)
		go func(lc cli.Lifecycle) {
			defer wg.Done()
			lc.Stop(ctx)
		}(comp)
	}
	wg.Wait()
}

// startCmd represents the top-level start command.
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start all components (NATS, controller, agent)",
	Long: `Start the embedded NATS server, controller, and agent in a single process.

This is the recommended way to run OSAPI on a single host. All three components
start in order (NATS → controller → agent) and shut down gracefully on SIGINT/SIGTERM.
`,
	Run: func(cmd *cobra.Command, _ []string) {
		ctx := cmd.Context()

		cli.ValidateDistribution(logger)

		shutdownTracer, err := tracing.InitTracer(
			ctx,
			"osapi",
			appConfig.Telemetry.Tracing,
		)
		if err != nil {
			cli.LogFatal(logger, "failed to initialize tracer", err)
		}

		job.Init(appConfig.Controller.NATS.Namespace)

		natsLog := logger.With("component", "nats")
		natsServer := setupNATSServer(ctx, natsLog)

		// Create the controller metrics server before the API server so the
		// MeterProvider can be injected into otelecho middleware.
		var metricsServers []*metrics.Server
		var controllerOpts []api.Option
		var controllerMetrics *metrics.Server

		if appConfig.Controller.Metrics.Enabled {
			s := metrics.New(
				appConfig.Controller.Metrics.Host,
				appConfig.Controller.Metrics.Port,
				logger.With("component", "controller"),
			)
			if s != nil {
				controllerMetrics = s
				controllerOpts = append(
					controllerOpts,
					api.WithMeterProvider(s.MeterProvider()),
				)
				metricsServers = append(metricsServers, s)
			}
		}

		sm, controllerBundle := setupController(
			ctx, logger.With("component", "controller"),
			appConfig.Controller.NATS,
			controllerOpts...,
		)

		if controllerMetrics != nil {
			if jc, ok := controllerBundle.jobClient.(*jobclient.Client); ok {
				jc.SetMeterProvider(controllerMetrics.MeterProvider())
			}
		}

		// Set readiness, register subsystems, and start controller metrics server.
		for _, s := range metricsServers {
			s.SetReadinessFunc(func() error {
				return controllerBundle.checker.CheckHealth(context.Background())
			})
			s.RegisterSubsystems(subsystemStatuses(controllerBundle.subComponents))
			s.Start()
		}

		startNATSHeartbeatFromKV(ctx, natsLog, controllerBundle.registryKV)

		agentServer, agentBundle, agentSubs := setupAgent(
			ctx, logger.With("component", "agent"), appConfig.Agent.NATS,
		)

		if appConfig.Agent.Metrics.Enabled {
			s := metrics.New(
				appConfig.Agent.Metrics.Host,
				appConfig.Agent.Metrics.Port,
				logger.With("component", "agent"),
			)
			s.SetReadinessFunc(func() error {
				return agentServer.IsReady()
			})

			agentServer.SetMeterProvider(s.MeterProvider())
			s.RegisterSubsystems(subsystemStatuses(agentSubs))
			s.RegisterHeartbeatAge(agentServer.LastHeartbeatTime)

			s.Start()

			metricsServers = append(metricsServers, s)
		}

		if appConfig.NATS.Server.Metrics.Enabled {
			s := metrics.New(
				appConfig.NATS.Server.Metrics.Host,
				appConfig.NATS.Server.Metrics.Port,
				logger.With("component", "nats"),
			)
			s.SetReadinessFunc(func() error {
				return nil
			})
			s.RegisterSubsystems(subsystemStatuses(buildNATSSubComponents()))
			s.Start()

			metricsServers = append(metricsServers, s)
		}

		composite := &compositeLifecycle{
			components: []cli.Lifecycle{
				sm,
				agentServer,
				&natsLifecycle{server: natsServer},
			},
		}

		composite.Start()
		cli.RunServer(ctx, composite, func() {
			for _, s := range metricsServers {
				s.Stop(context.Background())
			}

			_ = shutdownTracer(context.Background())
			cli.CloseNATSClient(agentBundle.nc)
			cli.CloseNATSClient(controllerBundle.nc)
		})
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
