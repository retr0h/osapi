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

package queue

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/retr0h/osapi/internal/api/queue/gen"
)

// PostQueue puts a single item into the queue API endpoint.
func (q Queue) PostQueue(
	ctx echo.Context,
) error {
	var newItem gen.PostQueueJSONBody

	if err := ctx.Bind(&newItem); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.QueueErrorResponse{
			Error: err.Error(),
		})
	}

	body := newItem.Body

	err := q.Manager.Put(ctx.Request().Context(), []byte(body))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.QueueErrorResponse{
			Error: err.Error(),
		})
	}

	return ctx.NoContent(http.StatusCreated)
}
