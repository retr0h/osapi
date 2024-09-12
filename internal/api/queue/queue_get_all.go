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
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/retr0h/osapi/internal/api/queue/gen"
)

// GetQueue get the queue.
func (q Queue) GetQueue(
	ctx echo.Context,
	params gen.GetQueueParams,
) error {
	// NOTE(retr0h): Viper also defines defaults, this feels unnecessary...
	limit := 10
	offset := 0

	if params.Limit != nil {
		limit = *params.Limit
	}

	if params.Offset != nil {
		offset = *params.Offset
	}

	queueItems, err := q.Manager.GetAll(context.Background(), limit, offset)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.QueueErrorResponse{
			Error: err.Error(),
		})
	}

	items := make([]gen.QueueItem, 0, len(queueItems))
	for _, i := range queueItems {
		item := gen.QueueItem{
			Body:     &i.Body,
			Id:       &i.ID,
			Received: &i.Received,
			Created:  &i.Created,
			Timeout:  &i.Timeout,
			Updated:  &i.Updated,
		}
		items = append(items, item)
	}

	totalItems, err := q.Manager.Count(context.Background())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.QueueErrorResponse{
			Error: err.Error(),
		})
	}

	return ctx.JSON(http.StatusOK, gen.QueueResponse{
		Items:      &items,
		TotalItems: &totalItems,
	})
}
