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

package queue_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/errors"
	"github.com/retr0h/osapi/internal/queue"
)

type DBDeleteByIDTestSuite struct {
	suite.Suite
	db   *sql.DB
	mock sqlmock.Sqlmock
}

func (suite *DBDeleteByIDTestSuite) SetupTest() {
	db, mock, err := sqlmock.New()
	suite.Require().NoError(err)
	suite.db = db
	suite.mock = mock
}

func (suite *DBDeleteByIDTestSuite) TearDownTest() {
	suite.Require().NoError(suite.mock.ExpectationsWereMet())
}

func (suite *DBDeleteByIDTestSuite) TestDeleteByID() {
	tests := []struct {
		name        string
		setupMock   func()
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when DeleteByID Ok",
			setupMock: func() {
				query := regexp.QuoteMeta(`DELETE FROM goqite WHERE id = ?`)

				suite.mock.ExpectPrepare(query)
				suite.mock.ExpectExec(query).
					WithArgs("message-id").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "when DeleteByID finds no item (no rows affected)",
			setupMock: func() {
				query := regexp.QuoteMeta(`DELETE FROM goqite WHERE id = ?`)

				suite.mock.ExpectPrepare(query)
				suite.mock.ExpectExec(query).
					WithArgs("message-id").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:     true,
			wantErrType: errors.NewNotFoundError("no item found with ID message-id"),
		},
		{
			name: "when DeleteByID fails to prepare statement",
			setupMock: func() {
				query := regexp.QuoteMeta(`DELETE FROM goqite WHERE id = ?`)

				suite.mock.ExpectPrepare(query).WillReturnError(assert.AnError)
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when DeleteByID fails to execute statement",
			setupMock: func() {
				query := regexp.QuoteMeta(`DELETE FROM goqite WHERE id = ?`)

				suite.mock.ExpectPrepare(query)
				suite.mock.ExpectExec(query).
					WithArgs("message-id").
					WillReturnError(assert.AnError)
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when DeleteByID fails to retrieve affected rows",
			setupMock: func() {
				query := regexp.QuoteMeta(`DELETE FROM goqite WHERE id = ?`)

				suite.mock.ExpectPrepare(query)
				suite.mock.ExpectExec(query).
					WithArgs("message-id").
					WillReturnResult(sqlmock.NewErrorResult(assert.AnError))
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			q := &queue.Queue{DB: suite.db}
			err := q.DeleteByID(context.Background(), "message-id")

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
			} else {
				assert.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDBDeleteByIDTestSuite(t *testing.T) {
	suite.Run(t, new(DBDeleteByIDTestSuite))
}
