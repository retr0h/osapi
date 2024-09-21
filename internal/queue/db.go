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
		q.logger.Error(
			"eror checking database initialization",
			slog.String("error", err.Error()),
		)
		return false
	}
	return tableCount > 0
}
