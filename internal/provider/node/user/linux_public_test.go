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

package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider"
	"github.com/osapi-io/osapi/internal/provider/node/user"
)

type LinuxPublicTestSuite struct {
	suite.Suite

	ctx      context.Context
	provider *user.Linux
}

func (suite *LinuxPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.provider = user.NewLinuxProvider()
}

func (suite *LinuxPublicTestSuite) TestListUsers() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.ListUsers(suite.ctx)

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestGetUser() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.GetUser(suite.ctx, "testuser")

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestCreateUser() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.CreateUser(suite.ctx, user.CreateUserOpts{
				Name: "testuser",
			})

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestUpdateUser() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.UpdateUser(suite.ctx, "testuser", user.UpdateUserOpts{})

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestDeleteUser() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.DeleteUser(suite.ctx, "testuser")

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestChangePassword() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.ChangePassword(suite.ctx, "testuser", "secret")

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestListGroups() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.ListGroups(suite.ctx)

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestGetGroup() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.GetGroup(suite.ctx, "testgroup")

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestCreateGroup() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.CreateGroup(suite.ctx, user.CreateGroupOpts{
				Name: "testgroup",
			})

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestUpdateGroup() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.UpdateGroup(
				suite.ctx,
				"testgroup",
				user.UpdateGroupOpts{},
			)

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestDeleteGroup() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.DeleteGroup(suite.ctx, "testgroup")

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestListKeys() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.ListKeys(suite.ctx, "testuser")

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestAddKey() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.AddKey(suite.ctx, "testuser", user.SSHKey{
				RawLine: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test@example",
			})

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestRemoveKey() {
	tests := []struct {
		name string
	}{
		{
			name: "returns ErrUnsupported",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result, err := suite.provider.RemoveKey(suite.ctx, "testuser", "SHA256:abc123")

			suite.Error(err)
			suite.Nil(result)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func TestLinuxPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LinuxPublicTestSuite))
}
