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

type PackageTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *PackageTypesPublicTestSuite) TestPackageInfoCollectionFromList() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.PackageCollectionResponse
		validateFunc func(client.Collection[client.PackageInfoResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.PackageCollectionResponse {
				pkgName := "curl"
				version := "7.88.1"
				desc := "command line tool"
				status := "installed"
				size := int64(2048)

				return &gen.PackageCollectionResponse{
					JobId: &testUUID,
					Results: []gen.PackageEntry{
						{
							Hostname: "web-01",
							Status:   gen.PackageEntryStatusOk,
							Packages: &[]gen.PackageInfo{
								{
									Name:        &pkgName,
									Version:     &version,
									Description: &desc,
									Status:      &status,
									Size:        &size,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PackageInfoResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Packages, 1)
				suite.Equal("curl", r.Packages[0].Name)
				suite.Equal("7.88.1", r.Packages[0].Version)
				suite.Equal("command line tool", r.Packages[0].Description)
				suite.Equal("installed", r.Packages[0].Status)
				suite.Equal(int64(2048), r.Packages[0].Size)
			},
		},
		{
			name: "when optional fields are nil",
			input: func() *gen.PackageCollectionResponse {
				errMsg := "unsupported"

				return &gen.PackageCollectionResponse{
					Results: []gen.PackageEntry{
						{
							Hostname: "web-02",
							Status:   gen.PackageEntryStatusSkipped,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PackageInfoResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-02", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Equal("unsupported", r.Error)
				suite.Nil(r.Packages)
			},
		},
		{
			name: "when packages slice is empty",
			input: &gen.PackageCollectionResponse{
				Results: []gen.PackageEntry{
					{
						Hostname: "web-03",
						Status:   gen.PackageEntryStatusOk,
						Packages: &[]gen.PackageInfo{},
					},
				},
			},
			validateFunc: func(c client.Collection[client.PackageInfoResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Empty(c.Results[0].Packages)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.PackageInfoCollectionFromList(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *PackageTypesPublicTestSuite) TestPackageInfoCollectionFromGet() {
	tests := []struct {
		name         string
		input        *gen.PackageCollectionResponse
		validateFunc func(client.Collection[client.PackageInfoResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.PackageCollectionResponse {
				pkgName := "vim"
				version := "9.0"

				return &gen.PackageCollectionResponse{
					Results: []gen.PackageEntry{
						{
							Hostname: "web-01",
							Status:   gen.PackageEntryStatusOk,
							Packages: &[]gen.PackageInfo{
								{
									Name:    &pkgName,
									Version: &version,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PackageInfoResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.Require().Len(c.Results[0].Packages, 1)
				suite.Equal("vim", c.Results[0].Packages[0].Name)
				suite.Equal("9.0", c.Results[0].Packages[0].Version)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.PackageInfoCollectionFromGet(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *PackageTypesPublicTestSuite) TestPackageMutationCollectionFromInstall() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.PackageMutationResponse
		validateFunc func(client.Collection[client.PackageMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.PackageMutationResponse {
				name := "curl"
				changed := true

				return &gen.PackageMutationResponse{
					JobId: &testUUID,
					Results: []gen.PackageMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.PackageMutationResultStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PackageMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("curl", r.Name)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when error field is set",
			input: func() *gen.PackageMutationResponse {
				errMsg := "package not found"

				return &gen.PackageMutationResponse{
					Results: []gen.PackageMutationResult{
						{
							Hostname: "web-02",
							Status:   gen.PackageMutationResultStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PackageMutationResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("failed", c.Results[0].Status)
				suite.Equal("package not found", c.Results[0].Error)
				suite.False(c.Results[0].Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.PackageMutationCollectionFromInstall(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *PackageTypesPublicTestSuite) TestPackageMutationCollectionFromRemove() {
	tests := []struct {
		name         string
		input        *gen.PackageMutationResponse
		validateFunc func(client.Collection[client.PackageMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.PackageMutationResponse {
				name := "curl"
				changed := true

				return &gen.PackageMutationResponse{
					Results: []gen.PackageMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.PackageMutationResultStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PackageMutationResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.Equal("curl", c.Results[0].Name)
				suite.True(c.Results[0].Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.PackageMutationCollectionFromRemove(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *PackageTypesPublicTestSuite) TestPackageMutationCollectionFromUpdate() {
	tests := []struct {
		name         string
		input        *gen.PackageMutationResponse
		validateFunc func(client.Collection[client.PackageMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.PackageMutationResponse {
				name := "curl"
				changed := true

				return &gen.PackageMutationResponse{
					Results: []gen.PackageMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.PackageMutationResultStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PackageMutationResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.Equal("curl", c.Results[0].Name)
				suite.True(c.Results[0].Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.PackageMutationCollectionFromUpdate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *PackageTypesPublicTestSuite) TestPackageUpdateCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.UpdateCollectionResponse
		validateFunc func(client.Collection[client.PackageUpdateResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.UpdateCollectionResponse {
				name := "curl"
				currentVer := "7.88.1"
				newVer := "7.88.2"

				return &gen.UpdateCollectionResponse{
					JobId: &testUUID,
					Results: []gen.UpdateEntry{
						{
							Hostname: "web-01",
							Status:   gen.UpdateEntryStatusOk,
							Updates: &[]gen.UpdateInfo{
								{
									Name:           &name,
									CurrentVersion: &currentVer,
									NewVersion:     &newVer,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PackageUpdateResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Updates, 1)
				suite.Equal("curl", r.Updates[0].Name)
				suite.Equal("7.88.1", r.Updates[0].CurrentVersion)
				suite.Equal("7.88.2", r.Updates[0].NewVersion)
			},
		},
		{
			name: "when optional fields are nil",
			input: func() *gen.UpdateCollectionResponse {
				errMsg := "unsupported"

				return &gen.UpdateCollectionResponse{
					Results: []gen.UpdateEntry{
						{
							Hostname: "web-02",
							Status:   gen.UpdateEntryStatusSkipped,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PackageUpdateResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-02", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Equal("unsupported", r.Error)
				suite.Nil(r.Updates)
			},
		},
		{
			name: "when updates slice is empty",
			input: &gen.UpdateCollectionResponse{
				Results: []gen.UpdateEntry{
					{
						Hostname: "web-03",
						Status:   gen.UpdateEntryStatusOk,
						Updates:  &[]gen.UpdateInfo{},
					},
				},
			},
			validateFunc: func(c client.Collection[client.PackageUpdateResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Empty(c.Results[0].Updates)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.PackageUpdateCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *PackageTypesPublicTestSuite) TestPackageInfosFromGen() {
	tests := []struct {
		name         string
		input        *[]gen.PackageInfo
		validateFunc func([]client.PackageInfo)
	}{
		{
			name:  "when input is nil",
			input: nil,
			validateFunc: func(result []client.PackageInfo) {
				suite.Nil(result)
			},
		},
		{
			name:  "when input is empty",
			input: &[]gen.PackageInfo{},
			validateFunc: func(result []client.PackageInfo) {
				suite.Empty(result)
			},
		},
		{
			name: "when all fields are populated",
			input: func() *[]gen.PackageInfo {
				name := "curl"
				version := "7.88.1"
				desc := "command line tool"
				status := "installed"
				size := int64(2048)

				return &[]gen.PackageInfo{
					{
						Name:        &name,
						Version:     &version,
						Description: &desc,
						Status:      &status,
						Size:        &size,
					},
				}
			}(),
			validateFunc: func(result []client.PackageInfo) {
				suite.Require().Len(result, 1)
				suite.Equal("curl", result[0].Name)
				suite.Equal("7.88.1", result[0].Version)
				suite.Equal("command line tool", result[0].Description)
				suite.Equal("installed", result[0].Status)
				suite.Equal(int64(2048), result[0].Size)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportPackageInfosFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *PackageTypesPublicTestSuite) TestUpdateInfosFromGen() {
	tests := []struct {
		name         string
		input        *[]gen.UpdateInfo
		validateFunc func([]client.UpdateInfo)
	}{
		{
			name:  "when input is nil",
			input: nil,
			validateFunc: func(result []client.UpdateInfo) {
				suite.Nil(result)
			},
		},
		{
			name:  "when input is empty",
			input: &[]gen.UpdateInfo{},
			validateFunc: func(result []client.UpdateInfo) {
				suite.Empty(result)
			},
		},
		{
			name: "when all fields are populated",
			input: func() *[]gen.UpdateInfo {
				name := "curl"
				currentVer := "7.88.1"
				newVer := "7.88.2"

				return &[]gen.UpdateInfo{
					{
						Name:           &name,
						CurrentVersion: &currentVer,
						NewVersion:     &newVer,
					},
				}
			}(),
			validateFunc: func(result []client.UpdateInfo) {
				suite.Require().Len(result, 1)
				suite.Equal("curl", result[0].Name)
				suite.Equal("7.88.1", result[0].CurrentVersion)
				suite.Equal("7.88.2", result[0].NewVersion)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportUpdateInfosFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestPackageTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PackageTypesPublicTestSuite))
}
