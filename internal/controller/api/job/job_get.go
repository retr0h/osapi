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

package job

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"k8s.io/utils/ptr"

	"github.com/google/uuid"

	"github.com/osapi-io/osapi/internal/controller/api/job/gen"
)

// GetJobByID retrieves details of a specific job by its ID.
func (j *Job) GetJobByID(
	ctx context.Context,
	request gen.GetJobByIDRequestObject,
) (gen.GetJobByIDResponseObject, error) {
	jobID := request.Id.String()

	j.logger.Debug(
		"getting job",
		slog.String("job_id", jobID),
	)

	qj, err := j.JobClient.GetJobStatus(ctx, jobID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return gen.GetJobByID404JSONResponse{
				Error: &errMsg,
			}, nil
		}
		return gen.GetJobByID500JSONResponse{
			Error: &errMsg,
		}, nil
	}

	jobUUID := uuid.MustParse(qj.ID)
	resp := gen.GetJobByID200JSONResponse{
		Id:      &jobUUID,
		Status:  &qj.Status,
		Created: &qj.Created,
	}
	if qj.Operation != nil {
		op := map[string]interface{}(qj.Operation)
		resp.Operation = &op
	}
	if qj.Error != "" {
		resp.Error = &qj.Error
	}
	if qj.Hostname != "" {
		resp.Hostname = &qj.Hostname
	}
	if qj.UpdatedAt != "" {
		resp.UpdatedAt = &qj.UpdatedAt
	}
	if qj.Result != nil {
		var result interface{}
		_ = json.Unmarshal(qj.Result, &result)
		resp.Result = result
	}
	resp.Changed = qj.Changed

	// Expose per-agent responses for broadcast jobs.
	if len(qj.Responses) > 0 {
		respMap := make(map[string]struct {
			Changed  *bool       `json:"changed,omitempty"`
			Data     interface{} `json:"data,omitempty"`
			Error    *string     `json:"error,omitempty"`
			Hostname *string     `json:"hostname,omitempty"`
			Status   *string     `json:"status,omitempty"`
		})
		for hostname, r := range qj.Responses {
			entry := struct {
				Changed  *bool       `json:"changed,omitempty"`
				Data     interface{} `json:"data,omitempty"`
				Error    *string     `json:"error,omitempty"`
				Hostname *string     `json:"hostname,omitempty"`
				Status   *string     `json:"status,omitempty"`
			}{
				Changed:  r.Changed,
				Status:   ptr.To(string(r.Status)),
				Hostname: ptr.To(r.Hostname),
			}
			if r.Data != nil {
				var data interface{}
				_ = json.Unmarshal(r.Data, &data)
				entry.Data = data
			}
			if r.Error != "" {
				entry.Error = ptr.To(r.Error)
			}
			respMap[hostname] = entry
		}
		resp.Responses = &respMap
	}

	// Expose timeline
	if len(qj.Timeline) > 0 {
		timeline := make([]struct {
			Error     *string `json:"error,omitempty"`
			Event     *string `json:"event,omitempty"`
			Hostname  *string `json:"hostname,omitempty"`
			Message   *string `json:"message,omitempty"`
			Timestamp *string `json:"timestamp,omitempty"`
		}, len(qj.Timeline))
		for i, te := range qj.Timeline {
			ts := te.Timestamp.Format("2006-01-02T15:04:05Z07:00")
			timeline[i].Timestamp = &ts
			timeline[i].Event = ptr.To(te.Event)
			timeline[i].Hostname = ptr.To(te.Hostname)
			timeline[i].Message = ptr.To(te.Message)
			if te.Error != "" {
				timeline[i].Error = ptr.To(te.Error)
			}
		}
		resp.Timeline = &timeline
	}

	// Expose agent states
	if len(qj.AgentStates) > 0 {
		wsMap := make(map[string]struct {
			Duration *string `json:"duration,omitempty"`
			Error    *string `json:"error,omitempty"`
			Status   *string `json:"status,omitempty"`
		})
		for hostname, ws := range qj.AgentStates {
			entry := struct {
				Duration *string `json:"duration,omitempty"`
				Error    *string `json:"error,omitempty"`
				Status   *string `json:"status,omitempty"`
			}{
				Status: ptr.To(ws.Status),
			}
			if ws.Error != "" {
				entry.Error = ptr.To(ws.Error)
			}
			if ws.Duration != "" {
				entry.Duration = ptr.To(ws.Duration)
			}
			wsMap[hostname] = entry
		}
		resp.AgentStates = &wsMap
	}

	return resp, nil
}
