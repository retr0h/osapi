// Copyright (c) 2026 John Dewey

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

package export_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/osapi-io/osapi/pkg/sdk/client"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/audit/export"
)

type FileExporterPublicTestSuite struct {
	suite.Suite

	ctx    context.Context
	tmpDir string
}

func (suite *FileExporterPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.tmpDir = suite.T().TempDir()
}

func (suite *FileExporterPublicTestSuite) newEntry(
	user string,
) client.AuditEntry {
	return client.AuditEntry{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:    time.Date(2026, 2, 21, 10, 30, 0, 0, time.UTC),
		User:         user,
		Roles:        []string{"admin"},
		Method:       "GET",
		Path:         "/node/hostname",
		SourceIP:     "127.0.0.1",
		ResponseCode: 200,
		DurationMs:   42,
	}
}

func (suite *FileExporterPublicTestSuite) TestOpenWriteClose() {
	tests := []struct {
		name         string
		entries      []client.AuditEntry
		validateFunc func(path string)
	}{
		{
			name:    "when single entry writes valid JSONL",
			entries: []client.AuditEntry{suite.newEntry("alice@example.com")},
			validateFunc: func(path string) {
				lines := suite.readLines(path)
				suite.Len(lines, 1)

				var entry client.AuditEntry
				err := json.Unmarshal([]byte(lines[0]), &entry)
				suite.NoError(err)
				suite.Equal("alice@example.com", entry.User)
			},
		},
		{
			name: "when multiple entries writes valid JSONL",
			entries: []client.AuditEntry{
				suite.newEntry("alice@example.com"),
				suite.newEntry("bob@example.com"),
				suite.newEntry("charlie@example.com"),
			},
			validateFunc: func(path string) {
				lines := suite.readLines(path)
				suite.Len(lines, 3)

				for i, user := range []string{"alice@example.com", "bob@example.com", "charlie@example.com"} {
					var entry client.AuditEntry
					err := json.Unmarshal([]byte(lines[i]), &entry)
					suite.NoError(err)
					suite.Equal(user, entry.User)
				}
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			path := filepath.Join(suite.tmpDir, tc.name+".jsonl")
			sut := export.NewFileExporter(path)

			err := sut.Open(suite.ctx)
			suite.Require().NoError(err)

			for _, entry := range tc.entries {
				err = sut.Write(suite.ctx, entry)
				suite.Require().NoError(err)
			}

			err = sut.Close(suite.ctx)
			suite.Require().NoError(err)

			tc.validateFunc(path)
		})
	}
}

func (suite *FileExporterPublicTestSuite) TestOpen() {
	tests := []struct {
		name         string
		path         string
		validateFunc func(err error)
	}{
		{
			name: "when path is invalid returns error",
			path: "/nonexistent/dir/file.jsonl",
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "opening export file")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			sut := export.NewFileExporter(tc.path)
			err := sut.Open(suite.ctx)
			tc.validateFunc(err)
		})
	}
}

func (suite *FileExporterPublicTestSuite) TestWrite() {
	tests := []struct {
		name         string
		setupWriter  func() (cleanup func())
		writeCount   int
		validateFunc func(err error)
	}{
		{
			name:        "when exporter not opened returns error",
			setupWriter: nil,
			writeCount:  1,
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "exporter not opened")
			},
		},
		{
			name: "when marshal fails returns error",
			setupWriter: func() (cleanup func()) {
				orig := export.MarshalJSONFunc()
				export.SetMarshalJSONFunc(func(_ any) ([]byte, error) {
					return nil, fmt.Errorf("marshal error")
				})
				return func() { export.SetMarshalJSONFunc(orig) }
			},
			writeCount: 1,
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "marshaling entry")
			},
		},
		{
			name: "when underlying writer fails on buffer flush returns write error",
			setupWriter: func() (cleanup func()) {
				orig := export.OpenFileFunc()
				export.SetOpenFileFunc(func(_ string) (io.WriteCloser, error) {
					return &failWriter{}, nil
				})
				return func() { export.SetOpenFileFunc(orig) }
			},
			// Each entry is ~200 bytes; 30 writes (~6KB) overflows
			// the 4096-byte bufio buffer, triggering a flush to the
			// underlying failWriter.
			writeCount: 30,
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "writ")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			sut := export.NewFileExporter("unused.jsonl")

			if tc.setupWriter != nil {
				cleanup := tc.setupWriter()
				defer cleanup()

				err := sut.Open(suite.ctx)
				suite.Require().NoError(err)
			}

			var writeErr error
			for i := range tc.writeCount {
				writeErr = sut.Write(suite.ctx, suite.newEntry(
					fmt.Sprintf("user%d@example.com", i),
				))
				if writeErr != nil {
					break
				}
			}
			tc.validateFunc(writeErr)
		})
	}
}

func (suite *FileExporterPublicTestSuite) TestWriteNewlineError() {
	entry := client.AuditEntry{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:    time.Date(2026, 2, 21, 10, 30, 0, 0, time.UTC),
		User:         "user@example.com",
		Roles:        []string{"admin"},
		Method:       "GET",
		Path:         "/node/hostname",
		SourceIP:     "127.0.0.1",
		ResponseCode: 200,
		DurationMs:   42,
	}

	tests := []struct {
		name         string
		validateFunc func(err error)
	}{
		{
			name: "when WriteByte triggers flush failure returns newline error",
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "writing newline")
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Marshal to get the exact byte size of the JSON data.
			data, err := json.Marshal(entry)
			suite.Require().NoError(err)

			// Create a bufio.Writer with a buffer sized exactly to the
			// JSON data. Write(data) fills the buffer completely, then
			// WriteByte('\n') finds Available()==0 and calls Flush(),
			// which hits the failing underlying writer.
			fw := &failWriter{}
			e := export.NewFileExporterWithWriter(bufio.NewWriterSize(fw, len(data)))

			err = e.Write(suite.ctx, entry)
			tt.validateFunc(err)
		})
	}
}

func (suite *FileExporterPublicTestSuite) TestClose() {
	tests := []struct {
		name         string
		setupWriter  func() (cleanup func())
		writeEntry   bool
		validateFunc func(err error)
	}{
		{
			name:        "when exporter not opened returns error",
			setupWriter: nil,
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "exporter not opened")
			},
		},
		{
			name: "when underlying writer fails on flush returns error",
			setupWriter: func() (cleanup func()) {
				orig := export.OpenFileFunc()
				export.SetOpenFileFunc(func(_ string) (io.WriteCloser, error) {
					return &failWriter{}, nil
				})
				return func() { export.SetOpenFileFunc(orig) }
			},
			writeEntry: true,
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "flushing writer")
			},
		},
		{
			name: "when file close fails returns error",
			setupWriter: func() (cleanup func()) {
				orig := export.OpenFileFunc()
				export.SetOpenFileFunc(func(_ string) (io.WriteCloser, error) {
					return &failCloseWriter{}, nil
				})
				return func() { export.SetOpenFileFunc(orig) }
			},
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "closing file")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			sut := export.NewFileExporter("unused.jsonl")

			if tc.setupWriter != nil {
				cleanup := tc.setupWriter()
				defer cleanup()

				err := sut.Open(suite.ctx)
				suite.Require().NoError(err)

				if tc.writeEntry {
					err = sut.Write(suite.ctx, suite.newEntry("test@example.com"))
					suite.Require().NoError(err)
				}
			}

			err := sut.Close(suite.ctx)
			tc.validateFunc(err)
		})
	}
}

func (suite *FileExporterPublicTestSuite) readLines(
	path string,
) []string {
	f, err := os.Open(path)
	suite.Require().NoError(err)
	defer func() { _ = f.Close() }()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	suite.Require().NoError(scanner.Err())

	return lines
}

func TestFileExporterPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FileExporterPublicTestSuite))
}

// failWriter is a writer that always returns an error on Write.
type failWriter struct{}

func (w *failWriter) Write(
	_ []byte,
) (int, error) {
	return 0, fmt.Errorf("write failed")
}

func (w *failWriter) Close() error {
	return nil
}

// failCloseWriter succeeds on Write but fails on Close.
type failCloseWriter struct{}

func (w *failCloseWriter) Write(
	p []byte,
) (int, error) {
	return len(p), nil
}

func (w *failCloseWriter) Close() error {
	return fmt.Errorf("close failed")
}
