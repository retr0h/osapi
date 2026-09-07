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
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/osapi-io/osapi/internal/telemetry/tracing"
)

type PropagationPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *PropagationPublicTestSuite) SetupTest() {
	s.ctx = context.Background()

	// Set up a real tracer provider and propagator for tests
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
}

func (s *PropagationPublicTestSuite) TestInjectExtractRoundtrip() {
	tests := []struct {
		name         string
		setupCtx     func() context.Context
		validateFunc func(originalCtx context.Context, data map[string]interface{})
	}{
		{
			name: "when active span roundtrips trace context",
			setupCtx: func() context.Context {
				ctx, _ := otel.Tracer("test").Start(s.ctx, "test-span")

				return ctx
			},
			validateFunc: func(originalCtx context.Context, data map[string]interface{}) {
				// traceparent should be set in the map
				s.Contains(data, "traceparent")
				s.NotEmpty(data["traceparent"])

				// Extract and verify trace ID matches
				extractedCtx := tracing.ExtractTraceContext(context.Background(), data)
				originalSC := trace.SpanContextFromContext(originalCtx)
				extractedSC := trace.SpanContextFromContext(extractedCtx)

				s.Equal(originalSC.TraceID(), extractedSC.TraceID())
			},
		},
		{
			name: "when no active span inject is noop",
			setupCtx: func() context.Context {
				return context.Background()
			},
			validateFunc: func(_ context.Context, data map[string]interface{}) {
				s.NotContains(data, "traceparent")
			},
		},
		{
			name: "when no traceparent extract returns original context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			validateFunc: func(_ context.Context, data map[string]interface{}) {
				extractedCtx := tracing.ExtractTraceContext(context.Background(), data)
				sc := trace.SpanContextFromContext(extractedCtx)
				s.False(sc.IsValid())
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			ctx := tc.setupCtx()
			data := make(map[string]interface{})
			tracing.InjectTraceContext(ctx, data)
			tc.validateFunc(ctx, data)
		})
	}
}

func (s *PropagationPublicTestSuite) TestHeaderInjectExtractRoundtrip() {
	tests := []struct {
		name         string
		setupCtx     func() context.Context
		validateFunc func(originalCtx context.Context, header http.Header)
	}{
		{
			name: "when active span roundtrips trace context via headers",
			setupCtx: func() context.Context {
				ctx, _ := otel.Tracer("test").Start(s.ctx, "test-span")

				return ctx
			},
			validateFunc: func(originalCtx context.Context, header http.Header) {
				s.NotEmpty(header.Get("Traceparent"))

				extractedCtx := tracing.ExtractTraceContextFromHeader(
					context.Background(),
					header,
				)
				originalSC := trace.SpanContextFromContext(originalCtx)
				extractedSC := trace.SpanContextFromContext(extractedCtx)

				s.Equal(originalSC.TraceID(), extractedSC.TraceID())
			},
		},
		{
			name: "when non-canonical header keys extracts trace context",
			setupCtx: func() context.Context {
				ctx, _ := otel.Tracer("test").Start(s.ctx, "test-span")

				return ctx
			},
			validateFunc: func(originalCtx context.Context, header http.Header) {
				// Simulate NATS JetStream delivering headers with lowercase keys
				lowercaseHeader := http.Header{}
				for k, v := range header {
					lowercaseHeader[strings.ToLower(k)] = v
				}

				extractedCtx := tracing.ExtractTraceContextFromHeader(
					context.Background(),
					lowercaseHeader,
				)
				originalSC := trace.SpanContextFromContext(originalCtx)
				extractedSC := trace.SpanContextFromContext(extractedCtx)

				s.Equal(originalSC.TraceID(), extractedSC.TraceID())
			},
		},
		{
			name: "when no active span inject is noop",
			setupCtx: func() context.Context {
				return context.Background()
			},
			validateFunc: func(_ context.Context, header http.Header) {
				s.Empty(header.Get("Traceparent"))
			},
		},
		{
			name: "when no traceparent extract returns original context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			validateFunc: func(_ context.Context, header http.Header) {
				extractedCtx := tracing.ExtractTraceContextFromHeader(
					context.Background(),
					header,
				)
				sc := trace.SpanContextFromContext(extractedCtx)
				s.False(sc.IsValid())
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			ctx := tc.setupCtx()
			header := make(http.Header)
			tracing.InjectTraceContextToHeader(ctx, header)
			tc.validateFunc(ctx, header)
		})
	}
}

func (s *PropagationPublicTestSuite) TestMapCarrierGet() {
	tests := []struct {
		name         string
		data         map[string]interface{}
		key          string
		validateFunc func(string)
	}{
		{
			name: "when key exists returns value",
			data: map[string]interface{}{
				"traceparent": "00-abc123-def456-01",
			},
			key: "traceparent",
			validateFunc: func(result string) {
				s.Equal("00-abc123-def456-01", result)
			},
		},
		{
			name: "when key does not exist returns empty string",
			data: map[string]interface{}{},
			key:  "traceparent",
			validateFunc: func(result string) {
				s.Empty(result)
			},
		},
		{
			name: "when value is not a string returns empty string",
			data: map[string]interface{}{
				"count": 42,
			},
			key: "count",
			validateFunc: func(result string) {
				s.Empty(result)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			c := tracing.ExportNewMapCarrier(tc.data)

			tc.validateFunc(c.Get(tc.key))
		})
	}
}

func (s *PropagationPublicTestSuite) TestMapCarrierSet() {
	tests := []struct {
		name         string
		key          string
		value        string
		validateFunc func(map[string]interface{})
	}{
		{
			name:  "when setting a value stores it in data",
			key:   "traceparent",
			value: "00-abc123-def456-01",
			validateFunc: func(data map[string]interface{}) {
				s.Equal("00-abc123-def456-01", data["traceparent"])
			},
		},
		{
			name:  "when setting overwrites existing value",
			key:   "traceparent",
			value: "00-new-value-01",
			validateFunc: func(data map[string]interface{}) {
				s.Equal("00-new-value-01", data["traceparent"])
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			data := map[string]interface{}{
				"traceparent": "00-old-value-01",
			}
			c := tracing.ExportNewMapCarrier(data)
			c.Set(tc.key, tc.value)

			tc.validateFunc(data)
		})
	}
}

func (s *PropagationPublicTestSuite) TestMapCarrierKeys() {
	tests := []struct {
		name         string
		data         map[string]interface{}
		validateFunc func([]string)
	}{
		{
			name: "when data has entries returns all keys",
			data: map[string]interface{}{
				"traceparent":   "00-abc123-def456-01",
				"tracestate":    "vendor=opaque",
				"custom-header": "value",
			},
			validateFunc: func(keys []string) {
				s.Len(keys, 3)
				s.ElementsMatch(
					[]string{"traceparent", "tracestate", "custom-header"},
					keys,
				)
			},
		},
		{
			name: "when data is empty returns empty slice",
			data: map[string]interface{}{},
			validateFunc: func(keys []string) {
				s.Empty(keys)
			},
		},
		{
			name: "when data has single entry returns one key",
			data: map[string]interface{}{
				"traceparent": "00-abc123-def456-01",
			},
			validateFunc: func(keys []string) {
				s.Len(keys, 1)
				s.Equal("traceparent", keys[0])
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			c := tracing.ExportNewMapCarrier(tc.data)

			tc.validateFunc(c.Keys())
		})
	}
}

func TestPropagationPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PropagationPublicTestSuite))
}
