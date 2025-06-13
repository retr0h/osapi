// Copyright (c) 2025 John Dewey

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

package task_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/task"
)

type CreateAndMarshalChangeDNSActionPublicTestSuite struct {
	suite.Suite
}

func (suite *CreateAndMarshalChangeDNSActionPublicTestSuite) SetupTest() {}

func (suite *CreateAndMarshalChangeDNSActionPublicTestSuite) SetupSubTest() {}

func (suite *CreateAndMarshalChangeDNSActionPublicTestSuite) TearDownTest() {}

func (suite *CreateAndMarshalChangeDNSActionPublicTestSuite) TestCreateAndMarshalChangeDNSAction() {
	tests := []struct {
		name          string
		servers       []string
		searchDomains []string
		interfaceName string
		wantErr       bool
		wantTask      *task.Task
	}{
		{
			name:          "when creating DNS action with all fields",
			servers:       []string{"8.8.8.8", "8.8.4.4"},
			searchDomains: []string{"example.com", "test.com"},
			interfaceName: "eth0",
			wantErr:       false,
			wantTask: &task.Task{
				Type: task.ActionTypeDNS,
				Data: json.RawMessage(
					`{"dns_servers":["8.8.8.8","8.8.4.4"],"search_domains":["example.com","test.com"],"interface_name":"eth0"}`,
				),
			},
		},
		{
			name:          "when creating DNS action with single server",
			servers:       []string{"1.1.1.1"},
			searchDomains: []string{"example.com"},
			interfaceName: "wlan0",
			wantErr:       false,
			wantTask: &task.Task{
				Type: task.ActionTypeDNS,
				Data: json.RawMessage(
					`{"dns_servers":["1.1.1.1"],"search_domains":["example.com"],"interface_name":"wlan0"}`,
				),
			},
		},
		{
			name:          "when creating DNS action with empty servers",
			servers:       []string{},
			searchDomains: []string{"example.com"},
			interfaceName: "eth1",
			wantErr:       false,
			wantTask: &task.Task{
				Type: task.ActionTypeDNS,
				Data: json.RawMessage(
					`{"dns_servers":[],"search_domains":["example.com"],"interface_name":"eth1"}`,
				),
			},
		},
		{
			name:          "when creating DNS action with empty search domains",
			servers:       []string{"8.8.8.8"},
			searchDomains: []string{},
			interfaceName: "eth2",
			wantErr:       false,
			wantTask: &task.Task{
				Type: task.ActionTypeDNS,
				Data: json.RawMessage(
					`{"dns_servers":["8.8.8.8"],"search_domains":[],"interface_name":"eth2"}`,
				),
			},
		},
		{
			name:          "when creating DNS action with empty interface name",
			servers:       []string{"8.8.8.8"},
			searchDomains: []string{"example.com"},
			interfaceName: "",
			wantErr:       false,
			wantTask: &task.Task{
				Type: task.ActionTypeDNS,
				Data: json.RawMessage(
					`{"dns_servers":["8.8.8.8"],"search_domains":["example.com"],"interface_name":""}`,
				),
			},
		},
		{
			name:          "when creating DNS action with all empty fields",
			servers:       []string{},
			searchDomains: []string{},
			interfaceName: "",
			wantErr:       false,
			wantTask: &task.Task{
				Type: task.ActionTypeDNS,
				Data: json.RawMessage(`{"dns_servers":[],"search_domains":[],"interface_name":""}`),
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got, err := task.CreateAndMarshalChangeDNSAction(
				tt.servers,
				tt.searchDomains,
				tt.interfaceName,
			)

			if tt.wantErr {
				suite.Error(err)
				suite.Nil(got)
				return
			}

			suite.NoError(err)
			suite.NotNil(got)

			// Unmarshal the result and compare with expected task
			var gotTask task.Task
			err = json.Unmarshal(got, &gotTask)
			suite.NoError(err)

			// Compare task type
			suite.Equal(tt.wantTask.Type, gotTask.Type)

			// Unmarshal and compare DNS action data
			var gotDNSAction task.ChangeDNSAction
			err = json.Unmarshal(gotTask.Data, &gotDNSAction)
			suite.NoError(err)

			var expectedDNSAction task.ChangeDNSAction
			err = json.Unmarshal(tt.wantTask.Data, &expectedDNSAction)
			suite.NoError(err)

			suite.Equal(expectedDNSAction.DNSServers, gotDNSAction.DNSServers)
			suite.Equal(expectedDNSAction.SearchDomains, gotDNSAction.SearchDomains)
			suite.Equal(expectedDNSAction.InterfaceName, gotDNSAction.InterfaceName)
		})
	}
}

func TestCreateAndMarshalChangeDNSActionPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CreateAndMarshalChangeDNSActionPublicTestSuite))
}
