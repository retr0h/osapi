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
	"database/sql"
	"log/slog"
	"time"

	"github.com/maragudk/goqite"
	"github.com/retr0h/osapi/internal/config"
)

// Queue implementation of Queue operations.
type Queue struct {
	logger    *slog.Logger
	appConfig config.Config

	DB    *sql.DB
	Queue *goqite.Queue
}

// Item represents an item in the goqite table.
type Item struct {
	// Unique identifier of the queue item
	ID string `json:"id"       db:"id"`
	// Creation timestamp of the queue item
	Created time.Time `json:"created"  db:"created"`
	// Last updated timestamp of the queue item
	Updated time.Time `json:"updated"  db:"updated"`
	// Name of the queue
	Queue string `json:"queue"    db:"queue"`
	// String representation of the body of the queue item
	Body string `json:"body"     db:"body"`
	// Timeout timestamp for the queue item
	Timeout time.Time `json:"timeout"  db:"timeout"`
	// Number of times the queue item has been received
	Received int `json:"received" db:"received"`
}
