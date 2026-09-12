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

type CertificateTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *CertificateTypesPublicTestSuite) TestCertificateCACollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.CertificateCACollectionResponse
		validateFunc func(client.Collection[client.CertificateCAResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.CertificateCACollectionResponse {
				name := "my-ca"
				source := gen.Custom
				object := "my-ca.pem"

				return &gen.CertificateCACollectionResponse{
					JobId: &testUUID,
					Results: []gen.CertificateCAEntry{
						{
							Hostname: "web-01",
							Status:   gen.CertificateCAEntryStatusOk,
							Certificates: &[]gen.CertificateCAInfo{
								{
									Name:   &name,
									Source: &source,
									Object: &object,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.CertificateCAResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Certificates, 1)
				suite.Equal("my-ca", r.Certificates[0].Name)
				suite.Equal("custom", r.Certificates[0].Source)
				suite.Equal("my-ca.pem", r.Certificates[0].Object)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.CertificateCACollectionResponse {
				errMsg := "permission denied"

				return &gen.CertificateCACollectionResponse{
					Results: []gen.CertificateCAEntry{
						{
							Hostname: "web-01",
							Status:   gen.CertificateCAEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.CertificateCAResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Equal("permission denied", r.Error)
				suite.Nil(r.Certificates)
			},
		},
		{
			name: "when multiple results with mixed status",
			input: func() *gen.CertificateCACollectionResponse {
				name := "corp-ca"
				source := gen.System
				errMsg := "unsupported"

				return &gen.CertificateCACollectionResponse{
					JobId: &testUUID,
					Results: []gen.CertificateCAEntry{
						{
							Hostname: "web-01",
							Status:   gen.CertificateCAEntryStatusOk,
							Certificates: &[]gen.CertificateCAInfo{
								{
									Name:   &name,
									Source: &source,
								},
							},
						},
						{
							Hostname: "web-02",
							Status:   gen.CertificateCAEntryStatusSkipped,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.CertificateCAResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.Equal("ok", c.Results[0].Status)
				suite.Require().Len(c.Results[0].Certificates, 1)
				suite.Equal("corp-ca", c.Results[0].Certificates[0].Name)
				suite.Equal("system", c.Results[0].Certificates[0].Source)

				suite.Equal("web-02", c.Results[1].Hostname)
				suite.Equal("skipped", c.Results[1].Status)
				suite.Equal("unsupported", c.Results[1].Error)
				suite.Nil(c.Results[1].Certificates)
			},
		},
		{
			name: "when certificates list is empty",
			input: &gen.CertificateCACollectionResponse{
				Results: []gen.CertificateCAEntry{
					{
						Hostname:     "web-01",
						Status:       gen.CertificateCAEntryStatusOk,
						Certificates: &[]gen.CertificateCAInfo{},
					},
				},
			},
			validateFunc: func(c client.Collection[client.CertificateCAResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.NotNil(c.Results[0].Certificates)
				suite.Empty(c.Results[0].Certificates)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.CertificateCACollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *CertificateTypesPublicTestSuite) TestCertificateCAInfoFromGen() {
	tests := []struct {
		name         string
		input        gen.CertificateCAInfo
		validateFunc func(client.CertificateCA)
	}{
		{
			name: "when all fields are populated",
			input: func() gen.CertificateCAInfo {
				name := "my-ca"
				source := gen.Custom
				object := "my-ca.pem"

				return gen.CertificateCAInfo{
					Name:   &name,
					Source: &source,
					Object: &object,
				}
			}(),
			validateFunc: func(c client.CertificateCA) {
				suite.Equal("my-ca", c.Name)
				suite.Equal("custom", c.Source)
				suite.Equal("my-ca.pem", c.Object)
			},
		},
		{
			name: "when source is system",
			input: func() gen.CertificateCAInfo {
				name := "mozilla-ca"
				source := gen.System

				return gen.CertificateCAInfo{
					Name:   &name,
					Source: &source,
				}
			}(),
			validateFunc: func(c client.CertificateCA) {
				suite.Equal("mozilla-ca", c.Name)
				suite.Equal("system", c.Source)
				suite.Empty(c.Object)
			},
		},
		{
			name:  "when all fields are nil",
			input: gen.CertificateCAInfo{},
			validateFunc: func(c client.CertificateCA) {
				suite.Empty(c.Name)
				suite.Empty(c.Source)
				suite.Empty(c.Object)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.CertificateCAInfoFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *CertificateTypesPublicTestSuite) TestCertificateCAMutationCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.CertificateCAMutationResponse
		validateFunc func(client.Collection[client.CertificateCAMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.CertificateCAMutationResponse {
				name := "my-ca"
				changed := true

				return &gen.CertificateCAMutationResponse{
					JobId: &testUUID,
					Results: []gen.CertificateCAMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.CertificateCAMutationEntryStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.CertificateCAMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("my-ca", r.Name)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.CertificateCAMutationResponse {
				errMsg := "permission denied"

				return &gen.CertificateCAMutationResponse{
					Results: []gen.CertificateCAMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.CertificateCAMutationEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.CertificateCAMutationResult]) {
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
			input: &gen.CertificateCAMutationResponse{
				Results: []gen.CertificateCAMutationEntry{
					{
						Hostname: "web-01",
						Status:   gen.CertificateCAMutationEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.CertificateCAMutationResult]) {
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
			input: func() *gen.CertificateCAMutationResponse {
				name1 := "ca-1"
				name2 := "ca-2"
				changed := true

				return &gen.CertificateCAMutationResponse{
					JobId: &testUUID,
					Results: []gen.CertificateCAMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.CertificateCAMutationEntryStatusOk,
							Name:     &name1,
							Changed:  &changed,
						},
						{
							Hostname: "web-02",
							Status:   gen.CertificateCAMutationEntryStatusOk,
							Name:     &name2,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.CertificateCAMutationResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Equal("ca-1", c.Results[0].Name)
				suite.Equal("ca-2", c.Results[1].Name)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.CertificateCAMutationCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestCertificateTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CertificateTypesPublicTestSuite))
}
