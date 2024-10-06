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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/system/host/mocks"
)

type UbuntuHostnamePublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller
}

func (suite *UbuntuHostnamePublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
}

func (suite *UbuntuHostnamePublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *UbuntuHostnamePublicTestSuite) TestGetHostname() {
	tests := []struct {
		name        string
		setupMock   func() *mocks.MockProvider
		want        interface{}
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when GetHostname Ok",
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			want:    "default-hostname",
			wantErr: false,
		},
		{
			name: "when GetHostname errors",
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetHostname().Return("", assert.AnError)

				return mock
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			got, err := mock.GetHostname()

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuHostnamePublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuHostnamePublicTestSuite))
}
