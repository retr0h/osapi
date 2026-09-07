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

package authtoken_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/authtoken"
)

type PermissionsPublicTestSuite struct {
	suite.Suite
}

func (s *PermissionsPublicTestSuite) TestResolvePermissions() {
	tests := []struct {
		name              string
		roles             []string
		directPermissions []string
		customRoles       map[string][]string
		validateFunc      func(map[string]bool)
	}{
		{
			name:  "admin role gets all permissions",
			roles: []string{"admin"},
			validateFunc: func(resolved map[string]bool) {
				for _, p := range authtoken.AllPermissions {
					s.True(resolved[p], "expected permission %s to be present", p)
				}
			},
		},
		{
			name:  "write role gets write permissions but not audit",
			roles: []string{"write"},
			validateFunc: func(resolved map[string]bool) {
				for _, p := range []string{
					authtoken.PermNodeRead,
					authtoken.PermNetworkRead,
					authtoken.PermNetworkWrite,
					authtoken.PermJobRead,
					authtoken.PermJobWrite,
					authtoken.PermHealthRead,
					authtoken.PermFileRead,
					authtoken.PermFileWrite,
				} {
					s.True(resolved[p], "expected permission %s to be present", p)
				}
				for _, p := range []string{
					authtoken.PermAuditRead,
				} {
					s.False(resolved[p], "expected permission %s to be absent", p)
				}
			},
		},
		{
			name:  "read role gets read-only permissions",
			roles: []string{"read"},
			validateFunc: func(resolved map[string]bool) {
				for _, p := range []string{
					authtoken.PermNodeRead,
					authtoken.PermNetworkRead,
					authtoken.PermJobRead,
					authtoken.PermHealthRead,
					authtoken.PermFileRead,
				} {
					s.True(resolved[p], "expected permission %s to be present", p)
				}
				for _, p := range []string{
					authtoken.PermNetworkWrite,
					authtoken.PermJobWrite,
					authtoken.PermAuditRead,
					authtoken.PermFileWrite,
				} {
					s.False(resolved[p], "expected permission %s to be absent", p)
				}
			},
		},
		{
			name:  "unknown role gets no permissions",
			roles: []string{"unknown"},
			validateFunc: func(resolved map[string]bool) {
				for _, p := range authtoken.AllPermissions {
					s.False(resolved[p], "expected permission %s to be absent", p)
				}
			},
		},
		{
			name:  "empty roles gets no permissions",
			roles: []string{},
			validateFunc: func(resolved map[string]bool) {
				for _, p := range authtoken.AllPermissions {
					s.False(resolved[p], "expected permission %s to be absent", p)
				}
			},
		},
		{
			name:  "nil roles gets no permissions",
			roles: nil,
			validateFunc: func(resolved map[string]bool) {
				for _, p := range authtoken.AllPermissions {
					s.False(resolved[p], "expected permission %s to be absent", p)
				}
			},
		},
		{
			name:              "direct permissions override roles",
			roles:             []string{"admin"},
			directPermissions: []string{authtoken.PermNodeRead},
			validateFunc: func(resolved map[string]bool) {
				for _, p := range []string{authtoken.PermNodeRead} {
					s.True(resolved[p], "expected permission %s to be present", p)
				}
				for _, p := range []string{
					authtoken.PermNetworkRead,
					authtoken.PermNetworkWrite,
					authtoken.PermJobRead,
					authtoken.PermJobWrite,
					authtoken.PermHealthRead,
					authtoken.PermAuditRead,
				} {
					s.False(resolved[p], "expected permission %s to be absent", p)
				}
			},
		},
		{
			name:  "custom role overrides default",
			roles: []string{"ops"},
			customRoles: map[string][]string{
				"ops": {authtoken.PermNodeRead, authtoken.PermHealthRead},
			},
			validateFunc: func(resolved map[string]bool) {
				for _, p := range []string{
					authtoken.PermNodeRead,
					authtoken.PermHealthRead,
				} {
					s.True(resolved[p], "expected permission %s to be present", p)
				}
				for _, p := range []string{
					authtoken.PermNetworkRead,
					authtoken.PermNetworkWrite,
					authtoken.PermJobRead,
					authtoken.PermJobWrite,
				} {
					s.False(resolved[p], "expected permission %s to be absent", p)
				}
			},
		},
		{
			name:  "custom role shadows built-in role",
			roles: []string{"read"},
			customRoles: map[string][]string{
				"read": {authtoken.PermHealthRead},
			},
			validateFunc: func(resolved map[string]bool) {
				for _, p := range []string{authtoken.PermHealthRead} {
					s.True(resolved[p], "expected permission %s to be present", p)
				}
				for _, p := range []string{
					authtoken.PermNodeRead,
					authtoken.PermNetworkRead,
				} {
					s.False(resolved[p], "expected permission %s to be absent", p)
				}
			},
		},
		{
			name:  "multiple roles merge permissions",
			roles: []string{"read", "write"},
			validateFunc: func(resolved map[string]bool) {
				for _, p := range []string{
					authtoken.PermNodeRead,
					authtoken.PermNetworkRead,
					authtoken.PermNetworkWrite,
					authtoken.PermJobRead,
					authtoken.PermJobWrite,
					authtoken.PermHealthRead,
				} {
					s.True(resolved[p], "expected permission %s to be present", p)
				}
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(authtoken.ResolvePermissions(
				tt.roles,
				tt.directPermissions,
				tt.customRoles,
			))
		})
	}
}

func (s *PermissionsPublicTestSuite) TestHasPermission() {
	tests := []struct {
		name         string
		resolved     map[string]bool
		required     string
		validateFunc func(bool)
	}{
		{
			name:     "present permission returns true",
			resolved: map[string]bool{authtoken.PermNodeRead: true},
			required: authtoken.PermNodeRead,
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:     "absent permission returns false",
			resolved: map[string]bool{authtoken.PermNodeRead: true},
			required: authtoken.PermJobWrite,
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:     "empty resolved set returns false",
			resolved: map[string]bool{},
			required: authtoken.PermNodeRead,
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:     "nil resolved set returns false",
			resolved: nil,
			required: authtoken.PermNodeRead,
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := authtoken.HasPermission(tt.resolved, tt.required)
			tt.validateFunc(result)
		})
	}
}

func TestPermissionsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PermissionsPublicTestSuite))
}
