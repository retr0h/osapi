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

package tracing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/telemetry/tracing"
)

type InitTracerPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *InitTracerPublicTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *InitTracerPublicTestSuite) TestInitTracer() {
	tests := []struct {
		name         string
		cfg          config.TracingConfig
		expectErr    bool
		errContains  string
		validateFunc func()
	}{
		{
			name: "when disabled returns noop provider",
			cfg: config.TracingConfig{
				Enabled: false,
			},
			expectErr: false,
			validateFunc: func() {
				// Noop provider should be set - creating a span should not panic
				_, span := otel.Tracer("test").Start(s.ctx, "test-span")
				defer span.End()
				s.False(span.SpanContext().IsValid())
			},
		},
		{
			name: "when stdout exporter configured",
			cfg: config.TracingConfig{
				Enabled:  true,
				Exporter: "stdout",
			},
			expectErr: false,
			validateFunc: func() {
				_, span := otel.Tracer("test").Start(s.ctx, "test-span")
				defer span.End()
				s.True(span.SpanContext().IsValid())
			},
		},
		{
			name: "when unsupported exporter returns error",
			cfg: config.TracingConfig{
				Enabled:  true,
				Exporter: "invalid",
			},
			expectErr:   true,
			errContains: "unsupported tracing exporter",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			shutdown, err := tracing.InitTracer(s.ctx, "test-service", tc.cfg)
			if tc.expectErr {
				s.Error(err)
				s.Contains(err.Error(), tc.errContains)

				return
			}

			s.NoError(err)
			s.NotNil(shutdown)

			if tc.validateFunc != nil {
				tc.validateFunc()
			}

			s.NoError(shutdown(s.ctx))
		})
	}
}

func (s *InitTracerPublicTestSuite) TestInitTracerWithInjection() {
	tests := []struct {
		name         string
		cfg          config.TracingConfig
		setupFn      func()
		teardownFn   func()
		validateFunc func(func(context.Context) error, error)
	}{
		{
			name: "when resource creation fails returns error",
			cfg: config.TracingConfig{
				Enabled: true,
			},
			setupFn: func() {
				tracing.SetResourceNewFn(func(
					_ context.Context,
					_ ...resource.Option,
				) (*resource.Resource, error) {
					return nil, errors.New("resource creation failed")
				})
			},
			teardownFn: func() {
				tracing.ResetResourceNewFn()
			},
			validateFunc: func(shutdown func(context.Context) error, err error) {
				s.Error(err)
				s.Nil(shutdown)
				s.Contains(err.Error(), "creating resource")
			},
		},
		{
			name: "when stdout exporter creation fails returns error",
			cfg: config.TracingConfig{
				Enabled:  true,
				Exporter: "stdout",
			},
			setupFn: func() {
				tracing.SetStdouttraceNewFn(func(
					_ ...stdouttrace.Option,
				) (*stdouttrace.Exporter, error) {
					return nil, errors.New("stdout exporter failed")
				})
			},
			teardownFn: func() {
				tracing.ResetStdouttraceNewFn()
			},
			validateFunc: func(shutdown func(context.Context) error, err error) {
				s.Error(err)
				s.Nil(shutdown)
				s.Contains(err.Error(), "creating stdout exporter")
			},
		},
		{
			name: "when OTLP exporter configured creates valid provider",
			cfg: config.TracingConfig{
				Enabled:      true,
				Exporter:     "otlp",
				OTLPEndpoint: "localhost:4317",
			},
			setupFn: func() {
				tracing.SetOtlptraceNewFn(func(
					_ context.Context,
					_ ...otlptracegrpc.Option,
				) (*otlptrace.Exporter, error) {
					return tracing.ExportNewNoopOTLPExporter(), nil
				})
			},
			teardownFn: func() {
				tracing.ResetOtlptraceNewFn()
			},
			validateFunc: func(shutdown func(context.Context) error, err error) {
				s.NoError(err)
				s.NotNil(shutdown)

				_, span := otel.Tracer("test").Start(s.ctx, "test-span")
				defer span.End()
				s.True(span.SpanContext().IsValid())

				s.NoError(shutdown(s.ctx))
			},
		},
		{
			name: "when OTLP exporter creation fails returns error",
			cfg: config.TracingConfig{
				Enabled:      true,
				Exporter:     "otlp",
				OTLPEndpoint: "localhost:4317",
			},
			setupFn: func() {
				tracing.SetOtlptraceNewFn(func(
					_ context.Context,
					_ ...otlptracegrpc.Option,
				) (*otlptrace.Exporter, error) {
					return nil, errors.New("otlp exporter failed")
				})
			},
			teardownFn: func() {
				tracing.ResetOtlptraceNewFn()
			},
			validateFunc: func(shutdown func(context.Context) error, err error) {
				s.Error(err)
				s.Nil(shutdown)
				s.Contains(err.Error(), "creating OTLP exporter")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			if tc.setupFn != nil {
				tc.setupFn()
			}
			if tc.teardownFn != nil {
				defer tc.teardownFn()
			}

			shutdown, err := tracing.InitTracer(s.ctx, "test-service", tc.cfg)
			tc.validateFunc(shutdown, err)
		})
	}
}

func TestInitTracerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(InitTracerPublicTestSuite))
}
