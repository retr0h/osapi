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

package enrollment_test

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent/pki"
	"github.com/osapi-io/osapi/internal/controller/enrollment"
	enrollMocks "github.com/osapi-io/osapi/internal/controller/enrollment/mocks"
	jobMocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type WatcherPublicTestSuite struct {
	suite.Suite

	ctx       context.Context
	mockCtrl  *gomock.Controller
	mockNC    *enrollMocks.MockNATSSubscriber
	mockKV    *jobMocks.MockKeyValue
	mockPKI   *enrollMocks.MockPKIProvider
	watcher   *enrollment.Watcher
	fixedTime time.Time
	pubKey    ed25519.PublicKey
}

func (s *WatcherPublicTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockCtrl = gomock.NewController(s.T())
	s.mockNC = enrollMocks.NewMockNATSSubscriber(s.mockCtrl)
	s.mockKV = jobMocks.NewMockKeyValue(s.mockCtrl)
	s.mockPKI = enrollMocks.NewMockPKIProvider(s.mockCtrl)
	s.fixedTime = time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	s.pubKey = make(ed25519.PublicKey, ed25519.PublicKeySize)

	enrollment.SetNowFn(func() time.Time { return s.fixedTime })

	s.watcher = enrollment.NewWatcher(
		slog.Default(),
		s.mockNC,
		s.mockKV,
		s.mockPKI,
		false,
		"osapi",
	)
}

func (s *WatcherPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *WatcherPublicTestSuite) TearDownSubTest() {
	enrollment.ResetMarshalFn()
	enrollment.ResetUnmarshalFn()
	enrollment.ResetNowFn()
}

func (s *WatcherPublicTestSuite) TestHandleEnrollmentRequest() {
	tests := []struct {
		name       string
		setupMock  func()
		msg        *nats.Msg
		validateFn func()
	}{
		{
			name: "stores pending agent in KV",
			setupMock: func() {
				s.mockKV.EXPECT().
					Put(gomock.Any(), "enrollment.machine-001", gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						value []byte,
					) (uint64, error) {
						var stored enrollment.PendingAgent
						err := json.Unmarshal(value, &stored)
						s.Require().NoError(err, "stored value must be valid JSON")
						s.Equal("machine-001", stored.MachineID)
						s.Equal("web-01", stored.Hostname)
						s.Equal("SHA256:abc123", stored.Fingerprint)
						s.False(
							stored.RequestedAt.IsZero(),
							"requested_at must be non-zero",
						)

						return uint64(1), nil
					})
			},
			msg: s.makeEnrollmentMsg("machine-001", "web-01", "SHA256:abc123"),
		},
		{
			name:      "logs warning on unmarshal failure",
			setupMock: func() {},
			msg:       &nats.Msg{Data: []byte("invalid json")},
		},
		{
			name: "logs warning on marshal failure",
			setupMock: func() {
				enrollment.SetMarshalFn(func(_ any) ([]byte, error) {
					return nil, errors.New("marshal error")
				})
			},
			msg: s.makeEnrollmentMsg("machine-001", "web-01", "SHA256:abc123"),
		},
		{
			name: "logs warning on KV put failure",
			setupMock: func() {
				s.mockKV.EXPECT().
					Put(gomock.Any(), "enrollment.machine-001", gomock.Any()).
					Return(uint64(0), errors.New("kv error"))
			},
			msg: s.makeEnrollmentMsg("machine-001", "web-01", "SHA256:abc123"),
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupMock()
			enrollment.ExportHandleEnrollmentRequest(s.ctx, s.watcher, tc.msg)
		})
	}
}

func (s *WatcherPublicTestSuite) TestHandleEnrollmentRequestAutoAccept() {
	autoWatcher := enrollment.NewWatcher(
		slog.Default(),
		s.mockNC,
		s.mockKV,
		s.mockPKI,
		true,
		"osapi",
	)

	tests := []struct {
		name      string
		setupMock func()
		msg       *nats.Msg
	}{
		{
			name: "auto-accepts on successful store",
			setupMock: func() {
				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				// Store in KV.
				s.mockKV.EXPECT().
					Put(gomock.Any(), "enrollment.machine-001", gomock.Any()).
					Return(uint64(1), nil)

				// AcceptAgent: Get from KV.
				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)

				// AcceptAgent: Publish response — verify the acceptance payload.
				s.mockPKI.EXPECT().PublicKey().Return(s.pubKey)
				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-001", gomock.Any()).
					DoAndReturn(func(
						_ string,
						data []byte,
					) error {
						var resp struct {
							Accepted            bool   `json:"accepted"`
							ControllerPublicKey []byte `json:"controller_public_key"`
						}
						err := json.Unmarshal(data, &resp)
						s.Require().NoError(err, "response must be valid JSON")
						s.True(resp.Accepted, "response must indicate acceptance")
						s.NotEmpty(
							resp.ControllerPublicKey,
							"response must include controller public key",
						)

						return nil
					})

				// AcceptAgent: Delete from KV.
				s.mockKV.EXPECT().
					Delete(gomock.Any(), "enrollment.machine-001").
					Return(nil)
			},
			msg: s.makeEnrollmentMsg("machine-001", "web-01", "SHA256:abc123"),
		},
		{
			name: "logs warning when auto-accept fails",
			setupMock: func() {
				// Store in KV.
				s.mockKV.EXPECT().
					Put(gomock.Any(), "enrollment.machine-002", gomock.Any()).
					Return(uint64(1), nil)

				// AcceptAgent: Get from KV fails.
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-002").
					Return(nil, errors.New("kv error"))
			},
			msg: s.makeEnrollmentMsg("machine-002", "web-02", "SHA256:def456"),
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupMock()
			enrollment.ExportHandleEnrollmentRequest(s.ctx, autoWatcher, tc.msg)
		})
	}
}

func (s *WatcherPublicTestSuite) TestAcceptAgent() {
	tests := []struct {
		name       string
		machineID  string
		setupMock  func()
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "accepts pending agent and publishes response",
			machineID: "machine-001",
			setupMock: func() {
				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)

				s.mockPKI.EXPECT().PublicKey().Return(s.pubKey)
				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-001", gomock.Any()).
					Return(nil)

				s.mockKV.EXPECT().
					Delete(gomock.Any(), "enrollment.machine-001").
					Return(nil)
			},
		},
		{
			name:      "returns error when pending agent not found",
			machineID: "missing",
			setupMock: func() {
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.missing").
					Return(nil, jetstream.ErrKeyNotFound)
			},
			wantErr:    true,
			wantErrMsg: "get pending agent missing",
		},
		{
			name:      "returns error on unmarshal failure",
			machineID: "machine-001",
			setupMock: func() {
				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte("bad json"))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)
			},
			wantErr:    true,
			wantErrMsg: "unmarshal pending agent machine-001",
		},
		{
			name:      "returns error on marshal failure",
			machineID: "machine-001",
			setupMock: func() {
				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)

				s.mockPKI.EXPECT().PublicKey().Return(s.pubKey)

				enrollment.SetMarshalFn(func(_ any) ([]byte, error) {
					return nil, errors.New("marshal error")
				})
			},
			wantErr:    true,
			wantErrMsg: "marshal acceptance response",
		},
		{
			name:      "returns error on publish failure",
			machineID: "machine-001",
			setupMock: func() {
				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)

				s.mockPKI.EXPECT().PublicKey().Return(s.pubKey)
				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-001", gomock.Any()).
					Return(errors.New("publish error"))
			},
			wantErr:    true,
			wantErrMsg: "publish acceptance for machine-001",
		},
		{
			name:      "returns error on delete failure",
			machineID: "machine-001",
			setupMock: func() {
				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)

				s.mockPKI.EXPECT().PublicKey().Return(s.pubKey)
				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-001", gomock.Any()).
					Return(nil)

				s.mockKV.EXPECT().
					Delete(gomock.Any(), "enrollment.machine-001").
					Return(errors.New("delete error"))
			},
			wantErr:    true,
			wantErrMsg: "delete pending agent machine-001",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupMock()
			err := s.watcher.AcceptAgent(s.ctx, tc.machineID)
			if tc.wantErr {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErrMsg)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *WatcherPublicTestSuite) TestRejectAgent() {
	tests := []struct {
		name       string
		machineID  string
		reason     string
		setupMock  func()
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "rejects pending agent and publishes response",
			machineID: "machine-001",
			reason:    "not authorized",
			setupMock: func() {
				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)

				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-001", gomock.Any()).
					DoAndReturn(func(_ string, data []byte) error {
						var resp pki.EnrollmentResponse
						s.Require().NoError(json.Unmarshal(data, &resp))
						s.False(resp.Accepted)
						s.Equal("not authorized", resp.Reason)
						return nil
					})

				s.mockKV.EXPECT().
					Delete(gomock.Any(), "enrollment.machine-001").
					Return(nil)
			},
		},
		{
			name:      "returns error when pending agent not found",
			machineID: "missing",
			reason:    "denied",
			setupMock: func() {
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.missing").
					Return(nil, jetstream.ErrKeyNotFound)
			},
			wantErr:    true,
			wantErrMsg: "get pending agent missing",
		},
		{
			name:      "returns error on unmarshal failure",
			machineID: "machine-001",
			reason:    "denied",
			setupMock: func() {
				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte("bad"))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)
			},
			wantErr:    true,
			wantErrMsg: "unmarshal pending agent machine-001",
		},
		{
			name:      "returns error on marshal failure",
			machineID: "machine-001",
			reason:    "denied",
			setupMock: func() {
				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)

				enrollment.SetMarshalFn(func(_ any) ([]byte, error) {
					return nil, errors.New("marshal error")
				})
			},
			wantErr:    true,
			wantErrMsg: "marshal rejection response",
		},
		{
			name:      "returns error on publish failure",
			machineID: "machine-001",
			reason:    "denied",
			setupMock: func() {
				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)

				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-001", gomock.Any()).
					Return(errors.New("publish error"))
			},
			wantErr:    true,
			wantErrMsg: "publish rejection for machine-001",
		},
		{
			name:      "returns error on delete failure",
			machineID: "machine-001",
			reason:    "denied",
			setupMock: func() {
				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				mockEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(mockEntry, nil)

				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-001", gomock.Any()).
					Return(nil)

				s.mockKV.EXPECT().
					Delete(gomock.Any(), "enrollment.machine-001").
					Return(errors.New("delete error"))
			},
			wantErr:    true,
			wantErrMsg: "delete pending agent machine-001",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupMock()
			err := s.watcher.RejectAgent(s.ctx, tc.machineID, tc.reason)
			if tc.wantErr {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErrMsg)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *WatcherPublicTestSuite) TestListPending() {
	tests := []struct {
		name       string
		setupMock  func()
		wantLen    int
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "returns all pending agents",
			setupMock: func() {
				keys := make(chan string, 2)
				keys <- "enrollment.machine-001"
				keys <- "enrollment.machine-002"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				entry1 := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry1.EXPECT().Value().Return(
					s.makePendingJSON("machine-001", "web-01", "SHA256:abc123"),
				)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(entry1, nil)

				entry2 := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry2.EXPECT().Value().Return(
					s.makePendingJSON("machine-002", "web-02", "SHA256:def456"),
				)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-002").
					Return(entry2, nil)
			},
			wantLen: 2,
		},
		{
			name: "returns nil when bucket is empty",
			setupMock: func() {
				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(nil, jetstream.ErrNoKeysFound)
			},
			wantLen: 0,
		},
		{
			name: "returns error on list failure",
			setupMock: func() {
				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(nil, errors.New("list error"))
			},
			wantErr:    true,
			wantErrMsg: "list enrollment keys",
		},
		{
			name: "skips entries with get errors",
			setupMock: func() {
				keys := make(chan string, 2)
				keys <- "enrollment.machine-001"
				keys <- "enrollment.machine-002"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(nil, errors.New("get error"))

				entry2 := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry2.EXPECT().Value().Return(
					s.makePendingJSON("machine-002", "web-02", "SHA256:def456"),
				)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-002").
					Return(entry2, nil)
			},
			wantLen: 1,
		},
		{
			name: "skips entries with unmarshal errors",
			setupMock: func() {
				keys := make(chan string, 1)
				keys <- "enrollment.machine-001"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return([]byte("bad json"))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(entry, nil)
			},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupMock()
			pending, err := s.watcher.ListPending(s.ctx)
			if tc.wantErr {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErrMsg)
			} else {
				s.Require().NoError(err)
				s.Len(pending, tc.wantLen)
			}
		})
	}
}

func (s *WatcherPublicTestSuite) TestAcceptByHostname() {
	tests := []struct {
		name       string
		hostname   string
		setupMock  func()
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:     "finds and accepts agent by hostname",
			hostname: "web-01",
			setupMock: func() {
				// findPendingBy: ListKeys + Get.
				keys := make(chan string, 1)
				keys <- "enrollment.machine-001"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				findEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				findEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(findEntry, nil)

				// AcceptAgent: Get from KV.
				acceptEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				acceptEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(acceptEntry, nil)

				// AcceptAgent: Publish + Delete.
				s.mockPKI.EXPECT().PublicKey().Return(s.pubKey)
				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-001", gomock.Any()).
					Return(nil)
				s.mockKV.EXPECT().
					Delete(gomock.Any(), "enrollment.machine-001").
					Return(nil)
			},
		},
		{
			name:     "returns error when no matching hostname found",
			hostname: "nonexistent",
			setupMock: func() {
				keys := make(chan string, 1)
				keys <- "enrollment.machine-001"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")
				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(entry, nil)
			},
			wantErr:    true,
			wantErrMsg: `no pending agent with hostname "nonexistent"`,
		},
		{
			name:     "returns error when bucket is empty",
			hostname: "web-01",
			setupMock: func() {
				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(nil, jetstream.ErrNoKeysFound)
			},
			wantErr:    true,
			wantErrMsg: `no pending agent with hostname "web-01"`,
		},
		{
			name:     "returns error on list failure",
			hostname: "web-01",
			setupMock: func() {
				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(nil, errors.New("list error"))
			},
			wantErr:    true,
			wantErrMsg: "list enrollment keys",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupMock()
			err := s.watcher.AcceptByHostname(s.ctx, tc.hostname)
			if tc.wantErr {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErrMsg)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *WatcherPublicTestSuite) TestAcceptByFingerprint() {
	tests := []struct {
		name        string
		fingerprint string
		setupMock   func()
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:        "finds and accepts agent by fingerprint",
			fingerprint: "SHA256:abc123",
			setupMock: func() {
				// findPendingBy: ListKeys + Get.
				keys := make(chan string, 1)
				keys <- "enrollment.machine-001"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				findEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				findEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(findEntry, nil)

				// AcceptAgent: Get from KV.
				acceptEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				acceptEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(acceptEntry, nil)

				// AcceptAgent: Publish + Delete.
				s.mockPKI.EXPECT().PublicKey().Return(s.pubKey)
				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-001", gomock.Any()).
					Return(nil)
				s.mockKV.EXPECT().
					Delete(gomock.Any(), "enrollment.machine-001").
					Return(nil)
			},
		},
		{
			name:        "returns error when no matching fingerprint found",
			fingerprint: "SHA256:unknown",
			setupMock: func() {
				keys := make(chan string, 1)
				keys <- "enrollment.machine-001"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")
				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(entry, nil)
			},
			wantErr:    true,
			wantErrMsg: `no pending agent with fingerprint "SHA256:unknown"`,
		},
		{
			name:        "returns error when bucket is empty",
			fingerprint: "SHA256:abc123",
			setupMock: func() {
				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(nil, jetstream.ErrNoKeysFound)
			},
			wantErr:    true,
			wantErrMsg: `no pending agent with fingerprint "SHA256:abc123"`,
		},
		{
			name:        "returns error on list failure",
			fingerprint: "SHA256:abc123",
			setupMock: func() {
				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(nil, errors.New("list error"))
			},
			wantErr:    true,
			wantErrMsg: "list enrollment keys",
		},
		{
			name:        "skips entries with get errors during scan",
			fingerprint: "SHA256:def456",
			setupMock: func() {
				keys := make(chan string, 2)
				keys <- "enrollment.machine-001"
				keys <- "enrollment.machine-002"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				// First entry: get error, skipped.
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(nil, errors.New("get error"))

				// Second entry: matches.
				pendingData := s.makePendingJSON("machine-002", "web-02", "SHA256:def456")
				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-002").
					Return(entry, nil)

				// AcceptAgent: Get from KV.
				acceptEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				acceptEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-002").
					Return(acceptEntry, nil)

				// AcceptAgent: Publish + Delete.
				s.mockPKI.EXPECT().PublicKey().Return(s.pubKey)
				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-002", gomock.Any()).
					Return(nil)
				s.mockKV.EXPECT().
					Delete(gomock.Any(), "enrollment.machine-002").
					Return(nil)
			},
		},
		{
			name:        "skips entries with unmarshal errors during scan",
			fingerprint: "SHA256:def456",
			setupMock: func() {
				keys := make(chan string, 2)
				keys <- "enrollment.machine-001"
				keys <- "enrollment.machine-002"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				// First entry: bad JSON, skipped.
				badEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				badEntry.EXPECT().Value().Return([]byte("bad"))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(badEntry, nil)

				// Second entry: matches.
				pendingData := s.makePendingJSON("machine-002", "web-02", "SHA256:def456")
				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-002").
					Return(entry, nil)

				// AcceptAgent: Get from KV.
				acceptEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				acceptEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-002").
					Return(acceptEntry, nil)

				// AcceptAgent: Publish + Delete.
				s.mockPKI.EXPECT().PublicKey().Return(s.pubKey)
				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-002", gomock.Any()).
					Return(nil)
				s.mockKV.EXPECT().
					Delete(gomock.Any(), "enrollment.machine-002").
					Return(nil)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupMock()
			err := s.watcher.AcceptByFingerprint(s.ctx, tc.fingerprint)
			if tc.wantErr {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErrMsg)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *WatcherPublicTestSuite) TestStart() {
	tests := []struct {
		name       string
		setupMock  func()
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "subscribes and blocks until context cancelled",
			setupMock: func() {
				s.mockNC.EXPECT().
					Subscribe("osapi.enroll.request", gomock.Any()).
					DoAndReturn(func(_ string, cb nats.MsgHandler) (*nats.Subscription, error) {
						// Invoke the callback to cover the closure body inside Start.
						// The msg contains invalid JSON, so handleEnrollmentRequest
						// logs a warning and returns without side effects.
						cb(&nats.Msg{Data: []byte("invalid")})
						return &nats.Subscription{}, nil
					})
			},
		},
		{
			name: "returns error on subscribe failure",
			setupMock: func() {
				s.mockNC.EXPECT().
					Subscribe("osapi.enroll.request", gomock.Any()).
					Return(nil, errors.New("subscribe error"))
			},
			wantErr:    true,
			wantErrMsg: "subscribe to enrollment requests",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupMock()

			if tc.wantErr {
				err := s.watcher.Start(s.ctx)
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErrMsg)
			} else {
				ctx, cancel := context.WithCancel(s.ctx)
				errCh := make(chan error, 1)

				go func() {
					errCh <- s.watcher.Start(ctx)
				}()

				cancel()

				err := <-errCh
				s.Require().NoError(err)
			}
		})
	}
}

func (s *WatcherPublicTestSuite) TestEnrollSubject() {
	tests := []struct {
		name         string
		namespace    string
		suffix       string
		validateFunc func(string)
	}{
		{
			name:      "with namespace",
			namespace: "osapi",
			suffix:    "enroll.request",
			validateFunc: func(got string) {
				s.Equal("osapi.enroll.request", got)
			},
		},
		{
			name:      "without namespace",
			namespace: "",
			suffix:    "enroll.request",
			validateFunc: func(got string) {
				s.Equal("enroll.request", got)
			},
		},
		{
			name:      "response subject with namespace",
			namespace: "osapi",
			suffix:    "enroll.response.machine-001",
			validateFunc: func(got string) {
				s.Equal("osapi.enroll.response.machine-001", got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := enrollment.EnrollSubject(tc.namespace, tc.suffix)
			tc.validateFunc(got)
		})
	}
}

func (s *WatcherPublicTestSuite) TestKVPrefix() {
	s.Equal("enrollment.", enrollment.KVPrefix())
}

// makeEnrollmentMsg creates a *nats.Msg with a serialized EnrollmentRequest.
func (s *WatcherPublicTestSuite) makeEnrollmentMsg(
	machineID string,
	hostname string,
	fingerprint string,
) *nats.Msg {
	s.T().Helper()

	req := pki.EnrollmentRequest{
		MachineID:   machineID,
		Hostname:    hostname,
		PublicKey:   []byte("test-pubkey"),
		Fingerprint: fingerprint,
	}

	data, err := json.Marshal(req)
	s.Require().NoError(err)

	return &nats.Msg{Data: data}
}

func (s *WatcherPublicTestSuite) TestRejectByHostname() {
	tests := []struct {
		name       string
		hostname   string
		setupMock  func()
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:     "finds and rejects agent by hostname",
			hostname: "web-01",
			setupMock: func() {
				// findPendingBy: ListKeys + Get.
				keys := make(chan string, 1)
				keys <- "enrollment.machine-001"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")

				findEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				findEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(findEntry, nil)

				// RejectAgent: Get from KV.
				rejectEntry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				rejectEntry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(rejectEntry, nil)

				// RejectAgent: Publish + Delete.
				s.mockNC.EXPECT().
					PublishCore("osapi.enroll.response.machine-001", gomock.Any()).
					Return(nil)
				s.mockKV.EXPECT().
					Delete(gomock.Any(), "enrollment.machine-001").
					Return(nil)
			},
		},
		{
			name:     "returns error when no matching hostname found",
			hostname: "nonexistent",
			setupMock: func() {
				keys := make(chan string, 1)
				keys <- "enrollment.machine-001"
				close(keys)

				mockLister := jobMocks.NewMockKeyLister(s.mockCtrl)
				mockLister.EXPECT().Keys().Return(keys)

				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(mockLister, nil)

				pendingData := s.makePendingJSON("machine-001", "web-01", "SHA256:abc123")
				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(pendingData)
				s.mockKV.EXPECT().
					Get(gomock.Any(), "enrollment.machine-001").
					Return(entry, nil)
			},
			wantErr:    true,
			wantErrMsg: `no pending agent with hostname "nonexistent"`,
		},
		{
			name:     "returns error when bucket is empty",
			hostname: "web-01",
			setupMock: func() {
				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(nil, jetstream.ErrNoKeysFound)
			},
			wantErr:    true,
			wantErrMsg: `no pending agent with hostname "web-01"`,
		},
		{
			name:     "returns error on list failure",
			hostname: "web-01",
			setupMock: func() {
				s.mockKV.EXPECT().
					ListKeys(gomock.Any()).
					Return(nil, errors.New("list error"))
			},
			wantErr:    true,
			wantErrMsg: "list enrollment keys",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupMock()
			err := s.watcher.RejectByHostname(s.ctx, tc.hostname, "rejected via API")
			if tc.wantErr {
				s.Require().Error(err)
				s.Contains(err.Error(), tc.wantErrMsg)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

// makePendingJSON creates serialized PendingAgent JSON.
func (s *WatcherPublicTestSuite) makePendingJSON(
	machineID string,
	hostname string,
	fingerprint string,
) []byte {
	s.T().Helper()

	pending := enrollment.PendingAgent{
		MachineID:   machineID,
		Hostname:    hostname,
		PublicKey:   []byte("test-pubkey"),
		Fingerprint: fingerprint,
		RequestedAt: s.fixedTime,
	}

	data, err := json.Marshal(pending)
	s.Require().NoError(err)

	return data
}

func TestWatcherPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WatcherPublicTestSuite))
}
