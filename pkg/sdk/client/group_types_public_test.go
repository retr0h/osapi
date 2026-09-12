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

type GroupTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *GroupTypesPublicTestSuite) TestGroupInfoCollectionFromList() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.GroupCollectionResponse
		validateFunc func(client.Collection[client.GroupInfoResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.GroupCollectionResponse {
				groupName := "sudo"
				gid := 27
				members := []string{"testuser", "admin"}

				return &gen.GroupCollectionResponse{
					JobId: &testUUID,
					Results: []gen.GroupEntry{
						{
							Hostname: "web-01",
							Status:   gen.GroupEntryStatusOk,
							Groups: &[]gen.GroupInfo{
								{
									Name:    &groupName,
									Gid:     &gid,
									Members: &members,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.GroupInfoResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Groups, 1)

				g := r.Groups[0]
				suite.Equal("sudo", g.Name)
				suite.Equal(27, g.GID)
				suite.Equal([]string{"testuser", "admin"}, g.Members)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.GroupCollectionResponse {
				errMsg := "permission denied"

				return &gen.GroupCollectionResponse{
					Results: []gen.GroupEntry{
						{
							Hostname: "web-01",
							Status:   gen.GroupEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.GroupInfoResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Equal("permission denied", r.Error)
				suite.Nil(r.Groups)
			},
		},
		{
			name: "when multiple hosts",
			input: func() *gen.GroupCollectionResponse {
				name1 := "sudo"
				name2 := "docker"

				return &gen.GroupCollectionResponse{
					JobId: &testUUID,
					Results: []gen.GroupEntry{
						{
							Hostname: "web-01",
							Status:   gen.GroupEntryStatusOk,
							Groups:   &[]gen.GroupInfo{{Name: &name1}},
						},
						{
							Hostname: "web-02",
							Status:   gen.GroupEntryStatusOk,
							Groups:   &[]gen.GroupInfo{{Name: &name2}},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.GroupInfoResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.Equal("web-02", c.Results[1].Hostname)
				suite.Equal("sudo", c.Results[0].Groups[0].Name)
				suite.Equal("docker", c.Results[1].Groups[0].Name)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.GroupInfoCollectionFromList(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *GroupTypesPublicTestSuite) TestGroupInfoCollectionFromGet() {
	tests := []struct {
		name         string
		input        *gen.GroupCollectionResponse
		validateFunc func(client.Collection[client.GroupInfoResult])
	}{
		{
			name: "when group found",
			input: func() *gen.GroupCollectionResponse {
				groupName := "docker"

				return &gen.GroupCollectionResponse{
					Results: []gen.GroupEntry{
						{
							Hostname: "web-01",
							Status:   gen.GroupEntryStatusOk,
							Groups:   &[]gen.GroupInfo{{Name: &groupName}},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.GroupInfoResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("docker", c.Results[0].Groups[0].Name)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.GroupInfoCollectionFromGet(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *GroupTypesPublicTestSuite) TestGroupMutationCollectionFromCreate() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.GroupMutationResponse
		validateFunc func(client.Collection[client.GroupMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.GroupMutationResponse {
				name := "newgroup"
				changed := true

				return &gen.GroupMutationResponse{
					JobId: &testUUID,
					Results: []gen.GroupMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.GroupMutationResultStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.GroupMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("newgroup", r.Name)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when error result",
			input: func() *gen.GroupMutationResponse {
				errMsg := "group already exists"
				changed := false

				return &gen.GroupMutationResponse{
					Results: []gen.GroupMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.GroupMutationResultStatusFailed,
							Changed:  &changed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.GroupMutationResult]) {
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("failed", r.Status)
				suite.False(r.Changed)
				suite.Equal("group already exists", r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.GroupMutationCollectionFromCreate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *GroupTypesPublicTestSuite) TestGroupMutationCollectionFromUpdate() {
	tests := []struct {
		name         string
		input        *gen.GroupMutationResponse
		validateFunc func(client.Collection[client.GroupMutationResult])
	}{
		{
			name: "when update succeeds",
			input: func() *gen.GroupMutationResponse {
				name := "docker"
				changed := true

				return &gen.GroupMutationResponse{
					Results: []gen.GroupMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.GroupMutationResultStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.GroupMutationResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("docker", c.Results[0].Name)
				suite.True(c.Results[0].Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.GroupMutationCollectionFromUpdate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *GroupTypesPublicTestSuite) TestGroupMutationCollectionFromDelete() {
	tests := []struct {
		name         string
		input        *gen.GroupMutationResponse
		validateFunc func(client.Collection[client.GroupMutationResult])
	}{
		{
			name: "when delete succeeds",
			input: func() *gen.GroupMutationResponse {
				name := "testgroup"
				changed := true

				return &gen.GroupMutationResponse{
					Results: []gen.GroupMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.GroupMutationResultStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.GroupMutationResult]) {
				suite.Require().Len(c.Results, 1)
				suite.True(c.Results[0].Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.GroupMutationCollectionFromDelete(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestGroupTypesPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(GroupTypesPublicTestSuite))
}
