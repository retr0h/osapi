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

package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/osapi-io/osapi/pkg/sdk/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/cli"
)

type UIPublicTestSuite struct {
	suite.Suite
}

func TestUIPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(UIPublicTestSuite))
}

func captureStdout(
	fn func(),
) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	return string(out)
}

func boolPtr(b bool) *bool { return &b }

func (suite *UIPublicTestSuite) TestBoolToSafeString() {
	tests := []struct {
		name string
		b    *bool
		want string
	}{
		{
			name: "when true returns true",
			b:    boolPtr(true),
			want: "true",
		},
		{
			name: "when false returns false",
			b:    boolPtr(false),
			want: "false",
		},
		{
			name: "when nil returns empty",
			b:    nil,
			want: "",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BoolToSafeString(tc.b)

			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestBuildBroadcastTable() {
	errMsg := "connection refused"

	tests := []struct {
		name         string
		results      []cli.ResultRow
		fieldHeaders []string
		wantHeaders  []string
		wantRows     [][]string
	}{
		{
			name:         "when no results returns hostname and status headers",
			results:      []cli.ResultRow{},
			fieldHeaders: nil,
			wantHeaders:  []string{"HOSTNAME", "STATUS"},
			wantRows:     [][]string{},
		},
		{
			name: "when all results succeed shows ok status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Fields: []string{"val1"}},
				{Hostname: "web-02", Fields: []string{"val2"}},
			},
			fieldHeaders: []string{"DATA"},
			wantHeaders:  []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "ok", "val1"},
				{"web-02", "ok", "val2"},
			},
		},
		{
			name: "when some results have errors shows err status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Status: "ok", Fields: []string{"val1"}},
				{Hostname: "web-02", Status: "failed", Error: &errMsg, Fields: []string{""}},
			},
			fieldHeaders: []string{"DATA"},
			wantHeaders:  []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "ok", "val1"},
				{"web-02", "err", ""},
			},
		},
		{
			name: "when skipped host shows skip status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Status: "ok", Fields: []string{"val1"}},
				{Hostname: "web-02", Status: "skipped", Error: &errMsg},
			},
			fieldHeaders: []string{"DATA"},
			wantHeaders:  []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "ok", "val1"},
				{"web-02", "skip"},
			},
		},
		{
			name: "when skipped host with fields is included in rows",
			results: []cli.ResultRow{
				{Hostname: "web-01", Status: "ok", Fields: []string{"val1"}},
				{Hostname: "web-02", Status: "skipped", Fields: []string{"partial"}},
			},
			fieldHeaders: []string{"DATA"},
			wantHeaders:  []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "ok", "val1"},
				{"web-02", "skip", "partial"},
			},
		},
		{
			name: "when single host shows hostname and status columns",
			results: []cli.ResultRow{
				{Hostname: "web-01", Fields: []string{"val1"}},
			},
			fieldHeaders: []string{"DATA"},
			wantHeaders:  []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "ok", "val1"},
			},
		},
		{
			name: "when all results have errors shows err status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Status: "failed", Error: &errMsg, Fields: []string{""}},
				{Hostname: "web-02", Status: "failed", Error: &errMsg, Fields: []string{""}},
			},
			fieldHeaders: []string{"DATA"},
			wantHeaders:  []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "err", ""},
				{"web-02", "err", ""},
			},
		},
		{
			name: "when changed is true shows changed status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Changed: boolPtr(true), Fields: []string{"val1"}},
				{Hostname: "web-02", Changed: boolPtr(false), Fields: []string{"val2"}},
			},
			fieldHeaders: []string{"DATA"},
			wantHeaders:  []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "changed", "val1"},
				{"web-02", "ok", "val2"},
			},
		},
		{
			name: "when changed and errors both present shows all rows",
			results: []cli.ResultRow{
				{
					Hostname: "web-01",
					Status:   "ok",
					Changed:  boolPtr(true),
					Fields:   []string{"val1"},
				},
				{
					Hostname: "web-02",
					Status:   "failed",
					Changed:  boolPtr(false),
					Error:    &errMsg,
					Fields:   []string{""},
				},
			},
			fieldHeaders: []string{"DATA"},
			wantHeaders:  []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "changed", "val1"},
				{"web-02", "err", ""},
			},
		},
		{
			name: "when no field headers shows only hostname and status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Changed: boolPtr(true)},
				{Hostname: "web-02"},
			},
			fieldHeaders: nil,
			wantHeaders:  []string{"HOSTNAME", "STATUS"},
			wantRows: [][]string{
				{"web-01", "changed"},
				{"web-02", "ok"},
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tr := cli.BuildBroadcastTable(tc.results, tc.fieldHeaders)

			assert.Equal(suite.T(), tc.wantHeaders, tr.Headers)
			assert.Equal(suite.T(), tc.wantRows, tr.Rows)
		})
	}
}

func (suite *UIPublicTestSuite) TestFormatLabels() {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{
			name:   "when nil returns empty",
			labels: nil,
			want:   "",
		},
		{
			name:   "when empty map returns empty",
			labels: map[string]string{},
			want:   "",
		},
		{
			name:   "when single label formats correctly",
			labels: map[string]string{"group": "web"},
			want:   "group:web",
		},
		{
			name:   "when multiple labels sorts by key",
			labels: map[string]string{"group": "web", "env": "prod", "az": "us-east"},
			want:   "az:us-east, env:prod, group:web",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := cli.FormatLabels(tc.labels)
			assert.Equal(suite.T(), tc.want, result)
		})
	}
}

func (suite *UIPublicTestSuite) TestBuildMutationTable() {
	errMsg := "interface not found"

	tests := []struct {
		name         string
		results      []cli.ResultRow
		fieldHeaders []string
		wantHeaders  []string
		wantRows     [][]string
	}{
		{
			name:         "when no results returns hostname and status headers",
			results:      []cli.ResultRow{},
			fieldHeaders: nil,
			wantHeaders:  []string{"HOSTNAME", "STATUS"},
			wantRows:     [][]string{},
		},
		{
			name: "when all succeed shows ok status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Status: "ok"},
				{Hostname: "web-02", Status: "ok"},
			},
			fieldHeaders: nil,
			wantHeaders:  []string{"HOSTNAME", "STATUS"},
			wantRows: [][]string{
				{"web-01", "ok"},
				{"web-02", "ok"},
			},
		},
		{
			name: "when some fail shows err status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Status: "ok"},
				{Hostname: "web-02", Status: "failed", Error: &errMsg},
			},
			fieldHeaders: nil,
			wantHeaders:  []string{"HOSTNAME", "STATUS"},
			wantRows: [][]string{
				{"web-01", "ok"},
				{"web-02", "err"},
			},
		},
		{
			name: "when single host shows hostname and status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Status: "ok", Fields: []string{"extra"}},
			},
			fieldHeaders: []string{"DETAIL"},
			wantHeaders:  []string{"HOSTNAME", "STATUS", "DETAIL"},
			wantRows: [][]string{
				{"web-01", "ok", "extra"},
			},
		},
		{
			name: "when changed is true shows changed status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Status: "ok", Changed: boolPtr(true)},
				{Hostname: "web-02", Status: "ok", Changed: boolPtr(true)},
			},
			fieldHeaders: nil,
			wantHeaders:  []string{"HOSTNAME", "STATUS"},
			wantRows: [][]string{
				{"web-01", "changed"},
				{"web-02", "changed"},
			},
		},
		{
			name: "when changed is false shows ok status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Status: "ok", Changed: boolPtr(false)},
				{Hostname: "web-02", Status: "ok", Changed: boolPtr(false)},
			},
			fieldHeaders: nil,
			wantHeaders:  []string{"HOSTNAME", "STATUS"},
			wantRows: [][]string{
				{"web-01", "ok"},
				{"web-02", "ok"},
			},
		},
		{
			name: "when field headers provided includes them after status",
			results: []cli.ResultRow{
				{Hostname: "web-01", Changed: boolPtr(true), Fields: []string{"val1"}},
				{Hostname: "web-02", Fields: []string{"val2"}},
			},
			fieldHeaders: []string{"DATA"},
			wantHeaders:  []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "changed", "val1"},
				{"web-02", "ok", "val2"},
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tr := cli.BuildMutationTable(tc.results, tc.fieldHeaders)

			assert.Equal(suite.T(), tc.wantHeaders, tr.Headers)
			assert.Equal(suite.T(), tc.wantRows, tr.Rows)
		})
	}
}

func (suite *UIPublicTestSuite) TestBuildBroadcastTableResult() {
	errMsg := "connection refused"
	skipMsg := "unsupported"

	tests := []struct {
		name        string
		results     []cli.ResultRow
		fieldHdrs   []string
		wantHeaders []string
		wantRows    [][]string
		wantErrors  []cli.ErrorEntry
	}{
		{
			name: "when errors exist they appear in errors field",
			results: []cli.ResultRow{
				{Hostname: "web-01", Fields: []string{"val1"}},
				{Hostname: "web-02", Error: &errMsg, Fields: []string{""}},
			},
			fieldHdrs:   []string{"DATA"},
			wantHeaders: []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "ok", "val1"},
				{"web-02", "err", ""},
			},
			wantErrors: []cli.ErrorEntry{
				{Hostname: "web-02", Message: "connection refused", Status: "err"},
			},
		},
		{
			name: "when no errors the errors field is nil",
			results: []cli.ResultRow{
				{Hostname: "web-01", Fields: []string{"val1"}},
				{Hostname: "web-02", Fields: []string{"val2"}},
			},
			fieldHdrs:   []string{"DATA"},
			wantHeaders: []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "ok", "val1"},
				{"web-02", "ok", "val2"},
			},
			wantErrors: nil,
		},
		{
			name: "when skipped host without fields appears in errors",
			results: []cli.ResultRow{
				{Hostname: "web-01", Fields: []string{"val1"}},
				{Hostname: "web-02", Status: "skipped", Error: &skipMsg},
			},
			fieldHdrs:   []string{"DATA"},
			wantHeaders: []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "ok", "val1"},
				{"web-02", "skip"},
			},
			wantErrors: []cli.ErrorEntry{
				{Hostname: "web-02", Message: "unsupported", Status: "skip"},
			},
		},
		{
			name: "when multiple errors collects all",
			results: []cli.ResultRow{
				{Hostname: "web-01", Error: &errMsg},
				{Hostname: "web-02", Error: &skipMsg},
				{Hostname: "web-03", Fields: []string{"ok"}},
			},
			fieldHdrs:   []string{"DATA"},
			wantHeaders: []string{"HOSTNAME", "STATUS", "DATA"},
			wantRows: [][]string{
				{"web-01", "err"},
				{"web-02", "err"},
				{"web-03", "ok", "ok"},
			},
			wantErrors: []cli.ErrorEntry{
				{Hostname: "web-01", Message: "connection refused", Status: "err"},
				{Hostname: "web-02", Message: "unsupported", Status: "err"},
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := cli.BuildBroadcastTable(tc.results, tc.fieldHdrs)

			assert.Equal(suite.T(), tc.wantHeaders, result.Headers)
			assert.Equal(suite.T(), tc.wantRows, result.Rows)
			assert.Equal(suite.T(), tc.wantErrors, result.Errors)
		})
	}
}

func (suite *UIPublicTestSuite) TestBuildMutationTableResult() {
	errMsg := "permission denied"

	tests := []struct {
		name        string
		results     []cli.ResultRow
		fieldHdrs   []string
		wantHeaders []string
		wantRows    [][]string
		wantErrors  []cli.ErrorEntry
	}{
		{
			name: "when mutation errors exist they appear in errors field",
			results: []cli.ResultRow{
				{Hostname: "web-01", Changed: boolPtr(true)},
				{Hostname: "web-02", Error: &errMsg},
			},
			fieldHdrs:   nil,
			wantHeaders: []string{"HOSTNAME", "STATUS"},
			wantRows: [][]string{
				{"web-01", "changed"},
				{"web-02", "err"},
			},
			wantErrors: []cli.ErrorEntry{
				{Hostname: "web-02", Message: "permission denied", Status: "err"},
			},
		},
		{
			name: "when no mutation errors the errors field is nil",
			results: []cli.ResultRow{
				{Hostname: "web-01", Changed: boolPtr(true)},
				{Hostname: "web-02", Changed: boolPtr(false)},
			},
			fieldHdrs:   nil,
			wantHeaders: []string{"HOSTNAME", "STATUS"},
			wantRows: [][]string{
				{"web-01", "changed"},
				{"web-02", "ok"},
			},
			wantErrors: nil,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := cli.BuildMutationTable(tc.results, tc.fieldHdrs)

			assert.Equal(suite.T(), tc.wantHeaders, result.Headers)
			assert.Equal(suite.T(), tc.wantRows, result.Rows)
			assert.Equal(suite.T(), tc.wantErrors, result.Errors)
		})
	}
}

func (suite *UIPublicTestSuite) TestFormatList() {
	tests := []struct {
		name string
		list []string
		want string
	}{
		{
			name: "when empty returns None",
			list: []string{},
			want: "None",
		},
		{
			name: "when single item returns it",
			list: []string{"alpha"},
			want: "alpha",
		},
		{
			name: "when multiple items joins with comma",
			list: []string{"alpha", "beta", "gamma"},
			want: "alpha, beta, gamma",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.FormatList(tc.list)

			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestCalculateColumnWidths() {
	tests := []struct {
		name       string
		headers    []string
		rows       [][]string
		minPadding int
		want       []int
	}{
		{
			name:       "when empty headers returns empty",
			headers:    []string{},
			rows:       nil,
			minPadding: 1,
			want:       []int{},
		},
		{
			name:       "when headers wider than rows uses header width",
			headers:    []string{"HOSTNAME", "STATUS"},
			rows:       [][]string{{"a", "b"}},
			minPadding: 1,
			want:       []int{10, 8},
		},
		{
			name:       "when rows wider than headers uses row width",
			headers:    []string{"A", "B"},
			rows:       [][]string{{"longvalue", "anotherlongvalue"}},
			minPadding: 1,
			want:       []int{11, 18},
		},
		{
			name:       "when multi-line content uses longest line width",
			headers:    []string{"DATA"},
			rows:       [][]string{{"short\nvery long line here"}},
			minPadding: 0,
			want:       []int{19},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.CalculateColumnWidths(tc.headers, tc.rows, tc.minPadding)

			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestGetMaxLineWidth() {
	tests := []struct {
		name string
		text string
		want int
	}{
		{
			name: "when single line returns its length",
			text: "hello",
			want: 5,
		},
		{
			name: "when multi-line returns longest",
			text: "short\na much longer line\nmed",
			want: 18,
		},
		{
			name: "when empty returns zero",
			text: "",
			want: 0,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.GetMaxLineWidth(tc.text)

			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestSafeString() {
	str := "hello"

	tests := []struct {
		name string
		s    *string
		want string
	}{
		{
			name: "when non-nil returns value",
			s:    &str,
			want: "hello",
		},
		{
			name: "when nil returns empty",
			s:    nil,
			want: "",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.SafeString(tc.s)

			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestSafeUUID() {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name string
		u    *uuid.UUID
		want string
	}{
		{
			name: "when non-nil returns string",
			u:    &id,
			want: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "when nil returns empty",
			u:    nil,
			want: "",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.SafeUUID(tc.u)

			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestFloat64ToSafeString() {
	val := 3.14

	tests := []struct {
		name string
		f    *float64
		want string
	}{
		{
			name: "when non-nil returns formatted float",
			f:    &val,
			want: "3.140000",
		},
		{
			name: "when nil returns N/A",
			f:    nil,
			want: "N/A",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.Float64ToSafeString(tc.f)

			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestIntToSafeString() {
	val := 42

	tests := []struct {
		name string
		i    *int
		want string
	}{
		{
			name: "when non-nil returns formatted int",
			i:    &val,
			want: "42",
		},
		{
			name: "when nil returns N/A",
			i:    nil,
			want: "N/A",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.IntToSafeString(tc.i)

			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestHandleError() {
	tests := []struct {
		name      string
		err       error
		wantInLog string
	}{
		{
			name: "when auth error logs api error with status code",
			err: &client.AuthError{
				APIError: client.APIError{StatusCode: 403, Message: "insufficient permissions"},
			},
			wantInLog: "insufficient permissions",
		},
		{
			name: "when not found error logs api error with status code",
			err: &client.NotFoundError{
				APIError: client.APIError{StatusCode: 404, Message: "job not found"},
			},
			wantInLog: "job not found",
		},
		{
			name: "when validation error logs api error with status code",
			err: &client.ValidationError{
				APIError: client.APIError{StatusCode: 400, Message: "invalid input"},
			},
			wantInLog: "invalid input",
		},
		{
			name: "when server error logs api error with status code",
			err: &client.ServerError{
				APIError: client.APIError{StatusCode: 500, Message: "internal server error"},
			},
			wantInLog: "internal server error",
		},
		{
			name:      "when generic error logs error message",
			err:       fmt.Errorf("connection refused"),
			wantInLog: "connection refused",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			// Prevent os.Exit from killing the test process.
			var exitCode int
			cli.SetOsExit(func(code int) { exitCode = code })
			defer cli.ResetOsExit()

			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))

			cli.HandleError(tc.err, logger)

			assert.Contains(suite.T(), buf.String(), tc.wantInLog)
			assert.Equal(suite.T(), 1, exitCode)
		})
	}
}

func (suite *UIPublicTestSuite) TestPrintKV() {
	tests := []struct {
		name       string
		pairs      []string
		wantOutput bool
	}{
		{
			name:       "when valid pairs prints output",
			pairs:      []string{"Key", "Value"},
			wantOutput: true,
		},
		{
			name:       "when multiple pairs prints all",
			pairs:      []string{"Name", "test", "Status", "ok"},
			wantOutput: true,
		},
		{
			name:       "when odd number of pairs prints nothing",
			pairs:      []string{"Key"},
			wantOutput: false,
		},
		{
			name:       "when empty prints nothing",
			pairs:      []string{},
			wantOutput: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			output := captureStdout(func() {
				cli.PrintKV(tc.pairs...)
			})

			if tc.wantOutput {
				assert.NotEmpty(suite.T(), output)
			} else {
				assert.Empty(suite.T(), output)
			}
		})
	}
}

func (suite *UIPublicTestSuite) TestPrintCompactTable() {
	tests := []struct {
		name        string
		sections    []cli.Section
		wantHeaders []string
	}{
		{
			name: "when section with title renders table",
			sections: []cli.Section{
				{
					Title:   "Test",
					Headers: []string{"COL1", "COL2"},
					Rows:    [][]string{{"a", "b"}},
				},
			},
			wantHeaders: []string{"COL1", "COL2"},
		},
		{
			name: "when section without title renders table",
			sections: []cli.Section{
				{
					Headers: []string{"COL1"},
					Rows:    [][]string{{"a"}},
				},
			},
			wantHeaders: []string{"COL1"},
		},
		{
			name: "when wide data aligns columns",
			sections: []cli.Section{
				{
					Title:   "Wide",
					Headers: []string{"A", "B", "C", "D", "E"},
					Rows: [][]string{{
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
						"ccccccccccccccccccccccccccccccc",
						"ddddddddddddddddddddddddddddd",
						"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
					}},
				},
			},
			wantHeaders: []string{"A", "B", "C", "D", "E"},
		},
		{
			name: "when many columns renders all headers",
			sections: []cli.Section{
				{
					Headers: []string{
						"X", "Y", "Z",
						"LONG-HEADER-1", "LONG-HEADER-2",
					},
					Rows: [][]string{{
						"a", "b", "c",
						"aaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbb",
					}},
				},
			},
			wantHeaders: []string{"X", "Y", "Z", "LONG-HEADER-1", "LONG-HEADER-2"},
		},
		{
			name: "when multiple rows alternates colors",
			sections: []cli.Section{
				{
					Headers: []string{"NAME", "VALUE"},
					Rows: [][]string{
						{"row0", "even"},
						{"row1", "odd"},
						{"row2", "even"},
					},
				},
			},
			wantHeaders: []string{"NAME", "VALUE"},
		},
		{
			name: "when column exceeds max width truncates to cap",
			sections: []cli.Section{
				{
					Headers: []string{"A", "B"},
					Rows: [][]string{{
						strings.Repeat("x", 90),
						"short",
					}},
				},
			},
			wantHeaders: []string{"A", "B"},
		},
		{
			name: "when cell exceeds column width shows ellipsis",
			sections: []cli.Section{
				{
					Headers: []string{"A"},
					Rows: [][]string{
						{strings.Repeat("x", 90)},
					},
				},
			},
			wantHeaders: []string{"A"},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			output := captureStdout(func() {
				cli.PrintCompactTable(tc.sections)
			})

			assert.NotEmpty(suite.T(), output)
			for _, h := range tc.wantHeaders {
				assert.Contains(suite.T(), output, h)
			}
		})
	}
}

func (suite *UIPublicTestSuite) TestFormatAge() {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{
			name: "when zero returns empty",
			d:    0,
			want: "",
		},
		{
			name: "when negative returns empty",
			d:    -1 * time.Hour,
			want: "",
		},
		{
			name: "when days and hours formats as days",
			d:    3*24*time.Hour + 4*time.Hour,
			want: "3d 4h",
		},
		{
			name: "when hours and minutes formats as hours",
			d:    12*time.Hour + 30*time.Minute,
			want: "12h 30m",
		},
		{
			name: "when only minutes formats as minutes",
			d:    45 * time.Minute,
			want: "45m",
		},
		{
			name: "when only seconds formats as seconds",
			d:    30 * time.Second,
			want: "30s",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.FormatAge(tc.d)

			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestFormatBytes() {
	tests := []struct {
		name string
		b    int
		want string
	}{
		{
			name: "when bytes returns bytes",
			b:    512,
			want: "512 B",
		},
		{
			name: "when kilobytes returns KB",
			b:    5 * 1024,
			want: "5.0 KB",
		},
		{
			name: "when megabytes returns MB",
			b:    3 * 1024 * 1024,
			want: "3.0 MB",
		},
		{
			name: "when gigabytes returns GB",
			b:    2 * 1024 * 1024 * 1024,
			want: "2.0 GB",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.FormatBytes(tc.b)

			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestDisplayJobDetail() {
	tests := []struct {
		name string
		resp *client.JobDetail
	}{
		{
			name: "when minimal response displays job info",
			resp: &client.JobDetail{
				Status: "completed",
			},
		},
		{
			name: "when full response displays all sections",
			resp: &client.JobDetail{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				Status:    "completed",
				Hostname:  "web-01",
				Created:   "2026-01-01T00:00:00Z",
				UpdatedAt: "2026-01-01T00:01:00Z",
				Error:     "timeout",
				Operation: map[string]any{"type": "node.hostname"},
				Result:    map[string]any{"hostname": "web-01"},
				Timeline: []client.TimelineEvent{
					{
						Event:     "completed",
						Timestamp: "2026-01-01T00:01:00Z",
						Hostname:  "web-01",
						Message:   "job completed",
					},
				},
				AgentStates: map[string]client.AgentState{
					"web-01": {
						Status:   "completed",
						Duration: "1.5s",
					},
				},
				Responses: map[string]client.AgentJobResponse{
					"web-01": {
						Status: "ok",
						Data:   map[string]string{"key": "val"},
					},
				},
			},
		},
		{
			name: "when agent states with multiple agents shows summary",
			resp: &client.JobDetail{
				Status: "completed",
				AgentStates: map[string]client.AgentState{
					"web-01": {Status: "completed", Duration: "1s"},
					"web-02": {Status: "failed", Duration: "1s", Error: "error"},
					"web-03": {Status: "started", Duration: "1s"},
				},
			},
		},
		{
			name: "when response has nil data shows no data placeholder",
			resp: &client.JobDetail{
				Status: "completed",
				Responses: map[string]client.AgentJobResponse{
					"web-01": {
						Status: "ok",
						Data:   nil,
					},
				},
			},
		},
		{
			name: "when response has error shows error message",
			resp: &client.JobDetail{
				Status: "failed",
				Responses: map[string]client.AgentJobResponse{
					"web-01": {
						Status: "failed",
						Error:  "timeout",
					},
				},
			},
		},
		{
			name: "when timeline has error shows error message",
			resp: &client.JobDetail{
				Status: "failed",
				Timeline: []client.TimelineEvent{
					{
						Event:     "failed",
						Timestamp: "2026-01-01T00:01:00Z",
						Error:     "connection refused",
					},
				},
			},
		},
		{
			name: "when agent states contain skipped status",
			resp: &client.JobDetail{
				Status: "completed",
				AgentStates: map[string]client.AgentState{
					"web-01": {Status: "completed", Duration: "1s"},
					"web-02": {Status: "skipped", Duration: "0s"},
					"web-03": {Status: "completed", Duration: "2s"},
				},
			},
		},
		{
			name: "when multiple agents skipped shows skipped in summary",
			resp: &client.JobDetail{
				Status: "skipped",
				AgentStates: map[string]client.AgentState{
					"web-01": {Status: "skipped"},
					"web-02": {Status: "skipped"},
					"web-03": {Status: "completed", Duration: "1s"},
				},
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			output := captureStdout(func() {
				cli.DisplayJobDetail(tc.resp)
			})

			assert.NotEmpty(suite.T(), output)
		})
	}
}

func (suite *UIPublicTestSuite) TestStatusWeight() {
	tests := []struct {
		name   string
		status string
		want   int
	}{
		{name: "when ok returns 0", status: "ok", want: 0},
		{name: "when changed returns 1", status: "changed", want: 1},
		{name: "when skip returns 2", status: "skip", want: 2},
		{name: "when err returns 3", status: "err", want: 3},
		{name: "when unknown returns 0", status: "unknown", want: 0},
		{name: "when empty returns 0", status: "", want: 0},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.ExportStatusWeight(tc.status)
			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestResolveStatus() {
	errMsg := "failed"

	tests := []struct {
		name string
		row  cli.ResultRow
		want string
	}{
		{
			name: "when no error and no change returns ok",
			row:  cli.ResultRow{Hostname: "h1"},
			want: "ok",
		},
		{
			name: "when changed true returns changed",
			row:  cli.ResultRow{Hostname: "h1", Changed: boolPtr(true)},
			want: "changed",
		},
		{
			name: "when changed false returns ok",
			row:  cli.ResultRow{Hostname: "h1", Changed: boolPtr(false)},
			want: "ok",
		},
		{
			name: "when error returns err",
			row:  cli.ResultRow{Hostname: "h1", Error: &errMsg},
			want: "err",
		},
		{
			name: "when status skipped returns skip even with error",
			row:  cli.ResultRow{Hostname: "h1", Status: "skipped", Error: &errMsg},
			want: "skip",
		},
		{
			name: "when status skip returns skip",
			row:  cli.ResultRow{Hostname: "h1", Status: "skip"},
			want: "skip",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.ExportResolveStatus(tc.row)
			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

func (suite *UIPublicTestSuite) TestPrintSummary() {
	tests := []struct {
		name         string
		section      cli.Section
		validateFunc func(output string)
	}{
		{
			name: "when mixed statuses shows counts",
			section: cli.Section{
				Rows: [][]string{
					{"web-01", "ok", "val"},
					{"web-02", "skip"},
					{"web-03", "err"},
				},
			},
			validateFunc: func(output string) {
				assert.Contains(suite.T(), output, "3 hosts:")
				assert.Contains(suite.T(), output, "1 ok")
				assert.Contains(suite.T(), output, "1 skipped")
				assert.Contains(suite.T(), output, "1 failed")
			},
		},
		{
			name: "when all ok shows only ok count",
			section: cli.Section{
				Rows: [][]string{
					{"web-01", "ok", "val"},
					{"web-02", "ok", "val"},
				},
			},
			validateFunc: func(output string) {
				assert.Contains(suite.T(), output, "2 hosts:")
				assert.Contains(suite.T(), output, "2 ok")
			},
		},
		{
			name: "when changed shows changed count",
			section: cli.Section{
				Rows: [][]string{
					{"web-01", "changed"},
					{"web-02", "ok"},
				},
			},
			validateFunc: func(output string) {
				assert.Contains(suite.T(), output, "2 hosts:")
				assert.Contains(suite.T(), output, "1 changed")
				assert.Contains(suite.T(), output, "1 ok")
			},
		},
		{
			name: "when duration set shows duration",
			section: cli.Section{
				Duration: "286ms",
				Rows: [][]string{
					{"web-01", "ok"},
				},
			},
			validateFunc: func(output string) {
				assert.Contains(suite.T(), output, "286ms")
			},
		},
		{
			name: "when no rows does not print summary line",
			section: cli.Section{
				Rows: [][]string{},
			},
			validateFunc: func(output string) {
				assert.NotContains(suite.T(), output, "hosts:")
			},
		},
		{
			name: "when error host not in rows counts from errors",
			section: cli.Section{
				Rows: [][]string{
					{"web-01", "ok"},
				},
				Errors: []cli.ErrorEntry{
					{Hostname: "web-02", Message: "timeout", Status: "err"},
				},
			},
			validateFunc: func(output string) {
				assert.Contains(suite.T(), output, "2 hosts:")
				assert.Contains(suite.T(), output, "1 ok")
				assert.Contains(suite.T(), output, "1 failed")
			},
		},
		{
			name: "when same host has multiple rows uses worst status",
			section: cli.Section{
				Rows: [][]string{
					{"web-01", "ok", "val1"},
					{"web-01", "err"},
				},
			},
			validateFunc: func(output string) {
				assert.Contains(suite.T(), output, "1 hosts:")
				assert.Contains(suite.T(), output, "1 failed")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			output := captureStdout(func() {
				cli.PrintCompactTable([]cli.Section{tc.section})
			})
			tc.validateFunc(output)
		})
	}
}

func (suite *UIPublicTestSuite) TestPrintErrors() {
	tests := []struct {
		name         string
		errors       []cli.ErrorEntry
		validateFunc func(output string)
	}{
		{
			name:   "when no errors prints nothing",
			errors: nil,
			validateFunc: func(output string) {
				assert.Empty(suite.T(), output)
			},
		},
		{
			name: "when error shows red details",
			errors: []cli.ErrorEntry{
				{Hostname: "web-01", Message: "connection refused", Status: "err"},
			},
			validateFunc: func(output string) {
				assert.Contains(suite.T(), output, "Details:")
				assert.Contains(suite.T(), output, "web-01")
				assert.Contains(suite.T(), output, "connection refused")
			},
		},
		{
			name: "when skip shows details",
			errors: []cli.ErrorEntry{
				{Hostname: "mac-01", Message: "unsupported OS", Status: "skip"},
			},
			validateFunc: func(output string) {
				assert.Contains(suite.T(), output, "Details:")
				assert.Contains(suite.T(), output, "mac-01")
				assert.Contains(suite.T(), output, "unsupported OS")
			},
		},
		{
			name: "when multiple entries shows all",
			errors: []cli.ErrorEntry{
				{Hostname: "web-01", Message: "timeout", Status: "err"},
				{Hostname: "mac-01", Message: "unsupported", Status: "skip"},
			},
			validateFunc: func(output string) {
				assert.Contains(suite.T(), output, "web-01")
				assert.Contains(suite.T(), output, "mac-01")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			output := captureStdout(func() {
				cli.PrintErrors(tc.errors)
			})
			tc.validateFunc(output)
		})
	}
}
