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

// Package facts provides shared fact key constants and validation for @fact. references.
package facts

import "strings"

// Built-in fact key constants. These are the canonical @fact. reference
// keys that agents resolve at execution time. All validation, resolution,
// and API responses should reference these constants.
const (
	KeyInterfacePrimary = "interface.primary"
	KeyHostname         = "hostname"
	KeyArch             = "arch"
	KeyKernel           = "kernel"
	KeyFQDN             = "fqdn"
	KeyContainerized    = "containerized"
)

// CustomPrefix is the prefix for user-defined fact keys.
const CustomPrefix = "custom."

// Prefix is the reference prefix used in API request fields.
const Prefix = "@fact."

// BuiltInKeys returns the list of all built-in fact keys.
func BuiltInKeys() []string {
	return []string{
		KeyInterfacePrimary,
		KeyHostname,
		KeyArch,
		KeyKernel,
		KeyFQDN,
		KeyContainerized,
	}
}

// IsKnownKey reports whether key is a recognized fact key.
// Known keys are the built-in keys plus any key with the "custom."
// prefix followed by at least one character.
func IsKnownKey(
	key string,
) bool {
	switch key {
	case KeyInterfacePrimary, KeyHostname, KeyArch, KeyKernel, KeyFQDN, KeyContainerized:
		return true
	default:
		return IsCustomKey(key)
	}
}

// IsCustomKey reports whether key is a valid custom fact key
// (starts with "custom." and has at least one character after the prefix).
func IsCustomKey(
	key string,
) bool {
	return strings.HasPrefix(key, CustomPrefix) && len(key) > len(CustomPrefix)
}
