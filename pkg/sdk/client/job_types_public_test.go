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

package client_test

import (
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
	"github.com/osapi-io/osapi/pkg/sdk/client/gen"
)

type JobTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *JobTypesPublicTestSuite) TestJobCreatedFromGen() {
	tests := []struct {
		name         string
		input        *gen.CreateJobResponse
		validateFunc func(client.JobCreated)
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.CreateJobResponse {
				rev := int64(42)
				ts := "2026-03-04T12:00:00Z"
				return &gen.CreateJobResponse{
					JobId: openapi_types.UUID(
						uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					),
					Status:    "pending",
					Revision:  &rev,
					Timestamp: &ts,
				}
			}(),
			validateFunc: func(j client.JobCreated) {
				suite.Equal("11111111-1111-1111-1111-111111111111", j.JobID)
				suite.Equal("pending", j.Status)
				suite.Equal(int64(42), j.Revision)
				suite.Equal("2026-03-04T12:00:00Z", j.Timestamp)
			},
		},
		{
			name: "when optional fields are nil",
			input: &gen.CreateJobResponse{
				JobId:  openapi_types.UUID(uuid.MustParse("22222222-2222-2222-2222-222222222222")),
				Status: "pending",
			},
			validateFunc: func(j client.JobCreated) {
				suite.Equal("22222222-2222-2222-2222-222222222222", j.JobID)
				suite.Equal("pending", j.Status)
				suite.Equal(int64(0), j.Revision)
				suite.Empty(j.Timestamp)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportJobCreatedFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *JobTypesPublicTestSuite) TestJobDetailFromGen() {
	tests := []struct {
		name         string
		input        *gen.JobDetailResponse
		validateFunc func(client.JobDetail)
	}{
		{
			name: "when all fields are populated with agent states responses and timeline",
			input: func() *gen.JobDetailResponse {
				id := openapi_types.UUID(uuid.MustParse("33333333-3333-3333-3333-333333333333"))
				status := "completed"
				hostname := "web-01"
				created := "2026-03-04T12:00:00Z"
				updatedAt := "2026-03-04T12:01:00Z"
				errMsg := "something failed"
				operation := map[string]interface{}{"type": "node.hostname"}
				result := map[string]interface{}{"hostname": "web-01"}
				changed := true

				agentStatus := "completed"
				agentDuration := "1.5s"
				agentError := ""
				agentStates := map[string]struct {
					Duration *string `json:"duration,omitempty"`
					Error    *string `json:"error,omitempty"`
					Status   *string `json:"status,omitempty"`
				}{
					"web-01": {
						Status:   &agentStatus,
						Duration: &agentDuration,
						Error:    &agentError,
					},
				}

				respHostname := "web-01"
				respStatus := "completed"
				respError := ""
				respChanged := true
				respData := map[string]interface{}{"hostname": "web-01"}
				responses := map[string]struct {
					Changed  *bool       `json:"changed,omitempty"`
					Data     interface{} `json:"data,omitempty"`
					Error    *string     `json:"error,omitempty"`
					Hostname *string     `json:"hostname,omitempty"`
					Status   *string     `json:"status,omitempty"`
				}{
					"web-01": {
						Hostname: &respHostname,
						Status:   &respStatus,
						Error:    &respError,
						Changed:  &respChanged,
						Data:     respData,
					},
				}

				tlTimestamp := "2026-03-04T12:00:00Z"
				tlEvent := "submitted"
				tlHostname := "api-server"
				tlMessage := "Job submitted"
				tlError := ""
				timeline := []struct {
					Error     *string `json:"error,omitempty"`
					Event     *string `json:"event,omitempty"`
					Hostname  *string `json:"hostname,omitempty"`
					Message   *string `json:"message,omitempty"`
					Timestamp *string `json:"timestamp,omitempty"`
				}{
					{
						Timestamp: &tlTimestamp,
						Event:     &tlEvent,
						Hostname:  &tlHostname,
						Message:   &tlMessage,
						Error:     &tlError,
					},
				}

				return &gen.JobDetailResponse{
					Id:          &id,
					Status:      &status,
					Hostname:    &hostname,
					Created:     &created,
					UpdatedAt:   &updatedAt,
					Error:       &errMsg,
					Operation:   &operation,
					Result:      result,
					Changed:     &changed,
					AgentStates: &agentStates,
					Responses:   &responses,
					Timeline:    &timeline,
				}
			}(),
			validateFunc: func(j client.JobDetail) {
				suite.Equal("33333333-3333-3333-3333-333333333333", j.ID)
				suite.Equal("completed", j.Status)
				suite.Equal("web-01", j.Hostname)
				suite.Equal("2026-03-04T12:00:00Z", j.Created)
				suite.Equal("2026-03-04T12:01:00Z", j.UpdatedAt)
				suite.Equal("something failed", j.Error)
				suite.Equal(map[string]interface{}{"type": "node.hostname"}, j.Operation)
				suite.Equal(map[string]interface{}{"hostname": "web-01"}, j.Result)
				suite.NotNil(j.Changed)
				suite.True(*j.Changed)

				suite.Len(j.AgentStates, 1)
				suite.Equal("completed", j.AgentStates["web-01"].Status)
				suite.Equal("1.5s", j.AgentStates["web-01"].Duration)
				suite.Empty(j.AgentStates["web-01"].Error)

				suite.Len(j.Responses, 1)
				suite.Equal("web-01", j.Responses["web-01"].Hostname)
				suite.Equal("completed", j.Responses["web-01"].Status)
				suite.Empty(j.Responses["web-01"].Error)
				suite.NotNil(j.Responses["web-01"].Changed)
				suite.True(*j.Responses["web-01"].Changed)
				suite.Equal(
					map[string]interface{}{"hostname": "web-01"},
					j.Responses["web-01"].Data,
				)

				suite.Len(j.Timeline, 1)
				suite.Equal("2026-03-04T12:00:00Z", j.Timeline[0].Timestamp)
				suite.Equal("submitted", j.Timeline[0].Event)
				suite.Equal("api-server", j.Timeline[0].Hostname)
				suite.Equal("Job submitted", j.Timeline[0].Message)
				suite.Empty(j.Timeline[0].Error)
			},
		},
		{
			name:  "when all fields are nil",
			input: &gen.JobDetailResponse{},
			validateFunc: func(j client.JobDetail) {
				suite.Empty(j.ID)
				suite.Empty(j.Status)
				suite.Empty(j.Hostname)
				suite.Empty(j.Created)
				suite.Empty(j.UpdatedAt)
				suite.Empty(j.Error)
				suite.Nil(j.Changed)
				suite.Nil(j.Operation)
				suite.Nil(j.Result)
				suite.Nil(j.AgentStates)
				suite.Nil(j.Responses)
				suite.Nil(j.Timeline)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportJobDetailFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *JobTypesPublicTestSuite) TestJobListFromGen() {
	tests := []struct {
		name         string
		input        *gen.ListJobsResponse
		validateFunc func(client.JobList)
	}{
		{
			name: "when items are present",
			input: func() *gen.ListJobsResponse {
				id := openapi_types.UUID(uuid.MustParse("44444444-4444-4444-4444-444444444444"))
				status := "pending"
				totalItems := 1
				statusCounts := map[string]int{
					"pending":   1,
					"completed": 0,
				}
				items := []gen.JobDetailResponse{
					{
						Id:     &id,
						Status: &status,
					},
				}

				return &gen.ListJobsResponse{
					Items:        &items,
					TotalItems:   &totalItems,
					StatusCounts: &statusCounts,
				}
			}(),
			validateFunc: func(jl client.JobList) {
				suite.Equal(1, jl.TotalItems)
				suite.Equal(map[string]int{
					"pending":   1,
					"completed": 0,
				}, jl.StatusCounts)
				suite.Len(jl.Items, 1)
				suite.Equal("44444444-4444-4444-4444-444444444444", jl.Items[0].ID)
				suite.Equal("pending", jl.Items[0].Status)
			},
		},
		{
			name: "when items are empty",
			input: func() *gen.ListJobsResponse {
				totalItems := 0
				items := []gen.JobDetailResponse{}

				return &gen.ListJobsResponse{
					Items:      &items,
					TotalItems: &totalItems,
				}
			}(),
			validateFunc: func(jl client.JobList) {
				suite.Equal(0, jl.TotalItems)
				suite.Empty(jl.Items)
			},
		},
		{
			name:  "when all fields are nil",
			input: &gen.ListJobsResponse{},
			validateFunc: func(jl client.JobList) {
				suite.Equal(0, jl.TotalItems)
				suite.Nil(jl.StatusCounts)
				suite.Nil(jl.Items)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportJobListFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestJobTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(JobTypesPublicTestSuite))
}
