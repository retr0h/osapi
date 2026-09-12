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
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/osapi-io/osapi/internal/telemetry/tracing"
)

type SlogPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *SlogPublicTestSuite) SetupTest() {
	s.ctx = context.Background()

	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
}

func (s *SlogPublicTestSuite) TestNewTraceHandler() {
	tests := []struct {
		name         string
		setupCtx     func() context.Context
		validateFunc func(output string)
	}{
		{
			name: "when active span adds trace_id and span_id",
			setupCtx: func() context.Context {
				ctx, _ := otel.Tracer("test").Start(s.ctx, "test-span")

				return ctx
			},
			validateFunc: func(output string) {
				s.Contains(output, "trace_id=")
				s.Contains(output, "span_id=")
			},
		},
		{
			name: "when no active span does not add trace fields",
			setupCtx: func() context.Context {
				return context.Background()
			},
			validateFunc: func(output string) {
				s.NotContains(output, "trace_id=")
				s.NotContains(output, "span_id=")
			},
		},
		{
			name: "when active span preserves exact trace ID",
			setupCtx: func() context.Context {
				ctx, _ := otel.Tracer("test").Start(s.ctx, "test-span")

				return ctx
			},
			validateFunc: func(output string) {
				ctx, span := otel.Tracer("test").Start(s.ctx, "verify-span")
				defer span.End()

				expectedTraceID := trace.SpanContextFromContext(ctx).TraceID().String()
				s.NotEmpty(expectedTraceID)
				s.Contains(output, "trace_id=")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var buf bytes.Buffer
			inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			handler := tracing.NewTraceHandler(inner)
			logger := slog.New(handler)

			ctx := tc.setupCtx()
			logger.InfoContext(ctx, "test message")

			tc.validateFunc(buf.String())
		})
	}
}

func (s *SlogPublicTestSuite) TestTraceHandlerWithAttrs() {
	tests := []struct {
		name         string
		attrs        []slog.Attr
		validateFunc func(output string, h slog.Handler)
	}{
		{
			name:  "when attrs set they appear in output with trace fields",
			attrs: []slog.Attr{slog.String("key", "value")},
			validateFunc: func(output string, h slog.Handler) {
				s.NotNil(h)
				s.Contains(output, "key=value")
				s.Contains(output, "trace_id=")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer
			inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			handler := tracing.NewTraceHandler(inner)

			withAttrs := handler.WithAttrs(tt.attrs)

			logger := slog.New(withAttrs)
			ctx, _ := otel.Tracer("test").Start(s.ctx, "test-span")
			logger.InfoContext(ctx, "test")

			tt.validateFunc(buf.String(), withAttrs)
		})
	}
}

func (s *SlogPublicTestSuite) TestTraceHandlerWithGroup() {
	tests := []struct {
		name         string
		group        string
		logField     slog.Attr
		validateFunc func(output string, h slog.Handler)
	}{
		{
			name:     "when group set attributes are prefixed with group name",
			group:    "mygroup",
			logField: slog.String("field", "value"),
			validateFunc: func(output string, h slog.Handler) {
				s.NotNil(h)
				s.Contains(output, "mygroup.field=value")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer
			inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			handler := tracing.NewTraceHandler(inner)

			withGroup := handler.WithGroup(tt.group)

			logger := slog.New(withGroup)
			ctx, _ := otel.Tracer("test").Start(s.ctx, "test-span")
			logger.InfoContext(ctx, "test", tt.logField)

			tt.validateFunc(buf.String(), withGroup)
		})
	}
}

func (s *SlogPublicTestSuite) TestTraceHandlerEnabled() {
	tests := []struct {
		name         string
		level        slog.Level
		minLevel     slog.Level
		validateFunc func(enabled bool)
	}{
		{
			name:     "when level below minimum returns false",
			level:    slog.LevelDebug,
			minLevel: slog.LevelWarn,
			validateFunc: func(enabled bool) {
				s.False(enabled)
			},
		},
		{
			name:     "when level at minimum returns true",
			level:    slog.LevelWarn,
			minLevel: slog.LevelWarn,
			validateFunc: func(enabled bool) {
				s.True(enabled)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			inner := slog.NewTextHandler(nil, &slog.HandlerOptions{Level: tt.minLevel})
			handler := tracing.NewTraceHandler(inner)

			tt.validateFunc(handler.Enabled(s.ctx, tt.level))
		})
	}
}

func TestSlogPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SlogPublicTestSuite))
}
