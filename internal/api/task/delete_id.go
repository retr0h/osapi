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
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/retr0h/osapi/internal/api/task/gen"
	"github.com/retr0h/osapi/internal/errors"
)

// DeleteTaskID deletes a single action through the task API endpoint.
func (t Task) DeleteTaskID(
	ctx echo.Context,
	messageID uint64,
) error {
	err := t.ClientManager.DeleteMessageBySeq(ctx.Request().Context(), messageID)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			return ctx.JSON(http.StatusNotFound, gen.TaskErrorResponse{
				Error: err.Error(),
			})
		}

		return ctx.JSON(http.StatusInternalServerError, gen.TaskErrorResponse{
			Error: err.Error(),
		})
	}

	return ctx.NoContent(http.StatusNoContent)
}
