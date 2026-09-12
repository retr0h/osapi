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

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
	"github.com/osapi-io/osapi/pkg/sdk/client/gen"
)

type ServiceTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *ServiceTypesPublicTestSuite) TestServiceListCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.ServiceListResponse
		validateFunc func(client.Collection[client.ServiceInfoResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.ServiceListResponse {
				name := "nginx.service"
				status := "active"
				enabled := true
				description := "A high performance web server"
				pid := 1234

				return &gen.ServiceListResponse{
					JobId: &testUUID,
					Results: []gen.ServiceListEntry{
						{
							Hostname: "web-01",
							Status:   gen.ServiceListEntryStatusOk,
							Services: &[]gen.ServiceInfo{
								{
									Name:        &name,
									Status:      &status,
									Enabled:     &enabled,
									Description: &description,
									Pid:         &pid,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ServiceInfoResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Services, 1)
				suite.Equal("nginx.service", r.Services[0].Name)
				suite.Equal("active", r.Services[0].Status)
				suite.True(r.Services[0].Enabled)
				suite.Equal("A high performance web server", r.Services[0].Description)
				suite.Equal(1234, r.Services[0].PID)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.ServiceListResponse {
				errMsg := "permission denied"

				return &gen.ServiceListResponse{
					Results: []gen.ServiceListEntry{
						{
							Hostname: "web-01",
							Status:   gen.ServiceListEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ServiceInfoResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Equal("permission denied", r.Error)
				suite.Nil(r.Services)
			},
		},
		{
			name: "when multiple results with mixed status",
			input: func() *gen.ServiceListResponse {
				name := "sshd.service"
				errMsg := "unsupported"

				return &gen.ServiceListResponse{
					JobId: &testUUID,
					Results: []gen.ServiceListEntry{
						{
							Hostname: "web-01",
							Status:   gen.ServiceListEntryStatusOk,
							Services: &[]gen.ServiceInfo{
								{
									Name: &name,
								},
							},
						},
						{
							Hostname: "web-02",
							Status:   gen.ServiceListEntryStatusSkipped,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ServiceInfoResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.Equal("ok", c.Results[0].Status)
				suite.Require().Len(c.Results[0].Services, 1)
				suite.Equal("sshd.service", c.Results[0].Services[0].Name)

				suite.Equal("web-02", c.Results[1].Hostname)
				suite.Equal("skipped", c.Results[1].Status)
				suite.Equal("unsupported", c.Results[1].Error)
				suite.Nil(c.Results[1].Services)
			},
		},
		{
			name: "when services list is empty",
			input: &gen.ServiceListResponse{
				Results: []gen.ServiceListEntry{
					{
						Hostname: "web-01",
						Status:   gen.ServiceListEntryStatusOk,
						Services: &[]gen.ServiceInfo{},
					},
				},
			},
			validateFunc: func(c client.Collection[client.ServiceInfoResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.NotNil(c.Results[0].Services)
				suite.Empty(c.Results[0].Services)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ServiceListCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *ServiceTypesPublicTestSuite) TestServiceInfoFromGen() {
	tests := []struct {
		name         string
		input        gen.ServiceInfo
		validateFunc func(client.ServiceInfo)
	}{
		{
			name: "when all fields are populated",
			input: func() gen.ServiceInfo {
				name := "nginx.service"
				status := "active"
				enabled := true
				description := "A high performance web server"
				pid := 1234

				return gen.ServiceInfo{
					Name:        &name,
					Status:      &status,
					Enabled:     &enabled,
					Description: &description,
					Pid:         &pid,
				}
			}(),
			validateFunc: func(s client.ServiceInfo) {
				suite.Equal("nginx.service", s.Name)
				suite.Equal("active", s.Status)
				suite.True(s.Enabled)
				suite.Equal("A high performance web server", s.Description)
				suite.Equal(1234, s.PID)
			},
		},
		{
			name:  "when all fields are nil",
			input: gen.ServiceInfo{},
			validateFunc: func(s client.ServiceInfo) {
				suite.Empty(s.Name)
				suite.Empty(s.Status)
				suite.False(s.Enabled)
				suite.Empty(s.Description)
				suite.Zero(s.PID)
			},
		},
		{
			name: "when partially populated",
			input: func() gen.ServiceInfo {
				name := "sshd.service"
				status := "inactive"

				return gen.ServiceInfo{
					Name:   &name,
					Status: &status,
				}
			}(),
			validateFunc: func(s client.ServiceInfo) {
				suite.Equal("sshd.service", s.Name)
				suite.Equal("inactive", s.Status)
				suite.False(s.Enabled)
				suite.Empty(s.Description)
				suite.Zero(s.PID)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ServiceInfoFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *ServiceTypesPublicTestSuite) TestServiceGetCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.ServiceGetResponse
		validateFunc func(client.Collection[client.ServiceGetResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.ServiceGetResponse {
				name := "nginx.service"
				status := "active"
				enabled := true
				description := "A high performance web server"
				pid := 1234

				return &gen.ServiceGetResponse{
					JobId: &testUUID,
					Results: []gen.ServiceGetEntry{
						{
							Hostname: "web-01",
							Status:   gen.ServiceGetEntryStatusOk,
							Service: &gen.ServiceInfo{
								Name:        &name,
								Status:      &status,
								Enabled:     &enabled,
								Description: &description,
								Pid:         &pid,
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ServiceGetResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().NotNil(r.Service)
				suite.Equal("nginx.service", r.Service.Name)
				suite.Equal("active", r.Service.Status)
				suite.True(r.Service.Enabled)
				suite.Equal("A high performance web server", r.Service.Description)
				suite.Equal(1234, r.Service.PID)
			},
		},
		{
			name: "when service is nil (error case)",
			input: func() *gen.ServiceGetResponse {
				errMsg := "service not found"

				return &gen.ServiceGetResponse{
					Results: []gen.ServiceGetEntry{
						{
							Hostname: "web-01",
							Status:   gen.ServiceGetEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ServiceGetResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Equal("service not found", r.Error)
				suite.Nil(r.Service)
			},
		},
		{
			name: "when multiple results",
			input: func() *gen.ServiceGetResponse {
				name := "nginx.service"
				errMsg := "unsupported"

				return &gen.ServiceGetResponse{
					JobId: &testUUID,
					Results: []gen.ServiceGetEntry{
						{
							Hostname: "web-01",
							Status:   gen.ServiceGetEntryStatusOk,
							Service: &gen.ServiceInfo{
								Name: &name,
							},
						},
						{
							Hostname: "web-02",
							Status:   gen.ServiceGetEntryStatusSkipped,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ServiceGetResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.NotNil(c.Results[0].Service)
				suite.Equal("nginx.service", c.Results[0].Service.Name)

				suite.Equal("web-02", c.Results[1].Hostname)
				suite.Equal("skipped", c.Results[1].Status)
				suite.Nil(c.Results[1].Service)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ServiceGetCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *ServiceTypesPublicTestSuite) TestServiceMutationCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.ServiceMutationResponse
		validateFunc func(client.Collection[client.ServiceMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.ServiceMutationResponse {
				name := "nginx.service"
				changed := true

				return &gen.ServiceMutationResponse{
					JobId: &testUUID,
					Results: []gen.ServiceMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.ServiceMutationEntryStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ServiceMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("nginx.service", r.Name)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.ServiceMutationResponse {
				errMsg := "permission denied"

				return &gen.ServiceMutationResponse{
					Results: []gen.ServiceMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.ServiceMutationEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ServiceMutationResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Empty(r.Name)
				suite.False(r.Changed)
				suite.Equal("permission denied", r.Error)
			},
		},
		{
			name: "when minimal with nil pointers",
			input: &gen.ServiceMutationResponse{
				Results: []gen.ServiceMutationEntry{
					{
						Hostname: "web-01",
						Status:   gen.ServiceMutationEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.ServiceMutationResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Empty(r.Name)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when multiple results",
			input: func() *gen.ServiceMutationResponse {
				name1 := "nginx.service"
				name2 := "sshd.service"
				changed := true

				return &gen.ServiceMutationResponse{
					JobId: &testUUID,
					Results: []gen.ServiceMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.ServiceMutationEntryStatusOk,
							Name:     &name1,
							Changed:  &changed,
						},
						{
							Hostname: "web-02",
							Status:   gen.ServiceMutationEntryStatusOk,
							Name:     &name2,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ServiceMutationResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Equal("nginx.service", c.Results[0].Name)
				suite.Equal("sshd.service", c.Results[1].Name)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ServiceMutationCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestServiceTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ServiceTypesPublicTestSuite))
}
