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

package cli_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/cli"
	climocks "github.com/osapi-io/osapi/internal/cli/mocks"
)

type LifecyclePublicTestSuite struct {
	suite.Suite
}

func TestLifecyclePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LifecyclePublicTestSuite))
}

func (suite *LifecyclePublicTestSuite) TestRunServer() {
	tests := []struct {
		name         string
		cleanupCount int
		wantCleanups int
	}{
		{
			name:         "when context cancelled stops server",
			cleanupCount: 0,
			wantCleanups: 0,
		},
		{
			name:         "when cleanup functions provided runs all",
			cleanupCount: 3,
			wantCleanups: 3,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ctrl := gomock.NewController(suite.T())
			mockServer := climocks.NewMockLifecycle(ctrl)
			mockServer.EXPECT().Stop(gomock.Any()).Times(1)

			ctx, cancel := context.WithCancel(context.Background())

			cleanupRan := 0
			cleanupFns := make([]func(), tc.cleanupCount)
			for i := range cleanupFns {
				cleanupFns[i] = func() { cleanupRan++ }
			}

			cancel()
			cli.RunServer(ctx, mockServer, cleanupFns...)

			assert.Equal(suite.T(), tc.wantCleanups, cleanupRan)
		})
	}
}
