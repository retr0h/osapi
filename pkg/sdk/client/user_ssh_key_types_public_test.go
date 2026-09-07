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

type UserSSHKeyTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *UserSSHKeyTypesPublicTestSuite) TestSSHKeyCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.SSHKeyCollectionResponse
		validateFunc func(client.Collection[client.SSHKeyInfoResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.SSHKeyCollectionResponse {
				keyType := "ssh-ed25519"
				fingerprint := "SHA256:abc123"
				comment := "user@host"

				return &gen.SSHKeyCollectionResponse{
					JobId: &testUUID,
					Results: []gen.SSHKeyEntry{
						{
							Hostname: "web-01",
							Status:   gen.SSHKeyEntryStatusOk,
							Keys: &[]gen.SSHKeyInfo{
								{
									Type:        &keyType,
									Fingerprint: &fingerprint,
									Comment:     &comment,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SSHKeyInfoResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Keys, 1)

				k := r.Keys[0]
				suite.Equal("ssh-ed25519", k.Type)
				suite.Equal("SHA256:abc123", k.Fingerprint)
				suite.Equal("user@host", k.Comment)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.SSHKeyCollectionResponse {
				errMsg := "permission denied"

				return &gen.SSHKeyCollectionResponse{
					Results: []gen.SSHKeyEntry{
						{
							Hostname: "web-01",
							Status:   gen.SSHKeyEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SSHKeyInfoResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Equal("permission denied", r.Error)
				suite.Nil(r.Keys)
			},
		},
		{
			name: "when multiple hosts",
			input: func() *gen.SSHKeyCollectionResponse {
				keyType1 := "ssh-rsa"
				keyType2 := "ssh-ed25519"

				return &gen.SSHKeyCollectionResponse{
					JobId: &testUUID,
					Results: []gen.SSHKeyEntry{
						{
							Hostname: "web-01",
							Status:   gen.SSHKeyEntryStatusOk,
							Keys:     &[]gen.SSHKeyInfo{{Type: &keyType1}},
						},
						{
							Hostname: "web-02",
							Status:   gen.SSHKeyEntryStatusOk,
							Keys:     &[]gen.SSHKeyInfo{{Type: &keyType2}},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SSHKeyInfoResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.Equal("web-02", c.Results[1].Hostname)
				suite.Equal("ssh-rsa", c.Results[0].Keys[0].Type)
				suite.Equal("ssh-ed25519", c.Results[1].Keys[0].Type)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.SSHKeyCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *UserSSHKeyTypesPublicTestSuite) TestSSHKeyInfoResultFromGen() {
	tests := []struct {
		name         string
		input        gen.SSHKeyEntry
		validateFunc func(client.SSHKeyInfoResult)
	}{
		{
			name: "when entry has keys",
			input: func() gen.SSHKeyEntry {
				keyType := "ssh-ed25519"
				fp := "SHA256:xyz"
				comment := "admin@server"

				return gen.SSHKeyEntry{
					Hostname: "web-01",
					Status:   gen.SSHKeyEntryStatusOk,
					Keys: &[]gen.SSHKeyInfo{
						{
							Type:        &keyType,
							Fingerprint: &fp,
							Comment:     &comment,
						},
					},
				}
			}(),
			validateFunc: func(r client.SSHKeyInfoResult) {
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Require().Len(r.Keys, 1)
				suite.Equal("ssh-ed25519", r.Keys[0].Type)
				suite.Equal("SHA256:xyz", r.Keys[0].Fingerprint)
				suite.Equal("admin@server", r.Keys[0].Comment)
			},
		},
		{
			name: "when entry has no keys",
			input: gen.SSHKeyEntry{
				Hostname: "web-01",
				Status:   gen.SSHKeyEntryStatusOk,
			},
			validateFunc: func(r client.SSHKeyInfoResult) {
				suite.Equal("web-01", r.Hostname)
				suite.Nil(r.Keys)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.SSHKeyInfoResultFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *UserSSHKeyTypesPublicTestSuite) TestSSHKeyInfoFromGen() {
	tests := []struct {
		name         string
		input        gen.SSHKeyInfo
		validateFunc func(client.SSHKeyInfo)
	}{
		{
			name: "when all fields populated",
			input: func() gen.SSHKeyInfo {
				keyType := "ssh-rsa"
				fp := "SHA256:def456"
				comment := "test@laptop"

				return gen.SSHKeyInfo{
					Type:        &keyType,
					Fingerprint: &fp,
					Comment:     &comment,
				}
			}(),
			validateFunc: func(k client.SSHKeyInfo) {
				suite.Equal("ssh-rsa", k.Type)
				suite.Equal("SHA256:def456", k.Fingerprint)
				suite.Equal("test@laptop", k.Comment)
			},
		},
		{
			name:  "when all fields nil",
			input: gen.SSHKeyInfo{},
			validateFunc: func(k client.SSHKeyInfo) {
				suite.Empty(k.Type)
				suite.Empty(k.Fingerprint)
				suite.Empty(k.Comment)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.SSHKeyInfoFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *UserSSHKeyTypesPublicTestSuite) TestSSHKeyMutationCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.SSHKeyMutationResponse
		validateFunc func(client.Collection[client.SSHKeyMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.SSHKeyMutationResponse {
				changed := true

				return &gen.SSHKeyMutationResponse{
					JobId: &testUUID,
					Results: []gen.SSHKeyMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.SSHKeyMutationEntryStatusOk,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SSHKeyMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when error result",
			input: func() *gen.SSHKeyMutationResponse {
				errMsg := "key already exists"
				changed := false

				return &gen.SSHKeyMutationResponse{
					Results: []gen.SSHKeyMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.SSHKeyMutationEntryStatusFailed,
							Changed:  &changed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SSHKeyMutationResult]) {
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("failed", r.Status)
				suite.False(r.Changed)
				suite.Equal("key already exists", r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.SSHKeyMutationCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *UserSSHKeyTypesPublicTestSuite) TestSSHKeyMutationResultFromGen() {
	tests := []struct {
		name         string
		input        gen.SSHKeyMutationEntry
		validateFunc func(client.SSHKeyMutationResult)
	}{
		{
			name: "when successful mutation",
			input: func() gen.SSHKeyMutationEntry {
				changed := true

				return gen.SSHKeyMutationEntry{
					Hostname: "web-01",
					Status:   gen.SSHKeyMutationEntryStatusOk,
					Changed:  &changed,
				}
			}(),
			validateFunc: func(r client.SSHKeyMutationResult) {
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when nil optional fields",
			input: gen.SSHKeyMutationEntry{
				Hostname: "web-01",
				Status:   gen.SSHKeyMutationEntryStatusSkipped,
			},
			validateFunc: func(r client.SSHKeyMutationResult) {
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.SSHKeyMutationResultFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestUserSSHKeyTypesPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(UserSSHKeyTypesPublicTestSuite))
}
