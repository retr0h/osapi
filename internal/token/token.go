// Copyright (c) 2024 John Dewey

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

package token

import (
	"log/slog"
)

// RoleHierarchy defines the relationship between roles and their associated scopes.
// Each role is mapped to a list of permissible scopes. This hierarchy is used to
// determine whether a user with a given role has access to specific actions.
//
// Example:
//   - "admin" includes "read", "write", and "admin" scopes.
//   - "write" includes "read" and "write" scopes.
//   - "read" includes only the "read" scope.
var RoleHierarchy = map[string][]string{
	"admin": {"read", "write", "admin"},
	"write": {"read", "write"},
	"read":  {"read"},
}

// New factory to create a new instance.
func New(
	logger *slog.Logger,
) *Token {
	return &Token{
		logger: logger,
	}
}

// GenerateAllowedRoles extracts the keys from RoleHierarchy to create a list of allowed roles.
func GenerateAllowedRoles(roleHierarchy map[string][]string) []string {
	roles := make([]string, 0, len(roleHierarchy))
	for role := range roleHierarchy {
		roles = append(roles, role)
	}
	return roles
}
