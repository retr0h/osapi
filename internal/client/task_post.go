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

package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/retr0h/osapi/internal/client/gen"
	"github.com/retr0h/osapi/internal/task"
)

// PostTask inserts a single item into the task API endpoint.
func (c *Client) PostTask(
	ctx context.Context,
	taskData task.Task,
) (*gen.PostTaskResponse, error) {
	// Convert task data to map for the API
	var dataMap map[string]interface{}
	dataBytes, err := json.Marshal(taskData.Data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(dataBytes, &dataMap)
	if err != nil {
		return nil, err
	}

	// Map our task types to the generated types
	var bodyType gen.PostTaskJSONBodyBodyType
	switch taskData.Type {
	case task.ActionTypeDNS:
		bodyType = gen.Dns
	case task.ActionTypeShutdown:
		bodyType = gen.Shutdown
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskData.Type)
	}

	body := gen.PostTaskJSONRequestBody{
		Body: struct {
			Data map[string]interface{}       `json:"data"`
			Type gen.PostTaskJSONBodyBodyType `json:"type"`
		}{
			Data: dataMap,
			Type: bodyType,
		},
	}

	return c.Client.PostTaskWithResponse(ctx, body)
}
