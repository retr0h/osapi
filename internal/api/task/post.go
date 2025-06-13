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

package task

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/retr0h/osapi/internal/api/task/gen"
)

// PostTask puts a single item into the task API endpoint.
func (t Task) PostTask(
	ctx echo.Context,
) error {
	var newItem gen.PostTaskJSONBody

	if err := ctx.Bind(&newItem); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.TaskErrorResponse{
			Error: err.Error(),
		})
	}

	if newItem.Body.Type == "" || newItem.Body.Data == nil {
		return ctx.JSON(http.StatusBadRequest, gen.TaskErrorResponse{
			Error: "Body field with type and data is required and cannot be empty",
		})
	}

	// Marshal the task JSON object to bytes for the message queue
	taskBytes, err := json.Marshal(newItem.Body)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.TaskErrorResponse{
			Error: "Invalid task format: " + err.Error(),
		})
	}

	seq, err := t.ClientManager.PublishToStream(ctx.Request().Context(), taskBytes)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.TaskErrorResponse{
			Error: err.Error(),
		})
	}

	return ctx.JSON(http.StatusCreated, gen.TaskItemIdResponse{
		Id: &seq,
	})
}
