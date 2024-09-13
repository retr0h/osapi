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
	"fmt"
	"time"
)

// GetAll the message in the queue.
func (q *Queue) GetAll(ctx context.Context, limit int, offset int) ([]Item, error) {
	const query = `
		SELECT id, created, updated, queue, body, timeout, received
		FROM goqite
		LIMIT ? OFFSET ?`

	var items []Item

	// Execute the query with limit and offset values.
	rows, err := q.DB.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var item Item
		var created, updated, timeout string

		// Scan row data into the struct fields, converting date-time fields from string.
		if err := rows.Scan(&item.ID, &created, &updated, &item.Queue, &item.Body, &timeout, &item.Received); err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
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

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return items, nil
}

// parseTimestamp parses a timestamp string in RFC3339Nano format and returns a time.Time value.
func parseTimestamp(timestamp string) (time.Time, error) {
	parsedTime, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return time.Time{}, err
	}
	return parsedTime, nil
}
