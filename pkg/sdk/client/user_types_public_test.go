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

type UserTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *UserTypesPublicTestSuite) TestUserInfoCollectionFromList() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.UserCollectionResponse
		validateFunc func(client.Collection[client.UserInfoResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.UserCollectionResponse {
				userName := "testuser"
				uid := 1000
				gid := 1000
				home := "/home/testuser"
				shell := "/bin/bash"
				groups := []string{"sudo", "docker"}
				locked := false

				return &gen.UserCollectionResponse{
					JobId: &testUUID,
					Results: []gen.UserEntry{
						{
							Hostname: "web-01",
							Status:   gen.UserEntryStatusOk,
							Users: &[]gen.UserInfo{
								{
									Name:   &userName,
									Uid:    &uid,
									Gid:    &gid,
									Home:   &home,
									Shell:  &shell,
									Groups: &groups,
									Locked: &locked,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.UserInfoResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Users, 1)

				u := r.Users[0]
				suite.Equal("testuser", u.Name)
				suite.Equal(1000, u.UID)
				suite.Equal(1000, u.GID)
				suite.Equal("/home/testuser", u.Home)
				suite.Equal("/bin/bash", u.Shell)
				suite.Equal([]string{"sudo", "docker"}, u.Groups)
				suite.False(u.Locked)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.UserCollectionResponse {
				errMsg := "permission denied"

				return &gen.UserCollectionResponse{
					Results: []gen.UserEntry{
						{
							Hostname: "web-01",
							Status:   gen.UserEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.UserInfoResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Equal("permission denied", r.Error)
				suite.Nil(r.Users)
			},
		},
		{
			name: "when multiple hosts",
			input: func() *gen.UserCollectionResponse {
				name1 := "root"
				name2 := "admin"

				return &gen.UserCollectionResponse{
					JobId: &testUUID,
					Results: []gen.UserEntry{
						{
							Hostname: "web-01",
							Status:   gen.UserEntryStatusOk,
							Users:    &[]gen.UserInfo{{Name: &name1}},
						},
						{
							Hostname: "web-02",
							Status:   gen.UserEntryStatusOk,
							Users:    &[]gen.UserInfo{{Name: &name2}},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.UserInfoResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.Equal("web-02", c.Results[1].Hostname)
				suite.Equal("root", c.Results[0].Users[0].Name)
				suite.Equal("admin", c.Results[1].Users[0].Name)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.UserInfoCollectionFromList(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *UserTypesPublicTestSuite) TestUserInfoCollectionFromGet() {
	tests := []struct {
		name         string
		input        *gen.UserCollectionResponse
		validateFunc func(client.Collection[client.UserInfoResult])
	}{
		{
			name: "when user found",
			input: func() *gen.UserCollectionResponse {
				userName := "testuser"

				return &gen.UserCollectionResponse{
					Results: []gen.UserEntry{
						{
							Hostname: "web-01",
							Status:   gen.UserEntryStatusOk,
							Users:    &[]gen.UserInfo{{Name: &userName}},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.UserInfoResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("testuser", c.Results[0].Users[0].Name)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.UserInfoCollectionFromGet(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *UserTypesPublicTestSuite) TestUserMutationCollectionFromCreate() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.UserMutationResponse
		validateFunc func(client.Collection[client.UserMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.UserMutationResponse {
				name := "newuser"
				changed := true

				return &gen.UserMutationResponse{
					JobId: &testUUID,
					Results: []gen.UserMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.UserMutationResultStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.UserMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("newuser", r.Name)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when error result",
			input: func() *gen.UserMutationResponse {
				errMsg := "user already exists"
				changed := false

				return &gen.UserMutationResponse{
					Results: []gen.UserMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.UserMutationResultStatusFailed,
							Changed:  &changed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.UserMutationResult]) {
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("failed", r.Status)
				suite.False(r.Changed)
				suite.Equal("user already exists", r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.UserMutationCollectionFromCreate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *UserTypesPublicTestSuite) TestUserMutationCollectionFromUpdate() {
	tests := []struct {
		name         string
		input        *gen.UserMutationResponse
		validateFunc func(client.Collection[client.UserMutationResult])
	}{
		{
			name: "when update succeeds",
			input: func() *gen.UserMutationResponse {
				name := "testuser"
				changed := true

				return &gen.UserMutationResponse{
					Results: []gen.UserMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.UserMutationResultStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.UserMutationResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("testuser", c.Results[0].Name)
				suite.True(c.Results[0].Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.UserMutationCollectionFromUpdate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *UserTypesPublicTestSuite) TestUserMutationCollectionFromDelete() {
	tests := []struct {
		name         string
		input        *gen.UserMutationResponse
		validateFunc func(client.Collection[client.UserMutationResult])
	}{
		{
			name: "when delete succeeds",
			input: func() *gen.UserMutationResponse {
				name := "testuser"
				changed := true

				return &gen.UserMutationResponse{
					Results: []gen.UserMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.UserMutationResultStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.UserMutationResult]) {
				suite.Require().Len(c.Results, 1)
				suite.True(c.Results[0].Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.UserMutationCollectionFromDelete(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *UserTypesPublicTestSuite) TestUserMutationCollectionFromPassword() {
	tests := []struct {
		name         string
		input        *gen.UserMutationResponse
		validateFunc func(client.Collection[client.UserMutationResult])
	}{
		{
			name: "when password change succeeds",
			input: func() *gen.UserMutationResponse {
				name := "testuser"
				changed := true

				return &gen.UserMutationResponse{
					Results: []gen.UserMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.UserMutationResultStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.UserMutationResult]) {
				suite.Require().Len(c.Results, 1)
				suite.True(c.Results[0].Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.UserMutationCollectionFromPassword(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestUserTypesPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(UserTypesPublicTestSuite))
}
