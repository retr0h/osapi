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
	"log/slog"
	"time"

	"github.com/maragudk/goqite"
	_ "modernc.org/sqlite" //lint:ignore

	"github.com/retr0h/osapi/internal/config"
)

// OpenDB initializes and returns a database connection.
func OpenDB(
	appConfig config.Config,
) (*sql.DB, error) {
	db, err := sql.Open(
		appConfig.Queue.Database.DriverName,
		appConfig.Queue.Database.DataSourceName,
	)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(appConfig.Database.MaxOpenConns)
	db.SetMaxIdleConns(appConfig.Database.MaxIdleConns)

	return db, nil
}

// SetupSchema setup the schema for use with the queue.
func (q *Queue) SetupSchema(ctx context.Context) error {
	if !q.isDatabaseInitialized() {
		if err := goqite.Setup(ctx, q.DB); err != nil {
			return fmt.Errorf("error setting up database: %w", err)
		}
	}
	return nil
}

// isDatabaseInitialized checks if the necessary tables are present in the database.
func (q *Queue) isDatabaseInitialized() bool {
	var tableCount int
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='goqite';`
	err := q.DB.QueryRow(query).Scan(&tableCount)
	if err != nil {
		// log.Printf("error checking database initialization: %v", err)
		q.logger.Error(
			"eror checking database initialization",
			slog.String("error", err.Error()),
		)
		return false
	}
	return tableCount > 0
}

// GetAll the message in the queue.
func (q *Queue) GetAll(ctx context.Context, limit int, offset int) ([]Item, error) {
	query := `
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

// GetByID fetches a single item from the database by its ID.
func (q *Queue) GetByID(ctx context.Context, messageID string) (*Item, error) {
	query := `
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
			return nil, fmt.Errorf("no item found with ID %s", messageID)
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

// DeleteByID deletes a row from the database by its ID.
func (q *Queue) DeleteByID(ctx context.Context, messageID string) error {
	query := `DELETE FROM goqite WHERE id = ?`

	stmt, err := q.DB.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	result, err := stmt.ExecContext(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to execute delete statement: %w", err)
	}

	// Check if the delete operation affected any rows.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no item found with ID: %s", messageID)
	}

	return nil
}

// Count counts the number of rows in the table.
func (q *Queue) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM goqite`

	var count int
	err := q.DB.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count rows: %w", err)
	}

	return count, nil
}

// parseTimestamp parses a timestamp string in RFC3339Nano format and returns a time.Time value.
func parseTimestamp(timestamp string) (time.Time, error) {
	parsedTime, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return time.Time{}, err
	}
	return parsedTime, nil
}
