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

package client_test

import (
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
	"github.com/osapi-io/osapi/pkg/sdk/client/gen"
)

type AuditTypesPublicTestSuite struct {
	suite.Suite
}

func (s *AuditTypesPublicTestSuite) TestAuditEntryFromGen() {
	now := time.Now().UTC().Truncate(time.Second)
	testUUID := openapi_types.UUID{
		0x55,
		0x0e,
		0x84,
		0x00,
		0xe2,
		0x9b,
		0x41,
		0xd4,
		0xa7,
		0x16,
		0x44,
		0x66,
		0x55,
		0x44,
		0x00,
		0x00,
	}
	operationID := "getNodeHostname"
	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"

	tests := []struct {
		name         string
		input        gen.AuditEntry
		validateFunc func(client.AuditEntry)
	}{
		{
			name: "when all fields are populated",
			input: gen.AuditEntry{
				Id:           testUUID,
				Timestamp:    now,
				User:         "admin@example.com",
				Roles:        []string{"admin", "write"},
				Method:       "GET",
				Path:         "/api/v1/node/web-01",
				ResponseCode: 200,
				DurationMs:   42,
				SourceIp:     "192.168.1.100",
				OperationId:  &operationID,
			},
			validateFunc: func(a client.AuditEntry) {
				s.Equal("550e8400-e29b-41d4-a716-446655440000", a.ID)
				s.Equal(now, a.Timestamp)
				s.Equal("admin@example.com", a.User)
				s.Equal([]string{"admin", "write"}, a.Roles)
				s.Equal("GET", a.Method)
				s.Equal("/api/v1/node/web-01", a.Path)
				s.Equal(200, a.ResponseCode)
				s.Equal(int64(42), a.DurationMs)
				s.Equal("192.168.1.100", a.SourceIP)
				s.Equal("getNodeHostname", a.OperationID)
			},
		},
		{
			name: "when TraceId is populated",
			input: gen.AuditEntry{
				Id:           testUUID,
				Timestamp:    now,
				User:         "admin@example.com",
				Roles:        []string{"admin"},
				Method:       "GET",
				Path:         "/api/v1/node/web-01",
				ResponseCode: 200,
				DurationMs:   42,
				SourceIp:     "192.168.1.100",
				TraceId:      &traceID,
			},
			validateFunc: func(a client.AuditEntry) {
				s.Equal(
					"4bf92f3577b34da6a3ce929d0e0e4736",
					a.TraceID,
				)
			},
		},
		{
			name: "when TraceId is nil",
			input: gen.AuditEntry{
				Id:           testUUID,
				Timestamp:    now,
				User:         "user@example.com",
				Roles:        []string{"read"},
				Method:       "GET",
				Path:         "/api/v1/health",
				ResponseCode: 200,
				DurationMs:   5,
				SourceIp:     "10.0.0.1",
				TraceId:      nil,
			},
			validateFunc: func(a client.AuditEntry) {
				s.Empty(a.TraceID)
			},
		},
		{
			name: "when OperationId is nil",
			input: gen.AuditEntry{
				Id:           testUUID,
				Timestamp:    now,
				User:         "user@example.com",
				Roles:        []string{"read"},
				Method:       "POST",
				Path:         "/api/v1/jobs",
				ResponseCode: 201,
				DurationMs:   15,
				SourceIp:     "10.0.0.1",
				OperationId:  nil,
			},
			validateFunc: func(a client.AuditEntry) {
				s.Equal("550e8400-e29b-41d4-a716-446655440000", a.ID)
				s.Equal(now, a.Timestamp)
				s.Equal("user@example.com", a.User)
				s.Equal([]string{"read"}, a.Roles)
				s.Equal("POST", a.Method)
				s.Equal("/api/v1/jobs", a.Path)
				s.Equal(201, a.ResponseCode)
				s.Equal(int64(15), a.DurationMs)
				s.Equal("10.0.0.1", a.SourceIP)
				s.Empty(a.OperationID)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result := client.ExportAuditEntryFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (s *AuditTypesPublicTestSuite) TestAuditListFromGen() {
	now := time.Now().UTC().Truncate(time.Second)
	testUUID1 := openapi_types.UUID{
		0x55,
		0x0e,
		0x84,
		0x00,
		0xe2,
		0x9b,
		0x41,
		0xd4,
		0xa7,
		0x16,
		0x44,
		0x66,
		0x55,
		0x44,
		0x00,
		0x01,
	}
	testUUID2 := openapi_types.UUID{
		0x55,
		0x0e,
		0x84,
		0x00,
		0xe2,
		0x9b,
		0x41,
		0xd4,
		0xa7,
		0x16,
		0x44,
		0x66,
		0x55,
		0x44,
		0x00,
		0x02,
	}

	tests := []struct {
		name         string
		input        *gen.ListAuditResponse
		validateFunc func(client.AuditList)
	}{
		{
			name: "when list contains items",
			input: &gen.ListAuditResponse{
				Items: []gen.AuditEntry{
					{
						Id:           testUUID1,
						Timestamp:    now,
						User:         "admin@example.com",
						Roles:        []string{"admin"},
						Method:       "GET",
						Path:         "/api/v1/health",
						ResponseCode: 200,
						DurationMs:   5,
						SourceIp:     "192.168.1.1",
					},
					{
						Id:           testUUID2,
						Timestamp:    now,
						User:         "user@example.com",
						Roles:        []string{"read"},
						Method:       "POST",
						Path:         "/api/v1/jobs",
						ResponseCode: 201,
						DurationMs:   30,
						SourceIp:     "10.0.0.1",
					},
				},
				TotalItems: 2,
			},
			validateFunc: func(al client.AuditList) {
				s.Equal(2, al.TotalItems)
				s.Require().Len(al.Items, 2)
				s.Equal("550e8400-e29b-41d4-a716-446655440001", al.Items[0].ID)
				s.Equal("admin@example.com", al.Items[0].User)
				s.Equal("550e8400-e29b-41d4-a716-446655440002", al.Items[1].ID)
				s.Equal("user@example.com", al.Items[1].User)
			},
		},
		{
			name: "when list is empty",
			input: &gen.ListAuditResponse{
				Items:      []gen.AuditEntry{},
				TotalItems: 0,
			},
			validateFunc: func(al client.AuditList) {
				s.Equal(0, al.TotalItems)
				s.Empty(al.Items)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result := client.ExportAuditListFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestAuditTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AuditTypesPublicTestSuite))
}
