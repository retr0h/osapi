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

package agent_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/failfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
	agentMocks "github.com/osapi-io/osapi/internal/agent/mocks"
	"github.com/osapi-io/osapi/internal/agent/pki"
	"github.com/osapi-io/osapi/internal/config"
	jobMocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type EnrollmentPublicTestSuite struct {
	suite.Suite
}

func (suite *EnrollmentPublicTestSuite) TestHandlePKIEnrollment() {
	tests := []struct {
		name         string
		setupFS      func() avfs.VFS
		wantErr      bool
		wantContains string
		validateFunc func(a *agent.Agent)
	}{
		{
			name: "when no existing keys generates keypair and enters pending state",
			setupFS: func() avfs.VFS {
				return memfs.New()
			},
			validateFunc: func(a *agent.Agent) {
				m := agent.GetAgentPKIManager(a)
				require.NotNil(suite.T(), m)
				assert.Len(suite.T(), m.PublicKey(), ed25519.PublicKeySize)
				assert.Len(suite.T(), m.PrivateKey(), ed25519.PrivateKeySize)
				assert.NotEmpty(suite.T(), m.Fingerprint())
				// No controller key — not enrolled.
				assert.Nil(suite.T(), m.ControllerPublicKey())
			},
		},
		{
			name: "when controller public key exists loads and marks enrolled",
			setupFS: func() avfs.VFS {
				fs := memfs.New()
				_ = fs.MkdirAll("/keys", 0o700)

				// Pre-generate agent keys.
				pub, priv, _ := ed25519.GenerateKey(rand.Reader)
				privPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PRIVATE KEY",
					Bytes: priv.Seed(),
				})
				pubPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PUBLIC KEY",
					Bytes: pub,
				})
				_ = fs.WriteFile("/keys/agent.key", privPEM, 0o600)
				_ = fs.WriteFile("/keys/agent.pub", pubPEM, 0o644)

				// Write controller public key.
				ctrlPub, _, _ := ed25519.GenerateKey(rand.Reader)
				ctrlPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PUBLIC KEY",
					Bytes: ctrlPub,
				})
				_ = fs.WriteFile("/keys/controller.pub", ctrlPEM, 0o644)

				return fs
			},
			validateFunc: func(a *agent.Agent) {
				m := agent.GetAgentPKIManager(a)
				require.NotNil(suite.T(), m)
				assert.Len(suite.T(), m.PublicKey(), ed25519.PublicKeySize)
				assert.Len(suite.T(), m.ControllerPublicKey(), ed25519.PublicKeySize)
			},
		},
		{
			name: "when controller public key PEM is corrupted returns error",
			setupFS: func() avfs.VFS {
				fs := memfs.New()
				_ = fs.MkdirAll("/keys", 0o700)

				// Pre-generate agent keys.
				pub, priv, _ := ed25519.GenerateKey(rand.Reader)
				privPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PRIVATE KEY",
					Bytes: priv.Seed(),
				})
				pubPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PUBLIC KEY",
					Bytes: pub,
				})
				_ = fs.WriteFile("/keys/agent.key", privPEM, 0o600)
				_ = fs.WriteFile("/keys/agent.pub", pubPEM, 0o644)

				// Write corrupted controller public key.
				_ = fs.WriteFile("/keys/controller.pub", []byte("not valid pem"), 0o644)

				return fs
			},
			wantErr:      true,
			wantContains: "failed to parse controller public key",
		},
		{
			name: "when controller public key PEM has wrong block type returns error",
			setupFS: func() avfs.VFS {
				fs := memfs.New()
				_ = fs.MkdirAll("/keys", 0o700)

				// Pre-generate agent keys.
				pub, priv, _ := ed25519.GenerateKey(rand.Reader)
				privPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PRIVATE KEY",
					Bytes: priv.Seed(),
				})
				pubPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PUBLIC KEY",
					Bytes: pub,
				})
				_ = fs.WriteFile("/keys/agent.key", privPEM, 0o600)
				_ = fs.WriteFile("/keys/agent.pub", pubPEM, 0o644)

				// Write controller key with wrong block type.
				ctrlPub, _, _ := ed25519.GenerateKey(rand.Reader)
				ctrlPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "RSA PUBLIC KEY",
					Bytes: ctrlPub,
				})
				_ = fs.WriteFile("/keys/controller.pub", ctrlPEM, 0o644)

				return fs
			},
			wantErr:      true,
			wantContains: "failed to parse controller public key",
		},
		{
			name: "when keypair generation fails returns error",
			setupFS: func() avfs.VFS {
				vfs := failfs.New(memfs.New())
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnMkdirAll {
						return errors.New("permission denied")
					}
					return nil
				})

				return vfs
			},
			wantErr:      true,
			wantContains: "create key directory",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := tc.setupFS()

			logger := slog.Default()
			cfg := config.Config{
				Agent: config.AgentConfig{
					PKI: config.AgentPKI{
						Enabled: true,
						KeyDir:  "/keys",
					},
				},
			}

			a := agent.New(
				fs,
				cfg,
				logger,
				nil, // jobClient
				"",  // streamName
				nil, // hostProvider
				nil, // diskProvider
				nil, // memProvider
				nil, // loadProvider
				nil, // netinfoProvider
				nil, // processProvider
				nil, // registry
				nil, // registryKV
				nil, // factsKV
				nil, // execManager
				nil, // natsClient
			)

			err := agent.ExportHandlePKIEnrollment(context.Background(), a)

			if tc.wantErr {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantContains)
			} else {
				require.NoError(suite.T(), err)
				if tc.validateFunc != nil {
					tc.validateFunc(a)
				}
			}
		})
	}
}

func (suite *EnrollmentPublicTestSuite) TestPublishEnrollmentRequest() {
	tests := []struct {
		name         string
		setupAgent   func() *agent.Agent
		wantErr      bool
		wantContains string
	}{
		{
			name: "when natsClient is nil returns error",
			setupAgent: func() *agent.Agent {
				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				)

				// Set up a PKI manager so PublicKey/Fingerprint don't panic.
				m := pki.New(fs, "/keys", "agent")
				require.NoError(suite.T(), m.LoadOrGenerate())
				agent.SetAgentPKIManager(a, m)

				return a
			},
			wantErr:      true,
			wantContains: "NATS client not available",
		},
		{
			name: "when publish succeeds without namespace",
			setupAgent: func() *agent.Agent {
				ctrl := gomock.NewController(suite.T())
				mockNATS := agentMocks.NewMockNATSPublisher(ctrl)
				mockNATS.EXPECT().
					PublishCore("enroll.request", gomock.Any()).
					Return(nil)

				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockNATS,
				)

				m := pki.New(fs, "/keys", "agent")
				require.NoError(suite.T(), m.LoadOrGenerate())
				agent.SetAgentPKIManager(a, m)

				return a
			},
			wantErr: false,
		},
		{
			name: "when publish succeeds with namespace",
			setupAgent: func() *agent.Agent {
				ctrl := gomock.NewController(suite.T())
				mockNATS := agentMocks.NewMockNATSPublisher(ctrl)
				mockNATS.EXPECT().
					PublishCore("osapi.enroll.request", gomock.Any()).
					Return(nil)

				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						NATS: config.NATSConnection{
							Namespace: "osapi",
						},
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockNATS,
				)

				m := pki.New(fs, "/keys", "agent")
				require.NoError(suite.T(), m.LoadOrGenerate())
				agent.SetAgentPKIManager(a, m)

				return a
			},
			wantErr: false,
		},
		{
			name: "when publish fails returns error",
			setupAgent: func() *agent.Agent {
				ctrl := gomock.NewController(suite.T())
				mockNATS := agentMocks.NewMockNATSPublisher(ctrl)
				mockNATS.EXPECT().
					PublishCore(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("connection refused"))

				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockNATS,
				)

				m := pki.New(fs, "/keys", "agent")
				require.NoError(suite.T(), m.LoadOrGenerate())
				agent.SetAgentPKIManager(a, m)

				return a
			},
			wantErr:      true,
			wantContains: "publish enrollment request",
		},
		{
			name: "when marshal fails returns error",
			setupAgent: func() *agent.Agent {
				agent.SetMarshalJSONEnrollment(func(_ interface{}) ([]byte, error) {
					return nil, fmt.Errorf("marshal boom")
				})
				suite.T().Cleanup(func() {
					agent.ResetMarshalJSONEnrollment()
				})

				ctrl := gomock.NewController(suite.T())
				mockNATS := agentMocks.NewMockNATSPublisher(ctrl)

				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockNATS,
				)

				m := pki.New(fs, "/keys", "agent")
				require.NoError(suite.T(), m.LoadOrGenerate())
				agent.SetAgentPKIManager(a, m)

				return a
			},
			wantErr:      true,
			wantContains: "marshal enrollment request",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			a := tc.setupAgent()

			err := agent.ExportPublishEnrollmentRequest(a)

			if tc.wantErr {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantContains)
			} else {
				require.NoError(suite.T(), err)
			}
		})
	}
}

func (suite *EnrollmentPublicTestSuite) TestStartEnrollmentListener() {
	tests := []struct {
		name         string
		setupAgent   func() (*agent.Agent, context.Context, context.CancelFunc)
		validateFunc func(a *agent.Agent, cancel context.CancelFunc)
	}{
		{
			name: "when natsClient is nil returns without subscribing",
			setupAgent: func() (*agent.Agent, context.Context, context.CancelFunc) {
				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				)

				agent.SetAgentMachineID(a, "test-machine")
				ctx, cancel := context.WithCancel(context.Background())

				return a, ctx, cancel
			},
			validateFunc: func(a *agent.Agent, _ context.CancelFunc) {
				// natsClient is nil, so no subscription is created and
				// no goroutine is started. WaitGroup should complete
				// immediately, proving nothing was spawned.
				done := make(chan struct{})
				go func() {
					agent.WaitAgentWG(a)
					close(done)
				}()
				select {
				case <-done:
					// WaitGroup completed immediately — no goroutine was started.
				case <-time.After(time.Second):
					suite.Fail("WaitGroup did not complete — goroutine was unexpectedly started")
				}
			},
		},
		{
			name: "when subscribe succeeds subscription is created and handler is callable",
			setupAgent: func() (*agent.Agent, context.Context, context.CancelFunc) {
				ctrl := gomock.NewController(suite.T())
				mockNATS := agentMocks.NewMockNATSPublisher(ctrl)
				mockNATS.EXPECT().
					Subscribe("enroll.response.test-machine", gomock.Any()).
					DoAndReturn(func(
						_ string,
						handler nats.MsgHandler,
					) (*nats.Subscription, error) {
						// Invoke the handler with invalid JSON to verify the
						// closure is wired correctly and early-returns on bad input.
						handler(&nats.Msg{Data: []byte("bad")})

						return &nats.Subscription{}, nil
					})

				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockNATS,
				)

				agent.SetAgentMachineID(a, "test-machine")
				ctx, cancel := context.WithCancel(context.Background())

				return a, ctx, cancel
			},
			validateFunc: func(a *agent.Agent, cancel context.CancelFunc) {
				// The handler was invoked with invalid JSON, so the
				// agent should NOT have transitioned to Ready.
				assert.Empty(
					suite.T(),
					agent.GetAgentState(a),
					"agent state should remain empty after invalid JSON",
				)
				// Goroutine was started; cancel to allow cleanup.
				cancel()
				agent.WaitAgentWG(a)
			},
		},
		{
			name: "when subscribe succeeds with namespace subject is prefixed",
			setupAgent: func() (*agent.Agent, context.Context, context.CancelFunc) {
				ctrl := gomock.NewController(suite.T())
				mockNATS := agentMocks.NewMockNATSPublisher(ctrl)
				mockNATS.EXPECT().
					Subscribe("osapi.enroll.response.test-machine", gomock.Any()).
					Return(&nats.Subscription{}, nil)

				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						NATS: config.NATSConnection{
							Namespace: "osapi",
						},
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockNATS,
				)

				agent.SetAgentMachineID(a, "test-machine")
				ctx, cancel := context.WithCancel(context.Background())

				return a, ctx, cancel
			},
			validateFunc: func(a *agent.Agent, cancel context.CancelFunc) {
				cancel()
				agent.WaitAgentWG(a)
			},
		},
		{
			name: "when subscribe fails logs warning and returns",
			setupAgent: func() (*agent.Agent, context.Context, context.CancelFunc) {
				ctrl := gomock.NewController(suite.T())
				mockNATS := agentMocks.NewMockNATSPublisher(ctrl)
				mockNATS.EXPECT().
					Subscribe(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("subscribe failed"))

				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockNATS,
				)

				agent.SetAgentMachineID(a, "test-machine")
				ctx, cancel := context.WithCancel(context.Background())

				return a, ctx, cancel
			},
			validateFunc: func(a *agent.Agent, _ context.CancelFunc) {
				// Subscribe failed, so no goroutine should be started.
				// WaitGroup should complete immediately.
				done := make(chan struct{})
				go func() {
					agent.WaitAgentWG(a)
					close(done)
				}()
				select {
				case <-done:
					// WaitGroup completed immediately — no goroutine was started.
				case <-time.After(time.Second):
					suite.Fail("WaitGroup did not complete — goroutine was unexpectedly started")
				}
			},
		},
		{
			name: "when context is cancelled unsubscribes",
			setupAgent: func() (*agent.Agent, context.Context, context.CancelFunc) {
				ctrl := gomock.NewController(suite.T())
				mockNATS := agentMocks.NewMockNATSPublisher(ctrl)
				mockNATS.EXPECT().
					Subscribe(gomock.Any(), gomock.Any()).
					Return(&nats.Subscription{}, nil)

				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockNATS,
				)

				agent.SetAgentMachineID(a, "test-machine")
				ctx, cancel := context.WithCancel(context.Background())

				return a, ctx, cancel
			},
			validateFunc: func(a *agent.Agent, cancel context.CancelFunc) {
				// Cancel triggers unsubscribe in the goroutine.
				cancel()
				agent.WaitAgentWG(a)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			a, ctx, cancel := tc.setupAgent()
			defer cancel()

			agent.ExportStartEnrollmentListener(ctx, a)

			tc.validateFunc(a, cancel)
		})
	}
}

func (suite *EnrollmentPublicTestSuite) TestHandleEnrollmentResponse() {
	tests := []struct {
		name         string
		setupAgent   func() *agent.Agent
		msg          *nats.Msg
		validateFunc func(a *agent.Agent)
	}{
		{
			name: "when invalid JSON logs warning and returns",
			setupAgent: func() *agent.Agent {
				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				)

				return a
			},
			msg: &nats.Msg{Data: []byte("not valid json")},
			validateFunc: func(a *agent.Agent) {
				// Invalid JSON should not change agent state.
				assert.Empty(
					suite.T(),
					agent.GetAgentState(a),
					"agent state should remain empty after invalid JSON",
				)
			},
		},
		{
			name: "when accepted is false logs rejection and returns",
			setupAgent: func() *agent.Agent {
				fs := memfs.New()
				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				)

				return a
			},
			msg: &nats.Msg{Data: []byte(`{"accepted":false,"reason":"denied by admin"}`)},
			validateFunc: func(a *agent.Agent) {
				// Rejection should leave agent state unchanged (empty).
				assert.Empty(
					suite.T(),
					agent.GetAgentState(a),
					"agent state should remain empty after rejection",
				)
			},
		},
		{
			name: "when accepted is true saves controller key and transitions to ready",
			setupAgent: func() *agent.Agent {
				ctrl := gomock.NewController(suite.T())
				mockJobClient := jobMocks.NewMockJobClient(ctrl)
				mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
				mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					AnyTimes()

				fs := memfs.New()
				_ = fs.MkdirAll("/keys", 0o700)

				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					fs, cfg, logger,
					mockJobClient, "test-stream",
					nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				)

				// Set up PKI manager.
				m := pki.New(fs, "/keys", "agent")
				require.NoError(suite.T(), m.LoadOrGenerate())
				agent.SetAgentPKIManager(a, m)

				agent.SetAgentMachineID(a, "test-machine")

				// Set agent context so startConsumers can create child context.
				ctx, cancel := context.WithCancel(context.Background())
				suite.T().Cleanup(cancel)
				agent.SetAgentLifecycle(ctx, ctx, a, cancel, cancel)

				return a
			},
			msg: func() *nats.Msg {
				ctrlPub, _, _ := ed25519.GenerateKey(rand.Reader)
				resp := pki.EnrollmentResponse{
					Accepted:            true,
					ControllerPublicKey: ctrlPub,
				}
				data, _ := json.Marshal(resp)

				return &nats.Msg{Data: data}
			}(),
			validateFunc: func(a *agent.Agent) {
				assert.Equal(suite.T(), "Ready", agent.GetAgentState(a))

				m := agent.GetAgentPKIManager(a)
				assert.NotNil(suite.T(), m.ControllerPublicKey())
				assert.Len(suite.T(), m.ControllerPublicKey(), ed25519.PublicKeySize)
			},
		},
		{
			name: "when accepted but file write fails logs error and returns",
			setupAgent: func() *agent.Agent {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/keys", 0o700)

				// Pre-generate agent keys on the base fs before wrapping.
				m := pki.New(baseFs, "/keys", "agent")
				require.NoError(suite.T(), m.LoadOrGenerate())

				vfs := failfs.New(baseFs)
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnOpenFile {
						return errors.New("disk full")
					}

					return nil
				})

				logger := slog.Default()
				cfg := config.Config{
					Agent: config.AgentConfig{
						PKI: config.AgentPKI{
							Enabled: true,
							KeyDir:  "/keys",
						},
					},
				}

				a := agent.New(
					vfs, cfg, logger,
					nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				)

				// Use the manager created from the base FS (keys already loaded).
				agent.SetAgentPKIManager(a, m)

				return a
			},
			msg: func() *nats.Msg {
				ctrlPub, _, _ := ed25519.GenerateKey(rand.Reader)
				resp := pki.EnrollmentResponse{
					Accepted:            true,
					ControllerPublicKey: ctrlPub,
				}
				data, _ := json.Marshal(resp)

				return &nats.Msg{Data: data}
			}(),
			validateFunc: func(a *agent.Agent) {
				// File write failed — state should NOT be Ready.
				assert.NotEqual(suite.T(), "Ready", agent.GetAgentState(a))
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			a := tc.setupAgent()

			agent.ExportHandleEnrollmentResponse(a, tc.msg)

			tc.validateFunc(a)
		})
	}
}

func TestEnrollmentPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(EnrollmentPublicTestSuite))
}
