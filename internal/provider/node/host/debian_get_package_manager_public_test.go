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

package host_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider/node/host"
)

type DebianGetPackageManagerPublicTestSuite struct {
	suite.Suite
}

func (suite *DebianGetPackageManagerPublicTestSuite) SetupTest() {}

func (suite *DebianGetPackageManagerPublicTestSuite) TearDownTest() {}

func (suite *DebianGetPackageManagerPublicTestSuite) TestGetPackageManager() {
	tests := []struct {
		name         string
		setupMock    func(u *host.Debian)
		want         interface{}
		wantErr      bool
		validateFunc func(got string)
	}{
		{
			name: "when apt detected",
			setupMock: func(u *host.Debian) {
				u.LookPathFn = func(file string) (string, error) {
					if file == "apt" {
						return "/usr/bin/apt", nil
					}
					return "", &host.ExecNotFoundError{Name: file}
				}
			},
			want:    "apt",
			wantErr: false,
		},
		{
			name: "when dnf detected",
			setupMock: func(u *host.Debian) {
				u.LookPathFn = func(file string) (string, error) {
					if file == "dnf" {
						return "/usr/bin/dnf", nil
					}
					return "", &host.ExecNotFoundError{Name: file}
				}
			},
			want:    "dnf",
			wantErr: false,
		},
		{
			name: "when yum detected",
			setupMock: func(u *host.Debian) {
				u.LookPathFn = func(file string) (string, error) {
					if file == "yum" {
						return "/usr/bin/yum", nil
					}
					return "", &host.ExecNotFoundError{Name: file}
				}
			},
			want:    "yum",
			wantErr: false,
		},
		{
			name: "when no package manager detected",
			setupMock: func(u *host.Debian) {
				u.LookPathFn = func(_ string) (string, error) {
					return "", &host.ExecNotFoundError{Name: "unknown"}
				}
			},
			want:    "unknown",
			wantErr: false,
		},
		{
			name: "when ExecNotFoundError formats message",
			setupMock: func(u *host.Debian) {
				u.LookPathFn = func(file string) (string, error) {
					return "", &host.ExecNotFoundError{Name: file}
				}
			},
			want:    "unknown",
			wantErr: false,
			validateFunc: func(_ string) {
				err := &host.ExecNotFoundError{Name: "apt"}
				suite.Equal("executable file not found: apt", err.Error())
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			debian := host.NewDebianProvider(nil)

			if tc.setupMock != nil {
				tc.setupMock(debian)
			}

			got, err := debian.GetPackageManager()

			if tc.wantErr {
				suite.Error(err)
				suite.Empty(got)
			} else {
				suite.NoError(err)
				suite.Equal(tc.want, got)
			}

			if tc.validateFunc != nil {
				tc.validateFunc(got)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDebianGetPackageManagerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianGetPackageManagerPublicTestSuite))
}
