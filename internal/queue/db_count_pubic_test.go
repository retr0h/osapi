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
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/queue"
)

type DBCountPublicTestSuite struct {
	suite.Suite

	db   *sql.DB
	mock sqlmock.Sqlmock
}

func (suite *DBCountPublicTestSuite) SetupTest() {
	db, mock, err := sqlmock.New()
	suite.Require().NoError(err)
	suite.db = db
	suite.mock = mock
}

func (suite *DBCountPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DBCountPublicTestSuite) TearDownTest() {
	suite.Require().NoError(suite.mock.ExpectationsWereMet())
}

func (suite *DBCountPublicTestSuite) TestCount() {
	tests := []struct {
		name        string
		setupMock   func()
		want        interface{}
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when Count Ok",
			setupMock: func() {
				query := regexp.QuoteMeta("SELECT COUNT(*) FROM goqite")
				suite.mock.ExpectQuery(query).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
			},
			want:    3,
			wantErr: false,
		},
		{
			name: "when Count fails to execute scan",
			setupMock: func() {
				query := regexp.QuoteMeta("SELECT COUNT(*) FROM goqite")
				suite.mock.ExpectQuery(query).
					WillReturnError(fmt.Errorf("query error"))
			},
			want:        0,
			wantErr:     true,
			wantErrType: fmt.Errorf("failed to count rows: query error"),
		},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			q := &queue.Queue{DB: suite.db}
			got, err := q.Count(context.Background())

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDBCountPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DBCountPublicTestSuite))
}
