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

package file_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider"
	"github.com/osapi-io/osapi/internal/provider/file"
	filemocks "github.com/osapi-io/osapi/internal/provider/file/mocks"
)

type TemplatePublicTestSuite struct {
	suite.Suite

	logger *slog.Logger
	ctx    context.Context
}

func (suite *TemplatePublicTestSuite) SetupTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.ctx = context.Background()
}

func (suite *TemplatePublicTestSuite) TearDownTest() {}

func (suite *TemplatePublicTestSuite) TestDeployTemplate() {
	tests := []struct {
		name        string
		template    string
		vars        map[string]any
		factsFn     provider.FactsFunc
		hostname    string
		wantContent string
		wantErr     bool
		wantErrMsg  string
		wantChanged bool
	}{
		{
			name:        "when simple var substitution",
			template:    "server {{ .Vars.host }}",
			vars:        map[string]any{"host": "10.0.0.1"},
			hostname:    "web-01",
			wantContent: "server 10.0.0.1",
			wantChanged: true,
		},
		{
			name:        "when hostname",
			template:    "# {{ .Hostname }}",
			hostname:    "web-01",
			wantContent: "# web-01",
			wantChanged: true,
		},
		{
			name:        "when conditional with vars",
			template:    `{{ if eq .Vars.env "prod" }}production{{ else }}dev{{ end }}`,
			vars:        map[string]any{"env": "prod"},
			hostname:    "web-01",
			wantContent: "production",
			wantChanged: true,
		},
		{
			name:     "when facts available",
			template: `arch: {{ index .Facts "architecture" }}`,
			factsFn: func() map[string]any {
				return map[string]any{"architecture": "amd64"}
			},
			hostname:    "web-01",
			wantContent: "arch: amd64",
			wantChanged: true,
		},
		{
			name:        "when nil facts",
			template:    "{{ .Hostname }}",
			factsFn:     nil,
			hostname:    "web-01",
			wantContent: "web-01",
			wantChanged: true,
		},
		{
			name:        "when nil vars",
			template:    "{{ .Hostname }}",
			vars:        nil,
			hostname:    "web-01",
			wantContent: "web-01",
			wantChanged: true,
		},
		{
			name:       "when template execution fails",
			template:   "{{ call .Hostname }}",
			hostname:   "web-01",
			wantErr:    true,
			wantErrMsg: "failed to render template",
		},
		{
			name:       "when invalid template syntax",
			template:   "{{ .Invalid",
			hostname:   "web-01",
			wantErr:    true,
			wantErrMsg: "failed to render template",
		},
		{
			name:       "when missing var key returns error",
			template:   "val={{ .Vars.missing }}",
			vars:       map[string]any{},
			hostname:   "web-01",
			wantErr:    true,
			wantErrMsg: "failed to render template",
		},
		{
			name:     "when missing fact key via index renders no value",
			template: `os={{ index .Facts "os_family" }}`,
			factsFn: func() map[string]any {
				return map[string]any{"architecture": "amd64"}
			},
			hostname:    "web-01",
			wantContent: "os=<no value>",
			wantChanged: true,
		},
		{
			name:        "when multiple vars",
			template:    "{{ .Vars.host }}:{{ .Vars.port }}",
			vars:        map[string]any{"host": "10.0.0.1", "port": "8080"},
			hostname:    "web-01",
			wantContent: "10.0.0.1:8080",
			wantChanged: true,
		},
		{
			name:     "when facts and vars combined",
			template: `host={{ .Hostname }} arch={{ index .Facts "architecture" }} env={{ .Vars.env }}`,
			factsFn: func() map[string]any {
				return map[string]any{"architecture": "arm64"}
			},
			vars:        map[string]any{"env": "staging"},
			hostname:    "web-02",
			wantContent: "host=web-02 arch=arm64 env=staging",
			wantChanged: true,
		},
		{
			name:        "when conditional false branch",
			template:    `{{ if eq .Vars.env "prod" }}production{{ else }}dev{{ end }}`,
			vars:        map[string]any{"env": "dev"},
			hostname:    "web-01",
			wantContent: "dev",
			wantChanged: true,
		},
		{
			name:        "when range over vars slice",
			template:    `{{ range .Vars.servers }}{{ . }} {{ end }}`,
			vars:        map[string]any{"servers": []any{"a", "b", "c"}},
			hostname:    "web-01",
			wantContent: "a b c ",
			wantChanged: true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ctrl := gomock.NewController(suite.T())
			defer ctrl.Finish()

			var appFs avfs.VFS = memfs.New()
			mockKV := jobmocks.NewMockKeyValue(ctrl)
			mockObj := filemocks.NewMockObjectStore(ctrl)

			mockObj.EXPECT().
				GetBytes(gomock.Any(), gomock.Any()).
				Return([]byte(tc.template), nil)

			if !tc.wantErr {
				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			}

			p := file.New(
				suite.logger,
				appFs,
				mockObj,
				mockKV,
				tc.hostname,
			)
			if tc.factsFn != nil {
				p.SetFactsFunc(tc.factsFn)
			}

			got, err := p.Deploy(suite.ctx, file.DeployRequest{
				ObjectName:  "test.conf",
				Path:        "/etc/test.conf",
				ContentType: "template",
				Vars:        tc.vars,
			})

			if tc.wantErr {
				suite.Error(err)
				suite.ErrorContains(err, tc.wantErrMsg)
				suite.Nil(got)
			} else {
				suite.NoError(err)
				suite.Require().NotNil(got)
				suite.Equal(tc.wantChanged, got.Changed)
				suite.Equal("/etc/test.conf", got.Path)

				data, readErr := appFs.ReadFile("/etc/test.conf")
				suite.Require().NoError(readErr)
				suite.Equal(tc.wantContent, string(data))
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestTemplatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TemplatePublicTestSuite))
}
