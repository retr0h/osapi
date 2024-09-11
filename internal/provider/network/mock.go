//go:build test
// +build test

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

package network

// MockNetwork is a mock implementation of the Network interface for testing.
type MockNetwork struct {
	GetResolvConfFunc func() (*DNSConfig, error)
}

// NewDefaultMockNetwork creates a MockNetwork with default return values.
func NewDefaultMockNetwork() *MockNetwork {
	return &MockNetwork{
		GetResolvConfFunc: func() (*DNSConfig, error) {
			return &DNSConfig{
				DNSServers: []string{
					"192.168.1.1",
					"8.8.8.8",
					"8.8.4.4",
					"2001:4860:4860::8888",
					"2001:4860:4860::8844",
				},
				SearchDomains: []string{
					"example.com",
					"local.lan",
				},
			}, nil
		},
	}
}

// GetResolvConf mocked for tests.
func (ms *MockNetwork) GetResolvConf() (*DNSConfig, error) {
	return ms.GetResolvConfFunc()
}
