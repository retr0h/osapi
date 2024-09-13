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
	"database/sql"
	"fmt"

	"github.com/retr0h/osapi/internal/errors"
)

// GetByID fetches a single item from the database by its ID.
func (q *Queue) GetByID(ctx context.Context, messageID string) (*Item, error) {
	const query = `
		SELECT id, created, updated, queue, body, timeout, received
		FROM goqite
		WHERE id = ?
	`

	var item Item
	var created, updated, timeout string

	// Execute the query and scan the result into the item struct
	row := q.DB.QueryRowContext(ctx, query, messageID)
	err := row.Scan(&item.ID, &created, &updated, &item.Queue, &item.Body, &timeout, &item.Received)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return the custom NotFoundError instead of a generic error
			return nil, errors.NewNotFoundError(fmt.Sprintf("no item found with ID %s", messageID))
		}
		return nil, fmt.Errorf("failed to scan item: %w", err)
	}

	item.Created, err = parseTimestamp(created)
	if err != nil {
		return nil, fmt.Errorf("invalid created timestamp: %w", err)
	}

	item.Updated, err = parseTimestamp(updated)
	if err != nil {
		return nil, fmt.Errorf("invalid updated timestamp: %w", err)
	}

	item.Timeout, err = parseTimestamp(timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout timestamp: %w", err)
	}

	return &item, nil
}
