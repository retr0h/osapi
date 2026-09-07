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
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/osapi-io/osapi/pkg/sdk/client"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/audit/export"
	exportmocks "github.com/osapi-io/osapi/internal/audit/export/mocks"
)

type ExportPublicTestSuite struct {
	suite.Suite

	ctx    context.Context
	logger *slog.Logger
}

func (suite *ExportPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.logger = slog.Default()
}

func (suite *ExportPublicTestSuite) newEntry(
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

func (suite *ExportPublicTestSuite) TestRun() {
	tests := []struct {
		name          string
		fetcher       export.Fetcher
		setupExporter func(
			ctrl *gomock.Controller,
			opened *bool,
			closed *bool,
			entries *[]client.AuditEntry,
		) *exportmocks.MockExporter
		batchSize    int
		validateFunc func(
			opened bool,
			closed bool,
			entries []client.AuditEntry,
			result *export.Result,
			err error,
		)
	}{
		{
			name: "when no entries returns zero counts",
			fetcher: func(_ context.Context, _, _ int) ([]client.AuditEntry, int, error) {
				return nil, 0, nil
			},
			setupExporter: func(
				ctrl *gomock.Controller,
				opened *bool,
				closed *bool,
				_ *[]client.AuditEntry,
			) *exportmocks.MockExporter {
				m := exportmocks.NewMockExporter(ctrl)
				m.EXPECT().Open(gomock.Any()).DoAndReturn(func(_ context.Context) error {
					*opened = true
					return nil
				})
				m.EXPECT().Close(gomock.Any()).DoAndReturn(func(_ context.Context) error {
					*closed = true
					return nil
				})
				return m
			},
			batchSize: 100,
			validateFunc: func(
				opened bool,
				closed bool,
				_ []client.AuditEntry,
				result *export.Result,
				err error,
			) {
				suite.NoError(err)
				suite.Equal(0, result.TotalEntries)
				suite.Equal(0, result.ExportedEntries)
				suite.True(opened)
				suite.True(closed)
			},
		},
		{
			name: "when single page exports all entries",
			fetcher: func(_ context.Context, _, _ int) ([]client.AuditEntry, int, error) {
				return []client.AuditEntry{
					suite.newEntry("alice@example.com"),
					suite.newEntry("bob@example.com"),
				}, 2, nil
			},
			setupExporter: func(
				ctrl *gomock.Controller,
				_ *bool,
				_ *bool,
				entries *[]client.AuditEntry,
			) *exportmocks.MockExporter {
				m := exportmocks.NewMockExporter(ctrl)
				m.EXPECT().Open(gomock.Any()).Return(nil)
				m.EXPECT().Write(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, entry client.AuditEntry) error {
						*entries = append(*entries, entry)
						return nil
					},
				).Times(2)
				m.EXPECT().Close(gomock.Any()).Return(nil)
				return m
			},
			batchSize: 100,
			validateFunc: func(
				_ bool,
				_ bool,
				entries []client.AuditEntry,
				result *export.Result,
				err error,
			) {
				suite.NoError(err)
				suite.Equal(2, result.TotalEntries)
				suite.Equal(2, result.ExportedEntries)
				suite.Len(entries, 2)
				suite.Equal("alice@example.com", entries[0].User)
				suite.Equal("bob@example.com", entries[1].User)
			},
		},
		{
			name: "when multi-page paginates correctly",
			fetcher: newPagedFetcher([][]client.AuditEntry{
				{suite.newEntry("alice@example.com"), suite.newEntry("bob@example.com")},
				{suite.newEntry("charlie@example.com")},
			}, 3),
			setupExporter: func(
				ctrl *gomock.Controller,
				_ *bool,
				_ *bool,
				entries *[]client.AuditEntry,
			) *exportmocks.MockExporter {
				m := exportmocks.NewMockExporter(ctrl)
				m.EXPECT().Open(gomock.Any()).Return(nil)
				m.EXPECT().Write(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, entry client.AuditEntry) error {
						*entries = append(*entries, entry)
						return nil
					},
				).Times(3)
				m.EXPECT().Close(gomock.Any()).Return(nil)
				return m
			},
			batchSize: 2,
			validateFunc: func(
				_ bool,
				_ bool,
				entries []client.AuditEntry,
				result *export.Result,
				err error,
			) {
				suite.NoError(err)
				suite.Equal(3, result.TotalEntries)
				suite.Equal(3, result.ExportedEntries)
				suite.Len(entries, 3)
			},
		},
		{
			name: "when fetcher errors returns partial result",
			fetcher: func(_ context.Context, _, offset int) ([]client.AuditEntry, int, error) {
				if offset > 0 {
					return nil, 0, fmt.Errorf("connection lost")
				}
				return []client.AuditEntry{suite.newEntry("alice@example.com")}, 3, nil
			},
			setupExporter: func(
				ctrl *gomock.Controller,
				_ *bool,
				_ *bool,
				_ *[]client.AuditEntry,
			) *exportmocks.MockExporter {
				m := exportmocks.NewMockExporter(ctrl)
				m.EXPECT().Open(gomock.Any()).Return(nil)
				m.EXPECT().Write(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().Close(gomock.Any()).Return(nil)
				return m
			},
			batchSize: 1,
			validateFunc: func(
				_ bool,
				_ bool,
				_ []client.AuditEntry,
				result *export.Result,
				err error,
			) {
				suite.Error(err)
				suite.Contains(err.Error(), "fetching entries at offset 1")
				suite.Contains(err.Error(), "connection lost")
				suite.Equal(1, result.ExportedEntries)
				suite.Equal(3, result.TotalEntries)
			},
		},
		{
			name: "when write errors returns partial result",
			fetcher: func(_ context.Context, _, _ int) ([]client.AuditEntry, int, error) {
				return []client.AuditEntry{suite.newEntry("alice@example.com")}, 1, nil
			},
			setupExporter: func(
				ctrl *gomock.Controller,
				_ *bool,
				_ *bool,
				_ *[]client.AuditEntry,
			) *exportmocks.MockExporter {
				m := exportmocks.NewMockExporter(ctrl)
				m.EXPECT().Open(gomock.Any()).Return(nil)
				m.EXPECT().Write(gomock.Any(), gomock.Any()).Return(fmt.Errorf("disk full"))
				m.EXPECT().Close(gomock.Any()).Return(nil)
				return m
			},
			batchSize: 100,
			validateFunc: func(
				_ bool,
				_ bool,
				_ []client.AuditEntry,
				result *export.Result,
				err error,
			) {
				suite.Error(err)
				suite.Contains(err.Error(), "writing entry")
				suite.Equal(0, result.ExportedEntries)
			},
		},
		{
			name: "when open errors returns nil result",
			fetcher: func(_ context.Context, _, _ int) ([]client.AuditEntry, int, error) {
				return nil, 0, nil
			},
			setupExporter: func(
				ctrl *gomock.Controller,
				_ *bool,
				_ *bool,
				_ *[]client.AuditEntry,
			) *exportmocks.MockExporter {
				m := exportmocks.NewMockExporter(ctrl)
				m.EXPECT().Open(gomock.Any()).Return(fmt.Errorf("permission denied"))
				return m
			},
			batchSize: 100,
			validateFunc: func(
				_ bool,
				_ bool,
				_ []client.AuditEntry,
				result *export.Result,
				err error,
			) {
				suite.Error(err)
				suite.Contains(err.Error(), "opening exporter")
				suite.Nil(result)
			},
		},
		{
			name: "when close errors logs warning but returns result",
			fetcher: func(_ context.Context, _, _ int) ([]client.AuditEntry, int, error) {
				return nil, 0, nil
			},
			setupExporter: func(
				ctrl *gomock.Controller,
				_ *bool,
				_ *bool,
				_ *[]client.AuditEntry,
			) *exportmocks.MockExporter {
				m := exportmocks.NewMockExporter(ctrl)
				m.EXPECT().Open(gomock.Any()).Return(nil)
				m.EXPECT().Close(gomock.Any()).Return(fmt.Errorf("close failed"))
				return m
			},
			batchSize: 100,
			validateFunc: func(
				_ bool,
				_ bool,
				_ []client.AuditEntry,
				result *export.Result,
				err error,
			) {
				suite.NoError(err)
				suite.Equal(0, result.TotalEntries)
				suite.Equal(0, result.ExportedEntries)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ctrl := gomock.NewController(suite.T())
			var opened, closed bool
			var entries []client.AuditEntry
			mockExp := tc.setupExporter(ctrl, &opened, &closed, &entries)

			result, err := export.Run(
				suite.ctx,
				suite.logger,
				tc.fetcher,
				mockExp,
				tc.batchSize,
				nil,
			)
			tc.validateFunc(opened, closed, entries, result, err)
		})
	}
}

func (suite *ExportPublicTestSuite) TestRunProgress() {
	tests := []struct {
		name         string
		fetcher      export.Fetcher
		batchSize    int
		validateFunc func(calls []progressCall)
	}{
		{
			name: "when multi-page calls progress after each batch",
			fetcher: newPagedFetcher([][]client.AuditEntry{
				{suite.newEntry("alice@example.com"), suite.newEntry("bob@example.com")},
				{suite.newEntry("charlie@example.com")},
			}, 3),
			batchSize: 2,
			validateFunc: func(calls []progressCall) {
				suite.Require().Len(calls, 2)
				suite.Equal(2, calls[0].exported)
				suite.Equal(3, calls[0].total)
				suite.Equal(3, calls[1].exported)
				suite.Equal(3, calls[1].total)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ctrl := gomock.NewController(suite.T())
			m := exportmocks.NewMockExporter(ctrl)
			m.EXPECT().Open(gomock.Any()).Return(nil)
			m.EXPECT().Write(gomock.Any(), gomock.Any()).Return(nil).Times(3)
			m.EXPECT().Close(gomock.Any()).Return(nil)

			var calls []progressCall
			onProgress := func(exported int, total int) {
				calls = append(calls, progressCall{exported: exported, total: total})
			}

			_, err := export.Run(
				suite.ctx,
				suite.logger,
				tc.fetcher,
				m,
				tc.batchSize,
				onProgress,
			)
			suite.NoError(err)
			tc.validateFunc(calls)
		})
	}
}

func TestExportPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ExportPublicTestSuite))
}

type progressCall struct {
	exported int
	total    int
}

// newPagedFetcher creates a fetcher that returns pages of entries based on offset.
func newPagedFetcher(
	pages [][]client.AuditEntry,
	total int,
) export.Fetcher {
	return func(
		_ context.Context,
		limit int,
		offset int,
	) ([]client.AuditEntry, int, error) {
		_ = limit
		pageIdx := 0
		remaining := offset
		for pageIdx < len(pages) && remaining >= len(pages[pageIdx]) {
			remaining -= len(pages[pageIdx])
			pageIdx++
		}
		if pageIdx >= len(pages) {
			return nil, total, nil
		}
		return pages[pageIdx], total, nil
	}
}
