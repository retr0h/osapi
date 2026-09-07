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
		expectPerms       []string
		expectMissing     []string
	}{
		{
			name:          "admin role gets all permissions",
			roles:         []string{"admin"},
			expectPerms:   authtoken.AllPermissions,
			expectMissing: nil,
		},
		{
			name:  "write role gets write permissions but not audit",
			roles: []string{"write"},
			expectPerms: []string{
				authtoken.PermNodeRead,
				authtoken.PermNetworkRead,
				authtoken.PermNetworkWrite,
				authtoken.PermJobRead,
				authtoken.PermJobWrite,
				authtoken.PermHealthRead,
				authtoken.PermFileRead,
				authtoken.PermFileWrite,
			},
			expectMissing: []string{
				authtoken.PermAuditRead,
			},
		},
		{
			name:  "read role gets read-only permissions",
			roles: []string{"read"},
			expectPerms: []string{
				authtoken.PermNodeRead,
				authtoken.PermNetworkRead,
				authtoken.PermJobRead,
				authtoken.PermHealthRead,
				authtoken.PermFileRead,
			},
			expectMissing: []string{
				authtoken.PermNetworkWrite,
				authtoken.PermJobWrite,
				authtoken.PermAuditRead,
				authtoken.PermFileWrite,
			},
		},
		{
			name:          "unknown role gets no permissions",
			roles:         []string{"unknown"},
			expectPerms:   nil,
			expectMissing: authtoken.AllPermissions,
		},
		{
			name:          "empty roles gets no permissions",
			roles:         []string{},
			expectPerms:   nil,
			expectMissing: authtoken.AllPermissions,
		},
		{
			name:          "nil roles gets no permissions",
			roles:         nil,
			expectPerms:   nil,
			expectMissing: authtoken.AllPermissions,
		},
		{
			name:              "direct permissions override roles",
			roles:             []string{"admin"},
			directPermissions: []string{authtoken.PermNodeRead},
			expectPerms:       []string{authtoken.PermNodeRead},
			expectMissing: []string{
				authtoken.PermNetworkRead,
				authtoken.PermNetworkWrite,
				authtoken.PermJobRead,
				authtoken.PermJobWrite,
				authtoken.PermHealthRead,
				authtoken.PermAuditRead,
			},
		},
		{
			name:  "custom role overrides default",
			roles: []string{"ops"},
			customRoles: map[string][]string{
				"ops": {authtoken.PermNodeRead, authtoken.PermHealthRead},
			},
			expectPerms: []string{
				authtoken.PermNodeRead,
				authtoken.PermHealthRead,
			},
			expectMissing: []string{
				authtoken.PermNetworkRead,
				authtoken.PermNetworkWrite,
				authtoken.PermJobRead,
				authtoken.PermJobWrite,
			},
		},
		{
			name:  "custom role shadows built-in role",
			roles: []string{"read"},
			customRoles: map[string][]string{
				"read": {authtoken.PermHealthRead},
			},
			expectPerms: []string{authtoken.PermHealthRead},
			expectMissing: []string{
				authtoken.PermNodeRead,
				authtoken.PermNetworkRead,
			},
		},
		{
			name:  "multiple roles merge permissions",
			roles: []string{"read", "write"},
			expectPerms: []string{
				authtoken.PermNodeRead,
				authtoken.PermNetworkRead,
				authtoken.PermNetworkWrite,
				authtoken.PermJobRead,
				authtoken.PermJobWrite,
				authtoken.PermHealthRead,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			resolved := authtoken.ResolvePermissions(
				tt.roles,
				tt.directPermissions,
				tt.customRoles,
			)

			for _, p := range tt.expectPerms {
				s.True(resolved[p], "expected permission %s to be present", p)
			}
			for _, p := range tt.expectMissing {
				s.False(resolved[p], "expected permission %s to be absent", p)
			}
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
