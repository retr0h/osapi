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
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/retr0h/osapi/internal/api/queue/gen"
	"github.com/retr0h/osapi/internal/errors"
)

// GetQueueID fetches a single item through the queue API endpoint.
func (q Queue) GetQueueID(
	ctx echo.Context,
	messageID string,
) error {
	queueItem, err := q.Manager.GetByID(ctx.Request().Context(), messageID)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			return ctx.JSON(http.StatusNotFound, gen.QueueErrorResponse{
				Error: err.Error(),
			})
		}

		return ctx.JSON(http.StatusInternalServerError, gen.QueueErrorResponse{
			Error: err.Error(),
		})
	}

	var decodedBody []byte
	if queueItem.Body != "" {
		decodedBody, err = base64.StdEncoding.DecodeString(queueItem.Body)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, gen.QueueErrorResponse{
				Error: fmt.Sprintf(
					"failed to decode body for queue item %s: %v",
					queueItem.ID,
					err,
				),
			})
		}
	}

	return ctx.JSON(http.StatusOK, gen.QueueItem{
		Body:     &decodedBody,
		Id:       &queueItem.ID,
		Received: &queueItem.Received,
		Created:  &queueItem.Created,
		Timeout:  &queueItem.Timeout,
		Updated:  &queueItem.Updated,
	})
}
