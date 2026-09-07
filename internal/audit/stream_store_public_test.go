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

package audit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/audit"
	"github.com/osapi-io/osapi/internal/audit/mocks"
)

type StreamStorePublicTestSuite struct {
	suite.Suite

	ctrl          *gomock.Controller
	mockStream    *mocks.MockStream
	mockPublisher *mocks.MockPublisher
	store         *audit.StreamStore
}

func (s *StreamStorePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockStream = mocks.NewMockStream(s.ctrl)
	s.mockPublisher = mocks.NewMockPublisher(s.ctrl)
	s.store = audit.NewStreamStore(
		slog.Default(),
		s.mockStream,
		s.mockPublisher,
		"audit.log",
	)
}

func (s *StreamStorePublicTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *StreamStorePublicTestSuite) TearDownSubTest() {
	audit.ResetMarshalJSON()
}

func (s *StreamStorePublicTestSuite) newEntry(
	id string,
) audit.Entry {
	return audit.Entry{
		ID:           id,
		Timestamp:    time.Now(),
		User:         "user@example.com",
		Roles:        []string{"admin"},
		Method:       "GET",
		Path:         "/node/hostname",
		SourceIP:     "127.0.0.1",
		ResponseCode: 200,
		DurationMs:   42,
		TraceID:      "abc123",
	}
}

func (s *StreamStorePublicTestSuite) newMsgChan(
	msgs ...jetstream.Msg,
) <-chan jetstream.Msg {
	ch := make(chan jetstream.Msg, len(msgs))
	for _, msg := range msgs {
		ch <- msg
	}
	close(ch)

	return ch
}

func (s *StreamStorePublicTestSuite) newStreamInfo(
	msgs uint64,
	firstSeq uint64,
) *jetstream.StreamInfo {
	return &jetstream.StreamInfo{
		State: jetstream.StreamState{
			Msgs:     msgs,
			FirstSeq: firstSeq,
		},
	}
}

func (s *StreamStorePublicTestSuite) TestWrite() {
	tests := []struct {
		name         string
		entry        audit.Entry
		setupMock    func()
		validateFunc func(error)
	}{
		{
			name:  "successfully publishes entry",
			entry: s.newEntry("entry-1"),
			setupMock: func() {
				s.mockPublisher.EXPECT().
					Publish(gomock.Any(), "audit.log.entry-1", gomock.Any()).
					Return(nil)
			},
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name:  "returns error when publish fails",
			entry: s.newEntry("entry-2"),
			setupMock: func() {
				s.mockPublisher.EXPECT().
					Publish(gomock.Any(), "audit.log.entry-2", gomock.Any()).
					Return(fmt.Errorf("publish error"))
			},
			validateFunc: func(err error) {
				s.Error(err)
				s.Contains(err.Error(), "publish audit entry")
			},
		},
		{
			name:  "returns error when marshal fails",
			entry: s.newEntry("entry-3"),
			setupMock: func() {
				audit.SetMarshalJSON(func(_ interface{}) ([]byte, error) {
					return nil, fmt.Errorf("marshal failure")
				})
			},
			validateFunc: func(err error) {
				s.Error(err)
				s.Contains(err.Error(), "marshal audit entry")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			err := s.store.Write(context.Background(), tt.entry)
			tt.validateFunc(err)
		})
	}
}

func (s *StreamStorePublicTestSuite) TestGet() {
	entry := s.newEntry("entry-1")
	data, _ := json.Marshal(entry)

	tests := []struct {
		name         string
		id           string
		setupMock    func()
		validateFunc func(*audit.Entry, error)
	}{
		{
			name: "successfully gets entry",
			id:   "entry-1",
			setupMock: func() {
				s.mockStream.EXPECT().
					GetLastMsgForSubject(gomock.Any(), "audit.log.entry-1").
					Return(&jetstream.RawStreamMsg{Data: data}, nil)
			},
			validateFunc: func(e *audit.Entry, err error) {
				s.NoError(err)
				s.Require().NotNil(e)
				s.Equal("entry-1", e.ID)
				s.Equal("user@example.com", e.User)
				s.Equal("abc123", e.TraceID)
			},
		},
		{
			name: "returns error with not found when subject not found",
			id:   "missing",
			setupMock: func() {
				s.mockStream.EXPECT().
					GetLastMsgForSubject(gomock.Any(), "audit.log.missing").
					Return(nil, jetstream.ErrMsgNotFound)
			},
			validateFunc: func(e *audit.Entry, err error) {
				s.Error(err)
				s.Nil(e)
				s.Contains(err.Error(), "not found")
			},
		},
		{
			name: "returns error when unmarshal fails",
			id:   "bad-json",
			setupMock: func() {
				s.mockStream.EXPECT().
					GetLastMsgForSubject(gomock.Any(), "audit.log.bad-json").
					Return(&jetstream.RawStreamMsg{Data: []byte("not-json")}, nil)
			},
			validateFunc: func(e *audit.Entry, err error) {
				s.Error(err)
				s.Nil(e)
				s.Contains(err.Error(), "unmarshal audit entry")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			result, err := s.store.Get(context.Background(), tt.id)
			tt.validateFunc(result, err)
		})
	}
}

func (s *StreamStorePublicTestSuite) TestList() {
	entry1 := s.newEntry("aaa")
	entry2 := s.newEntry("bbb")
	entry3 := s.newEntry("ccc")
	data1, _ := json.Marshal(entry1)
	data2, _ := json.Marshal(entry2)
	data3, _ := json.Marshal(entry3)

	tests := []struct {
		name         string
		limit        int
		offset       int
		setupMock    func()
		validateFunc func([]audit.Entry, int, error)
	}{
		{
			name:   "returns entries newest-first within limit",
			limit:  10,
			offset: 0,
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(3, 1), nil)

				mockConsumer := mocks.NewMockConsumer(s.ctrl)
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), jetstream.OrderedConsumerConfig{
						DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
						OptStartSeq:   1,
					}).
					Return(mockConsumer, nil)

				msg1 := mocks.NewMockMsg(s.ctrl)
				msg1.EXPECT().Data().Return(data1)
				msg2 := mocks.NewMockMsg(s.ctrl)
				msg2.EXPECT().Data().Return(data2)
				msg3 := mocks.NewMockMsg(s.ctrl)
				msg3.EXPECT().Data().Return(data3)

				mockBatch := mocks.NewMockMessageBatch(s.ctrl)
				mockBatch.EXPECT().Messages().Return(s.newMsgChan(msg1, msg2, msg3))
				mockBatch.EXPECT().Error().Return(nil)

				mockConsumer.EXPECT().
					Fetch(3, gomock.Any()).
					Return(mockBatch, nil)
			},
			validateFunc: func(entries []audit.Entry, total int, err error) {
				s.NoError(err)
				s.Equal(3, total)
				s.Len(entries, 3)
				// Reversed: newest first
				s.Equal("ccc", entries[0].ID)
				s.Equal("bbb", entries[1].ID)
				s.Equal("aaa", entries[2].ID)
			},
		},
		{
			name:   "applies pagination offset and limit",
			limit:  1,
			offset: 1,
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(3, 1), nil)

				mockConsumer := mocks.NewMockConsumer(s.ctrl)
				// newestSeq=3, offset=1, count=1 => startSeq=3-1-1+1=2
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), jetstream.OrderedConsumerConfig{
						DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
						OptStartSeq:   2,
					}).
					Return(mockConsumer, nil)

				msg := mocks.NewMockMsg(s.ctrl)
				msg.EXPECT().Data().Return(data2)

				mockBatch := mocks.NewMockMessageBatch(s.ctrl)
				mockBatch.EXPECT().Messages().Return(s.newMsgChan(msg))
				mockBatch.EXPECT().Error().Return(nil)

				mockConsumer.EXPECT().
					Fetch(1, gomock.Any()).
					Return(mockBatch, nil)
			},
			validateFunc: func(entries []audit.Entry, total int, err error) {
				s.NoError(err)
				s.Equal(3, total)
				s.Len(entries, 1)
				s.Equal("bbb", entries[0].ID)
			},
		},
		{
			name:   "returns empty when offset exceeds total",
			limit:  10,
			offset: 100,
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(3, 1), nil)
			},
			validateFunc: func(entries []audit.Entry, total int, err error) {
				s.NoError(err)
				s.Equal(3, total)
				s.Empty(entries)
			},
		},
		{
			name:   "returns empty for empty stream",
			limit:  10,
			offset: 0,
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(0, 0), nil)
			},
			validateFunc: func(entries []audit.Entry, total int, err error) {
				s.NoError(err)
				s.Equal(0, total)
				s.Empty(entries)
			},
		},
		{
			name:   "returns error when Info fails",
			limit:  10,
			offset: 0,
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(nil, fmt.Errorf("connection error"))
			},
			validateFunc: func(entries []audit.Entry, total int, err error) {
				s.Error(err)
				s.Nil(entries)
				s.Equal(0, total)
				s.Contains(err.Error(), "get stream info")
			},
		},
		{
			name:   "returns error when OrderedConsumer fails",
			limit:  10,
			offset: 0,
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(3, 1), nil)
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("consumer error"))
			},
			validateFunc: func(entries []audit.Entry, total int, err error) {
				s.Error(err)
				s.Nil(entries)
				s.Equal(0, total)
				s.Contains(err.Error(), "create ordered consumer")
			},
		},
		{
			name:   "returns error when Fetch fails",
			limit:  10,
			offset: 0,
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(3, 1), nil)

				mockConsumer := mocks.NewMockConsumer(s.ctrl)
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), gomock.Any()).
					Return(mockConsumer, nil)
				mockConsumer.EXPECT().
					Fetch(3, gomock.Any()).
					Return(nil, fmt.Errorf("fetch error"))
			},
			validateFunc: func(entries []audit.Entry, total int, err error) {
				s.Error(err)
				s.Nil(entries)
				s.Equal(0, total)
				s.Contains(err.Error(), "fetch audit entries")
			},
		},
		{
			name:   "skips entries when unmarshal fails",
			limit:  10,
			offset: 0,
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(2, 1), nil)

				mockConsumer := mocks.NewMockConsumer(s.ctrl)
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), gomock.Any()).
					Return(mockConsumer, nil)

				badMsg := mocks.NewMockMsg(s.ctrl)
				badMsg.EXPECT().Data().Return([]byte("not-json"))
				goodMsg := mocks.NewMockMsg(s.ctrl)
				goodMsg.EXPECT().Data().Return(data1)

				mockBatch := mocks.NewMockMessageBatch(s.ctrl)
				mockBatch.EXPECT().Messages().Return(s.newMsgChan(badMsg, goodMsg))
				mockBatch.EXPECT().Error().Return(nil)

				mockConsumer.EXPECT().
					Fetch(2, gomock.Any()).
					Return(mockBatch, nil)
			},
			validateFunc: func(entries []audit.Entry, total int, err error) {
				s.NoError(err)
				s.Equal(2, total)
				s.Len(entries, 1)
				s.Equal("aaa", entries[0].ID)
			},
		},
		{
			name:   "logs warning on batch error",
			limit:  10,
			offset: 0,
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(1, 1), nil)

				mockConsumer := mocks.NewMockConsumer(s.ctrl)
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), gomock.Any()).
					Return(mockConsumer, nil)

				msg := mocks.NewMockMsg(s.ctrl)
				msg.EXPECT().Data().Return(data1)

				mockBatch := mocks.NewMockMessageBatch(s.ctrl)
				mockBatch.EXPECT().Messages().Return(s.newMsgChan(msg))
				mockBatch.EXPECT().Error().Return(fmt.Errorf("batch timeout"))

				mockConsumer.EXPECT().
					Fetch(1, gomock.Any()).
					Return(mockBatch, nil)
			},
			validateFunc: func(entries []audit.Entry, total int, err error) {
				s.NoError(err)
				s.Equal(1, total)
				s.Len(entries, 1)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			entries, total, err := s.store.List(context.Background(), tt.limit, tt.offset)
			tt.validateFunc(entries, total, err)
		})
	}
}

func (s *StreamStorePublicTestSuite) TestListAll() {
	entry1 := s.newEntry("aaa")
	entry2 := s.newEntry("bbb")
	entry3 := s.newEntry("ccc")
	data1, _ := json.Marshal(entry1)
	data2, _ := json.Marshal(entry2)
	data3, _ := json.Marshal(entry3)

	tests := []struct {
		name         string
		setupMock    func()
		validateFunc func([]audit.Entry, error)
	}{
		{
			name: "returns all entries newest-first",
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(3, 1), nil)

				mockConsumer := mocks.NewMockConsumer(s.ctrl)
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), jetstream.OrderedConsumerConfig{
						DeliverPolicy: jetstream.DeliverAllPolicy,
					}).
					Return(mockConsumer, nil)

				msg1 := mocks.NewMockMsg(s.ctrl)
				msg1.EXPECT().Data().Return(data1)
				msg2 := mocks.NewMockMsg(s.ctrl)
				msg2.EXPECT().Data().Return(data2)
				msg3 := mocks.NewMockMsg(s.ctrl)
				msg3.EXPECT().Data().Return(data3)

				mockBatch := mocks.NewMockMessageBatch(s.ctrl)
				mockBatch.EXPECT().Messages().Return(s.newMsgChan(msg1, msg2, msg3))
				mockBatch.EXPECT().Error().Return(nil)

				mockConsumer.EXPECT().
					Fetch(3, gomock.Any()).
					Return(mockBatch, nil)
			},
			validateFunc: func(entries []audit.Entry, err error) {
				s.NoError(err)
				s.Len(entries, 3)
				// Reversed: newest first
				s.Equal("ccc", entries[0].ID)
				s.Equal("bbb", entries[1].ID)
				s.Equal("aaa", entries[2].ID)
			},
		},
		{
			name: "returns empty for empty stream",
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(0, 0), nil)
			},
			validateFunc: func(entries []audit.Entry, err error) {
				s.NoError(err)
				s.Empty(entries)
			},
		},
		{
			name: "returns error when Info fails",
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(nil, fmt.Errorf("connection error"))
			},
			validateFunc: func(entries []audit.Entry, err error) {
				s.Error(err)
				s.Nil(entries)
				s.Contains(err.Error(), "get stream info")
			},
		},
		{
			name: "returns error when OrderedConsumer fails",
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(3, 1), nil)
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("consumer error"))
			},
			validateFunc: func(entries []audit.Entry, err error) {
				s.Error(err)
				s.Nil(entries)
				s.Contains(err.Error(), "create ordered consumer")
			},
		},
		{
			name: "returns error when Fetch fails",
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(3, 1), nil)

				mockConsumer := mocks.NewMockConsumer(s.ctrl)
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), gomock.Any()).
					Return(mockConsumer, nil)
				mockConsumer.EXPECT().
					Fetch(3, gomock.Any()).
					Return(nil, fmt.Errorf("fetch error"))
			},
			validateFunc: func(entries []audit.Entry, err error) {
				s.Error(err)
				s.Nil(entries)
				s.Contains(err.Error(), "fetch audit entries")
			},
		},
		{
			name: "skips entries when unmarshal fails",
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(2, 1), nil)

				mockConsumer := mocks.NewMockConsumer(s.ctrl)
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), gomock.Any()).
					Return(mockConsumer, nil)

				badMsg := mocks.NewMockMsg(s.ctrl)
				badMsg.EXPECT().Data().Return([]byte("not-json"))
				goodMsg := mocks.NewMockMsg(s.ctrl)
				goodMsg.EXPECT().Data().Return(data1)

				mockBatch := mocks.NewMockMessageBatch(s.ctrl)
				mockBatch.EXPECT().Messages().Return(s.newMsgChan(badMsg, goodMsg))
				mockBatch.EXPECT().Error().Return(nil)

				mockConsumer.EXPECT().
					Fetch(2, gomock.Any()).
					Return(mockBatch, nil)
			},
			validateFunc: func(entries []audit.Entry, err error) {
				s.NoError(err)
				s.Len(entries, 1)
				s.Equal("aaa", entries[0].ID)
			},
		},
		{
			name: "logs warning on batch error",
			setupMock: func() {
				s.mockStream.EXPECT().
					Info(gomock.Any()).
					Return(s.newStreamInfo(1, 1), nil)

				mockConsumer := mocks.NewMockConsumer(s.ctrl)
				s.mockStream.EXPECT().
					OrderedConsumer(gomock.Any(), gomock.Any()).
					Return(mockConsumer, nil)

				msg := mocks.NewMockMsg(s.ctrl)
				msg.EXPECT().Data().Return(data1)

				mockBatch := mocks.NewMockMessageBatch(s.ctrl)
				mockBatch.EXPECT().Messages().Return(s.newMsgChan(msg))
				mockBatch.EXPECT().Error().Return(fmt.Errorf("batch timeout"))

				mockConsumer.EXPECT().
					Fetch(1, gomock.Any()).
					Return(mockBatch, nil)
			},
			validateFunc: func(entries []audit.Entry, err error) {
				s.NoError(err)
				s.Len(entries, 1)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			entries, err := s.store.ListAll(context.Background())
			tt.validateFunc(entries, err)
		})
	}
}

func TestStreamStorePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(StreamStorePublicTestSuite))
}
