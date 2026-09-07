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

package notify_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/controller/notify"
	notifymocks "github.com/osapi-io/osapi/internal/controller/notify/mocks"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type WatcherPublicTestSuite struct {
	suite.Suite

	ctrl         *gomock.Controller
	mockNotifier *notifymocks.MockNotifier
	capturedEvts []notify.ConditionEvent
	watcher      *notify.Watcher
}

func (s *WatcherPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockNotifier = notifymocks.NewMockNotifier(s.ctrl)
	s.capturedEvts = nil
	s.watcher = notify.NewWatcher(nil, s.mockNotifier, slog.Default(), 0)
}

func (s *WatcherPublicTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *WatcherPublicTestSuite) TestDetectTransitions() {
	tests := []struct {
		name          string
		key           string
		componentType string
		hostname      string
		conditions    []job.Condition
		setupPrev     func()
		wantEvents    int
		validateFunc  func(events []notify.ConditionEvent)
	}{
		{
			name:          "detects new condition",
			key:           "agents.web_01",
			componentType: "agent",
			hostname:      "web-01",
			conditions: []job.Condition{
				{
					Type:               job.ConditionMemoryPressure,
					Status:             true,
					Reason:             "memory above threshold",
					LastTransitionTime: time.Now(),
				},
			},
			setupPrev:  func() {},
			wantEvents: 1,
			validateFunc: func(events []notify.ConditionEvent) {
				s.Require().Len(events, 1)
				s.Equal("agent", events[0].ComponentType)
				s.Equal("web-01", events[0].Hostname)
				s.Equal(job.ConditionMemoryPressure, events[0].Condition)
				s.True(events[0].Active)
			},
		},
		{
			name:          "detects resolved condition",
			key:           "agents.web_01",
			componentType: "agent",
			hostname:      "web-01",
			conditions:    []job.Condition{},
			setupPrev: func() {
				notify.WatcherSetPrevEntry(
					s.watcher,
					"agents.web_01",
					map[string]*notify.ExportConditionState{
						job.ConditionMemoryPressure: notify.NewConditionState(true, time.Now()),
					},
				)
			},
			wantEvents: 1,
			validateFunc: func(events []notify.ConditionEvent) {
				s.Require().Len(events, 1)
				s.Equal("agent", events[0].ComponentType)
				s.Equal("web-01", events[0].Hostname)
				s.Equal(job.ConditionMemoryPressure, events[0].Condition)
				s.False(events[0].Active)
			},
		},
		{
			name:          "no notification when no change",
			key:           "agents.web_01",
			componentType: "agent",
			hostname:      "web-01",
			conditions: []job.Condition{
				{
					Type:               job.ConditionMemoryPressure,
					Status:             true,
					Reason:             "memory above threshold",
					LastTransitionTime: time.Now(),
				},
			},
			setupPrev: func() {
				notify.WatcherSetPrevEntry(
					s.watcher,
					"agents.web_01",
					map[string]*notify.ExportConditionState{
						job.ConditionMemoryPressure: notify.NewConditionState(true, time.Now()),
					},
				)
			},
			wantEvents: 0,
			validateFunc: func(events []notify.ConditionEvent) {
				s.Empty(events)
			},
		},
		{
			name:          "re-notifies after renotify interval",
			key:           "agents.web_01",
			componentType: "agent",
			hostname:      "web-01",
			conditions: []job.Condition{
				{
					Type:               job.ConditionMemoryPressure,
					Status:             true,
					LastTransitionTime: time.Now(),
				},
			},
			setupPrev: func() {
				notify.WatcherSetRenotifyInterval(s.watcher, 1*time.Millisecond)
				notify.WatcherSetPrevEntry(
					s.watcher,
					"agents.web_01",
					map[string]*notify.ExportConditionState{
						job.ConditionMemoryPressure: notify.NewConditionState(
							true,
							time.Now().Add(-1*time.Second),
						),
					},
				)
			},
			wantEvents: 1,
			validateFunc: func(events []notify.ConditionEvent) {
				s.Require().Len(events, 1)
				s.True(events[0].Active)
				s.Equal(job.ConditionMemoryPressure, events[0].Condition)
			},
		},
		{
			name:          "does not re-notify before interval elapses",
			key:           "agents.web_01",
			componentType: "agent",
			hostname:      "web-01",
			conditions: []job.Condition{
				{
					Type:               job.ConditionMemoryPressure,
					Status:             true,
					LastTransitionTime: time.Now(),
				},
			},
			setupPrev: func() {
				notify.WatcherSetRenotifyInterval(s.watcher, 1*time.Hour)
				notify.WatcherSetPrevEntry(
					s.watcher,
					"agents.web_01",
					map[string]*notify.ExportConditionState{
						job.ConditionMemoryPressure: notify.NewConditionState(true, time.Now()),
					},
				)
			},
			wantEvents: 0,
			validateFunc: func(events []notify.ConditionEvent) {
				s.Empty(events)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Reset captured events and watcher state between sub-tests.
			s.capturedEvts = nil
			notify.WatcherSetPrev(
				s.watcher,
				make(map[string]map[string]*notify.ExportConditionState),
			)

			if tt.wantEvents > 0 {
				s.mockNotifier.EXPECT().
					Notify(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						event notify.ConditionEvent,
					) error {
						s.capturedEvts = append(s.capturedEvts, event)
						return nil
					}).
					Times(tt.wantEvents)
			}

			tt.setupPrev()

			notify.WatcherDetectTransitions(
				s.watcher,
				tt.key,
				tt.componentType,
				tt.hostname,
				tt.conditions,
			)

			tt.validateFunc(s.capturedEvts)
		})
	}
}

func (s *WatcherPublicTestSuite) TestParseRegistryKey() {
	tests := []struct {
		name              string
		key               string
		wantComponentType string
		wantHostname      string
		wantOK            bool
	}{
		{
			name:              "agents prefix returns agent type",
			key:               "agents.web-01",
			wantComponentType: "agent",
			wantHostname:      "web-01",
			wantOK:            true,
		},
		{
			name:              "api prefix returns api type",
			key:               "api.api-server-01",
			wantComponentType: "api",
			wantHostname:      "api-server-01",
			wantOK:            true,
		},
		{
			name:              "nats prefix returns nats type",
			key:               "nats.nats-01",
			wantComponentType: "nats",
			wantHostname:      "nats-01",
			wantOK:            true,
		},
		{
			name:              "controller prefix returns controller type",
			key:               "controller.ctrl-01",
			wantComponentType: "controller",
			wantHostname:      "ctrl-01",
			wantOK:            true,
		},
		{
			name:              "unknown prefix returns false",
			key:               "unknown.host-01",
			wantComponentType: "",
			wantHostname:      "",
			wantOK:            false,
		},
		{
			name:              "key without dot returns false",
			key:               "invalid",
			wantComponentType: "",
			wantHostname:      "",
			wantOK:            false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			componentType, hostname, ok := notify.ParseRegistryKey(tt.key)

			s.Equal(tt.wantComponentType, componentType)
			s.Equal(tt.wantHostname, hostname)
			s.Equal(tt.wantOK, ok)
		})
	}
}

func (s *WatcherPublicTestSuite) TestResolveDisplayName() {
	tests := []struct {
		name          string
		componentType string
		identifier    string
		value         []byte
		want          string
	}{
		{
			name:          "non-agent returns identifier as-is",
			componentType: "controller",
			identifier:    "ctrl-01",
			value:         nil,
			want:          "ctrl-01",
		},
		{
			name:          "agent with valid hostname returns hostname",
			componentType: "agent",
			identifier:    "abc123",
			value:         []byte(`{"hostname":"web-01"}`),
			want:          "web-01",
		},
		{
			name:          "agent with invalid JSON returns identifier",
			componentType: "agent",
			identifier:    "abc123",
			value:         []byte("invalid"),
			want:          "abc123",
		},
		{
			name:          "agent with empty hostname returns identifier",
			componentType: "agent",
			identifier:    "abc123",
			value:         []byte(`{"hostname":""}`),
			want:          "abc123",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := notify.ExportResolveDisplayName(
				tt.componentType,
				tt.identifier,
				tt.value,
			)
			s.Equal(tt.want, got)
		})
	}
}

func (s *WatcherPublicTestSuite) TestExtractConditions() {
	tests := []struct {
		name          string
		key           string
		componentType string
		value         func() []byte
		wantOK        bool
		validateFunc  func(conditions []job.Condition)
	}{
		{
			name:          "extracts conditions from agent registration",
			key:           "agents.web-01",
			componentType: "agent",
			value: func() []byte {
				reg := job.AgentRegistration{
					Hostname: "web-01",
					Conditions: []job.Condition{
						{
							Type:   job.ConditionMemoryPressure,
							Status: true,
						},
					},
				}
				b, _ := json.Marshal(reg)
				return b
			},
			wantOK: true,
			validateFunc: func(conditions []job.Condition) {
				s.Require().Len(conditions, 1)
				s.Equal(job.ConditionMemoryPressure, conditions[0].Type)
				s.True(conditions[0].Status)
			},
		},
		{
			name:          "extracts conditions from api component registration",
			key:           "api.api-server-01",
			componentType: "api",
			value: func() []byte {
				reg := job.ComponentRegistration{
					Type:     "api",
					Hostname: "api-server-01",
					Conditions: []job.Condition{
						{
							Type:   job.ConditionHighLoad,
							Status: true,
						},
					},
				}
				b, _ := json.Marshal(reg)
				return b
			},
			wantOK: true,
			validateFunc: func(conditions []job.Condition) {
				s.Require().Len(conditions, 1)
				s.Equal(job.ConditionHighLoad, conditions[0].Type)
			},
		},
		{
			name:          "extracts conditions from nats component registration",
			key:           "nats.nats-01",
			componentType: "nats",
			value: func() []byte {
				reg := job.ComponentRegistration{
					Type:       "nats",
					Hostname:   "nats-01",
					Conditions: []job.Condition{},
				}
				b, _ := json.Marshal(reg)
				return b
			},
			wantOK: true,
			validateFunc: func(conditions []job.Condition) {
				s.Empty(conditions)
			},
		},
		{
			name:          "returns false for invalid agent json",
			key:           "agents.web-01",
			componentType: "agent",
			value: func() []byte {
				return []byte("not-valid-json")
			},
			wantOK: false,
			validateFunc: func(conditions []job.Condition) {
				s.Nil(conditions)
			},
		},
		{
			name:          "returns false for invalid component json",
			key:           "api.api-01",
			componentType: "api",
			value: func() []byte {
				return []byte("not-valid-json")
			},
			wantOK: false,
			validateFunc: func(conditions []job.Condition) {
				s.Nil(conditions)
			},
		},
		{
			name:          "returns false for unknown component type",
			key:           "other.host-01",
			componentType: "other",
			value: func() []byte {
				return []byte("{}")
			},
			wantOK: false,
			validateFunc: func(conditions []job.Condition) {
				s.Nil(conditions)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			conditions, ok := notify.WatcherExtractConditions(
				s.watcher,
				tt.key,
				tt.componentType,
				tt.value(),
			)

			s.Equal(tt.wantOK, ok)
			tt.validateFunc(conditions)
		})
	}
}

func (s *WatcherPublicTestSuite) TestHandleEntry() {
	tests := []struct {
		name         string
		setupEntry   func(ctrl *gomock.Controller) *jobmocks.MockKeyValueEntry
		wantEvents   int
		validateFunc func(events []notify.ConditionEvent)
	}{
		{
			name: "ignores entry with unrecognized key",
			setupEntry: func(ctrl *gomock.Controller) *jobmocks.MockKeyValueEntry {
				entry := jobmocks.NewMockKeyValueEntry(ctrl)
				entry.EXPECT().Key().Return("unknown.host")
				return entry
			},
			wantEvents: 0,
			validateFunc: func(events []notify.ConditionEvent) {
				s.Empty(events)
			},
		},
		{
			name: "handles delete operation and emits ComponentUnreachable",
			setupEntry: func(ctrl *gomock.Controller) *jobmocks.MockKeyValueEntry {
				entry := jobmocks.NewMockKeyValueEntry(ctrl)
				entry.EXPECT().Key().Return("agents.web-01")
				entry.EXPECT().Operation().Return(jetstream.KeyValueDelete)
				return entry
			},
			wantEvents: 1,
			validateFunc: func(events []notify.ConditionEvent) {
				s.Require().Len(events, 1)
				s.Equal("agent", events[0].ComponentType)
				s.Equal("web-01", events[0].Hostname)
				s.Equal("ComponentUnreachable", events[0].Condition)
				s.True(events[0].Active)
				s.Equal("heartbeat expired", events[0].Reason)
			},
		},
		{
			name: "handles purge operation and emits ComponentUnreachable",
			setupEntry: func(ctrl *gomock.Controller) *jobmocks.MockKeyValueEntry {
				// Operation() is called twice: first checks KeyValueDelete (false),
				// then checks KeyValuePurge (true).
				entry := jobmocks.NewMockKeyValueEntry(ctrl)
				entry.EXPECT().Key().Return("nats.nats-01")
				entry.EXPECT().Operation().Return(jetstream.KeyValuePurge).Times(2)
				return entry
			},
			wantEvents: 1,
			validateFunc: func(events []notify.ConditionEvent) {
				s.Require().Len(events, 1)
				s.Equal("nats", events[0].ComponentType)
				s.Equal("nats-01", events[0].Hostname)
				s.Equal("ComponentUnreachable", events[0].Condition)
				s.True(events[0].Active)
			},
		},
		{
			name: "handles put operation with valid agent registration",
			setupEntry: func(ctrl *gomock.Controller) *jobmocks.MockKeyValueEntry {
				reg := job.AgentRegistration{
					Hostname: "web-01",
					Conditions: []job.Condition{
						{Type: job.ConditionMemoryPressure, Status: true},
					},
				}
				b, _ := json.Marshal(reg)

				// Operation() is called twice: first checks KeyValueDelete (false),
				// then checks KeyValuePurge (false), so Put proceeds to Value().
				entry := jobmocks.NewMockKeyValueEntry(ctrl)
				entry.EXPECT().Key().Return("agents.web-01")
				entry.EXPECT().Operation().Return(jetstream.KeyValuePut).Times(2)
				entry.EXPECT().Value().Return(b).AnyTimes()
				return entry
			},
			wantEvents: 1,
			validateFunc: func(events []notify.ConditionEvent) {
				s.Require().Len(events, 1)
				s.Equal("agent", events[0].ComponentType)
				s.Equal("web-01", events[0].Hostname)
				s.Equal(job.ConditionMemoryPressure, events[0].Condition)
				s.True(events[0].Active)
			},
		},
		{
			name: "ignores put operation with invalid json",
			setupEntry: func(ctrl *gomock.Controller) *jobmocks.MockKeyValueEntry {
				// Operation() is called twice: checks KeyValueDelete and KeyValuePurge.
				entry := jobmocks.NewMockKeyValueEntry(ctrl)
				entry.EXPECT().Key().Return("agents.web-01")
				entry.EXPECT().Operation().Return(jetstream.KeyValuePut).Times(2)
				entry.EXPECT().Value().Return([]byte("invalid")).AnyTimes()
				return entry
			},
			wantEvents: 0,
			validateFunc: func(events []notify.ConditionEvent) {
				s.Empty(events)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()

			s.capturedEvts = nil
			notify.WatcherSetPrev(
				s.watcher,
				make(map[string]map[string]*notify.ExportConditionState),
			)

			if tt.wantEvents > 0 {
				s.mockNotifier.EXPECT().
					Notify(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						event notify.ConditionEvent,
					) error {
						s.capturedEvts = append(s.capturedEvts, event)
						return nil
					}).
					Times(tt.wantEvents)
			}

			entry := tt.setupEntry(ctrl)
			notify.WatcherHandleEntry(context.Background(), s.watcher, entry)

			tt.validateFunc(s.capturedEvts)
		})
	}
}

func (s *WatcherPublicTestSuite) TestHandleDelete() {
	tests := []struct {
		name          string
		key           string
		componentType string
		hostname      string
		setupPrev     func()
		validateFunc  func(events []notify.ConditionEvent)
	}{
		{
			name:          "emits ComponentUnreachable event",
			key:           "agents.web-01",
			componentType: "agent",
			hostname:      "web-01",
			setupPrev:     func() {},
			validateFunc: func(events []notify.ConditionEvent) {
				s.Require().Len(events, 1)
				s.Equal("agent", events[0].ComponentType)
				s.Equal("web-01", events[0].Hostname)
				s.Equal("ComponentUnreachable", events[0].Condition)
				s.True(events[0].Active)
				s.Equal("heartbeat expired", events[0].Reason)
			},
		},
		{
			name:          "removes key from prev state",
			key:           "agents.web-01",
			componentType: "agent",
			hostname:      "web-01",
			setupPrev: func() {
				notify.WatcherSetPrevEntry(
					s.watcher,
					"agents.web-01",
					map[string]*notify.ExportConditionState{
						job.ConditionMemoryPressure: notify.NewConditionState(true, time.Now()),
					},
				)
			},
			validateFunc: func(events []notify.ConditionEvent) {
				s.Require().Len(events, 1)
				s.Equal("ComponentUnreachable", events[0].Condition)
				prev := notify.WatcherGetPrev(s.watcher)
				s.Nil(prev["agents.web-01"])
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.capturedEvts = nil
			notify.WatcherSetPrev(
				s.watcher,
				make(map[string]map[string]*notify.ExportConditionState),
			)

			// HandleDelete always emits exactly one ComponentUnreachable event.
			s.mockNotifier.EXPECT().
				Notify(gomock.Any(), gomock.Any()).
				DoAndReturn(func(
					_ context.Context,
					event notify.ConditionEvent,
				) error {
					s.capturedEvts = append(s.capturedEvts, event)
					return nil
				}).
				Times(1)

			tt.setupPrev()

			notify.WatcherHandleDelete(
				context.Background(),
				s.watcher,
				tt.key,
				tt.componentType,
				tt.hostname,
			)

			tt.validateFunc(s.capturedEvts)
		})
	}
}

func (s *WatcherPublicTestSuite) TestStart() {
	tests := []struct {
		name         string
		setup        func(ctrl *gomock.Controller, ctx context.Context) *jobmocks.MockKeyValue
		validateFunc func(err error)
	}{
		{
			name: "returns error when WatchAll fails",
			setup: func(ctrl *gomock.Controller, _ context.Context) *jobmocks.MockKeyValue {
				kv := jobmocks.NewMockKeyValue(ctrl)
				kv.EXPECT().WatchAll(gomock.Any()).Return(nil, errors.New("watch failed"))
				return kv
			},
			validateFunc: func(err error) {
				s.Error(err)
				s.Contains(err.Error(), "watch failed")
			},
		},
		{
			name: "returns nil when context is cancelled",
			setup: func(ctrl *gomock.Controller, _ context.Context) *jobmocks.MockKeyValue {
				ch := make(chan jetstream.KeyValueEntry, 1)

				w := jobmocks.NewMockKeyWatcher(ctrl)
				w.EXPECT().Updates().Return(ch).AnyTimes()
				w.EXPECT().Stop().Return(nil)

				kv := jobmocks.NewMockKeyValue(ctrl)
				kv.EXPECT().WatchAll(gomock.Any()).Return(w, nil)
				return kv
			},
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name: "returns nil when updates channel is closed",
			setup: func(ctrl *gomock.Controller, _ context.Context) *jobmocks.MockKeyValue {
				ch := make(chan jetstream.KeyValueEntry)
				close(ch)

				w := jobmocks.NewMockKeyWatcher(ctrl)
				w.EXPECT().Updates().Return(ch).AnyTimes()
				w.EXPECT().Stop().Return(nil)

				kv := jobmocks.NewMockKeyValue(ctrl)
				kv.EXPECT().WatchAll(gomock.Any()).Return(w, nil)
				return kv
			},
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name: "skips nil entry (initial values sentinel) and exits on closed channel",
			setup: func(ctrl *gomock.Controller, _ context.Context) *jobmocks.MockKeyValue {
				ch := make(chan jetstream.KeyValueEntry, 2)
				ch <- nil // sentinel nil entry
				close(ch)

				w := jobmocks.NewMockKeyWatcher(ctrl)
				w.EXPECT().Updates().Return(ch).AnyTimes()
				w.EXPECT().Stop().Return(nil)

				kv := jobmocks.NewMockKeyValue(ctrl)
				kv.EXPECT().WatchAll(gomock.Any()).Return(w, nil)
				return kv
			},
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name: "logs warning when Stop returns error",
			setup: func(ctrl *gomock.Controller, _ context.Context) *jobmocks.MockKeyValue {
				ch := make(chan jetstream.KeyValueEntry, 1)

				w := jobmocks.NewMockKeyWatcher(ctrl)
				w.EXPECT().Updates().Return(ch).AnyTimes()
				w.EXPECT().Stop().Return(errors.New("stop failed"))

				kv := jobmocks.NewMockKeyValue(ctrl)
				kv.EXPECT().WatchAll(gomock.Any()).Return(w, nil)
				return kv
			},
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name: "processes real entry from updates channel then exits on close",
			setup: func(ctrl *gomock.Controller, _ context.Context) *jobmocks.MockKeyValue {
				// Build an entry with an unrecognized key so handleEntry returns
				// quickly without needing further mock expectations.
				entry := jobmocks.NewMockKeyValueEntry(ctrl)
				entry.EXPECT().Key().Return("unknown.host")

				ch := make(chan jetstream.KeyValueEntry, 2)
				ch <- entry
				close(ch)

				w := jobmocks.NewMockKeyWatcher(ctrl)
				w.EXPECT().Updates().Return(ch).AnyTimes()
				w.EXPECT().Stop().Return(nil)

				kv := jobmocks.NewMockKeyValue(ctrl)
				kv.EXPECT().WatchAll(gomock.Any()).Return(w, nil)
				return kv
			},
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()

			s.capturedEvts = nil
			notify.WatcherSetPrev(
				s.watcher,
				make(map[string]map[string]*notify.ExportConditionState),
			)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			kv := tt.setup(ctrl, ctx)
			notify.WatcherSetKV(s.watcher, kv)

			// Cancel context immediately for cases that need context cancellation.
			// For cases with closed channels, the channel close drives exit.
			switch tt.name {
			case "returns nil when context is cancelled",
				"logs warning when Stop returns error":
				cancel()
			}

			doneCh := make(chan error, 1)
			go func() {
				doneCh <- s.watcher.Start(ctx)
			}()

			select {
			case err := <-doneCh:
				tt.validateFunc(err)
			case <-time.After(2 * time.Second):
				s.Fail("Start did not return within timeout")
			}
		})
	}
}

func TestWatcherPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WatcherPublicTestSuite))
}
