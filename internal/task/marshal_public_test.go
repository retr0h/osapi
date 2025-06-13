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

package task_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/task"
)

type SafeMarshalTaskToSummaryPublicTestSuite struct {
	suite.Suite
}

func (suite *SafeMarshalTaskToSummaryPublicTestSuite) SetupTest() {}

func (suite *SafeMarshalTaskToSummaryPublicTestSuite) SetupSubTest() {}

func (suite *SafeMarshalTaskToSummaryPublicTestSuite) TearDownTest() {}

func (suite *SafeMarshalTaskToSummaryPublicTestSuite) TestSafeMarshalTaskToSummary() {
	tests := []struct {
		name     string
		taskData *task.Task
		body     *[]byte
		want     string
	}{
		{
			name: "when DNS task with single server and domain",
			taskData: &task.Task{
				Type: task.ActionTypeDNS,
				Data: json.RawMessage(
					`{"dns_servers":["8.8.8.8"],"search_domains":["example.com"],"interface_name":"eth0"}`,
				),
			},
			want: "DNS: eth0 → 8.8.8.8 | example.com",
		},
		{
			name: "when DNS task with multiple servers and domains",
			taskData: &task.Task{
				Type: task.ActionTypeDNS,
				Data: json.RawMessage(
					`{"dns_servers":["8.8.8.8","8.8.4.4","1.1.1.1"],"search_domains":["example.com","test.com"],"interface_name":"eth1"}`,
				),
			},
			want: "DNS: eth1 → 8.8.8.8 (+2 more) | example.com (+1 more)",
		},
		{
			name: "when DNS task with no servers or domains",
			taskData: &task.Task{
				Type: task.ActionTypeDNS,
				Data: json.RawMessage(
					`{"dns_servers":[],"search_domains":[],"interface_name":"wlan0"}`,
				),
			},
			want: "DNS: wlan0 → none | none",
		},
		{
			name: "when shutdown task with delay and message",
			taskData: &task.Task{
				Type: task.ActionTypeShutdown,
				Data: json.RawMessage(
					`{"action_type":"reboot","delay_seconds":60,"message":"System maintenance"}`,
				),
			},
			want: "Shutdown: reboot (delay: 60s) - System maintenance",
		},
		{
			name: "when shutdown task without delay or message",
			taskData: &task.Task{
				Type: task.ActionTypeShutdown,
				Data: json.RawMessage(`{"action_type":"shutdown","delay_seconds":0,"message":""}`),
			},
			want: "Shutdown: shutdown",
		},
		{
			name: "when shutdown task with only delay",
			taskData: &task.Task{
				Type: task.ActionTypeShutdown,
				Data: json.RawMessage(`{"action_type":"reboot","delay_seconds":30,"message":""}`),
			},
			want: "Shutdown: reboot (delay: 30s)",
		},
		{
			name: "when unknown task type",
			taskData: &task.Task{
				Type: "unknown",
				Data: json.RawMessage(`{"some":"data"}`),
			},
			want: "Unknown: unknown",
		},
		{
			name: "when body is nil",
			body: nil,
			want: "N/A",
		},
		{
			name: "when body contains invalid JSON",
			body: func() *[]byte {
				invalid := []byte(`{invalid json`)
				return &invalid
			}(),
			want: "Error: failed to unmarshal JSON: invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "when DNS task with invalid action data",
			body: func() *[]byte {
				// Create a valid Task but with invalid DNS action data
				invalid := []byte(`{"type":"dns","data":"invalid"}`)
				return &invalid
			}(),
			want: "DNS: Error parsing data",
		},
		{
			name: "when shutdown task with invalid action data",
			body: func() *[]byte {
				// Create a valid Task but with invalid shutdown action data
				invalid := []byte(`{"type":"shutdown","data":"invalid"}`)
				return &invalid
			}(),
			want: "Shutdown: Error parsing data",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			var body *[]byte
			if tt.body != nil {
				body = tt.body
			} else if tt.taskData != nil {
				taskBytes, err := json.Marshal(tt.taskData)
				suite.Require().NoError(err)
				body = &taskBytes
			}

			got := task.SafeMarshalTaskToSummary(body)
			suite.Equal(tt.want, got)
		})
	}
}

func TestSafeMarshalTaskToSummaryPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SafeMarshalTaskToSummaryPublicTestSuite))
}
