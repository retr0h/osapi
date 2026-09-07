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

package pki_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/failfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/agent/pki"
)

type KeypairPublicTestSuite struct {
	suite.Suite
}

func (suite *KeypairPublicTestSuite) TearDownSubTest() {
	pki.ResetGenerateKeyPairFn()
}

func (suite *KeypairPublicTestSuite) TestLoadOrGenerate() {
	tests := []struct {
		name         string
		setup        func(m *pki.Manager)
		setupFn      func()
		wantErr      bool
		wantContains string
		validateFunc func(m *pki.Manager)
	}{
		{
			name:  "when no keys exist generates new keypair",
			setup: func(_ *pki.Manager) {},
			validateFunc: func(m *pki.Manager) {
				assert.Len(suite.T(), m.PublicKey(), ed25519.PublicKeySize)
				assert.Len(suite.T(), m.PrivateKey(), ed25519.PrivateKeySize)
			},
		},
		{
			name: "when valid keys exist loads from disk",
			setup: func(m *pki.Manager) {
				// Generate and save first.
				err := m.LoadOrGenerate()
				require.NoError(suite.T(), err)
			},
			validateFunc: func(m *pki.Manager) {
				assert.Len(suite.T(), m.PublicKey(), ed25519.PublicKeySize)
				assert.Len(suite.T(), m.PrivateKey(), ed25519.PrivateKeySize)
			},
		},
		{
			name:  "when generate keypair fails",
			setup: func(_ *pki.Manager) {},
			setupFn: func() {
				pki.SetGenerateKeyPairFn(func() (ed25519.PublicKey, ed25519.PrivateKey, error) {
					return nil, nil, fmt.Errorf("entropy exhausted")
				})
			},
			wantErr:      true,
			wantContains: "generate keypair",
		},
		{
			name: "when private key PEM is invalid",
			setup: func(m *pki.Manager) {
				fs := memfs.New()
				_ = fs.MkdirAll("/keys", 0o700)
				_ = fs.WriteFile("/keys/agent.key", []byte("not a pem"), 0o600)
				pubKey, _, _ := ed25519.GenerateKey(rand.Reader)
				pubPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PUBLIC KEY",
					Bytes: pubKey,
				})
				_ = fs.WriteFile("/keys/agent.pub", pubPEM, 0o644)
				*m = *pki.New(fs, "/keys", "agent")
			},
			wantErr:      true,
			wantContains: "decode private key PEM",
		},
		{
			name: "when public key PEM is invalid",
			setup: func(m *pki.Manager) {
				fs := memfs.New()
				_ = fs.MkdirAll("/keys", 0o700)
				_, privKey, _ := ed25519.GenerateKey(rand.Reader)
				privPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PRIVATE KEY",
					Bytes: privKey.Seed(),
				})
				_ = fs.WriteFile("/keys/agent.key", privPEM, 0o600)
				_ = fs.WriteFile("/keys/agent.pub", []byte("not a pem"), 0o644)
				*m = *pki.New(fs, "/keys", "agent")
			},
			wantErr:      true,
			wantContains: "decode public key PEM",
		},
		{
			name: "when private key PEM has wrong block type",
			setup: func(m *pki.Manager) {
				fs := memfs.New()
				_ = fs.MkdirAll("/keys", 0o700)
				_, privKey, _ := ed25519.GenerateKey(rand.Reader)
				privPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "RSA PRIVATE KEY",
					Bytes: privKey.Seed(),
				})
				pubKey := privKey.Public().(ed25519.PublicKey)
				pubPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PUBLIC KEY",
					Bytes: pubKey,
				})
				_ = fs.WriteFile("/keys/agent.key", privPEM, 0o600)
				_ = fs.WriteFile("/keys/agent.pub", pubPEM, 0o644)
				*m = *pki.New(fs, "/keys", "agent")
			},
			wantErr:      true,
			wantContains: "unexpected block type",
		},
		{
			name: "when public key PEM has wrong block type",
			setup: func(m *pki.Manager) {
				fs := memfs.New()
				_ = fs.MkdirAll("/keys", 0o700)
				_, privKey, _ := ed25519.GenerateKey(rand.Reader)
				privPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PRIVATE KEY",
					Bytes: privKey.Seed(),
				})
				pubKey := privKey.Public().(ed25519.PublicKey)
				pubPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "RSA PUBLIC KEY",
					Bytes: pubKey,
				})
				_ = fs.WriteFile("/keys/agent.key", privPEM, 0o600)
				_ = fs.WriteFile("/keys/agent.pub", pubPEM, 0o644)
				*m = *pki.New(fs, "/keys", "agent")
			},
			wantErr:      true,
			wantContains: "unexpected block type",
		},
		{
			name: "when key directory creation fails",
			setup: func(m *pki.Manager) {
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
				*m = *pki.New(vfs, "/keys", "agent")
			},
			wantErr:      true,
			wantContains: "create key directory",
		},
		{
			name: "when writing private key fails",
			setup: func(m *pki.Manager) {
				vfs := failfs.New(memfs.New())
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
				*m = *pki.New(vfs, "/keys", "agent")
			},
			wantErr:      true,
			wantContains: "write private key",
		},
		{
			name: "when writing public key fails",
			setup: func(m *pki.Manager) {
				var callCount atomic.Int32
				vfs := failfs.New(memfs.New())
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnOpenFile {
						// Let early OpenFile calls succeed (private
						// key write), then fail for the public key.
						if callCount.Add(1) > 3 {
							return errors.New("disk full")
						}
					}
					return nil
				})
				*m = *pki.New(vfs, "/keys", "agent")
			},
			wantErr:      true,
			wantContains: "write public key",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()
			m := pki.New(fs, "/keys", "agent")

			tc.setup(m)
			if tc.setupFn != nil {
				tc.setupFn()
			}

			err := m.LoadOrGenerate()

			if tc.wantErr {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantContains)
			} else {
				require.NoError(suite.T(), err)
				if tc.validateFunc != nil {
					tc.validateFunc(m)
				}
			}
		})
	}
}

func (suite *KeypairPublicTestSuite) TestLoadOrGenerateRoundTrip() {
	tests := []struct {
		name         string
		validateFunc func(original, loaded *pki.Manager)
	}{
		{
			name: "when keys are saved and reloaded they match",
			validateFunc: func(original, loaded *pki.Manager) {
				assert.Equal(suite.T(), original.PublicKey(), loaded.PublicKey())
				assert.Equal(suite.T(), original.PrivateKey(), loaded.PrivateKey())
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()

			original := pki.New(fs, "/keys", "agent")
			err := original.LoadOrGenerate()
			require.NoError(suite.T(), err)

			loaded := pki.New(fs, "/keys", "agent")
			err = loaded.LoadOrGenerate()
			require.NoError(suite.T(), err)

			tc.validateFunc(original, loaded)
		})
	}
}

func (suite *KeypairPublicTestSuite) TestSignAndVerify() {
	tests := []struct {
		name         string
		data         []byte
		tamper       bool
		validateFunc func(bool)
	}{
		{
			name: "when data is signed and verified",
			data: []byte("hello world"),
			validateFunc: func(got bool) {
				assert.Equal(suite.T(), true, got)
			},
		},
		{
			name:   "when data is tampered after signing",
			data:   []byte("hello world"),
			tamper: true,
			validateFunc: func(got bool) {
				assert.Equal(suite.T(), false, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()
			m := pki.New(fs, "/keys", "agent")
			err := m.LoadOrGenerate()
			require.NoError(suite.T(), err)

			sig := m.Sign(tc.data)

			verifyData := tc.data
			if tc.tamper {
				verifyData = []byte("tampered data")
			}

			got := pki.Verify(m.PublicKey(), verifyData, sig)
			tc.validateFunc(got)
		})
	}
}

func (suite *KeypairPublicTestSuite) TestFingerprint() {
	tests := []struct {
		name         string
		setup        func() *pki.Manager
		wantEmpty    bool
		wantPrefix   string
		validateFunc func(fp string)
	}{
		{
			name: "when public key is set returns SHA256 fingerprint",
			setup: func() *pki.Manager {
				fs := memfs.New()
				m := pki.New(fs, "/keys", "agent")
				_ = m.LoadOrGenerate()
				return m
			},
			wantPrefix: "SHA256:",
			validateFunc: func(fp string) {
				// SHA256: prefix + 64 hex chars.
				assert.True(suite.T(), strings.HasPrefix(fp, "SHA256:"))
				hexPart := strings.TrimPrefix(fp, "SHA256:")
				assert.Len(suite.T(), hexPart, 64)
			},
		},
		{
			name: "when public key is empty returns empty string",
			setup: func() *pki.Manager {
				fs := memfs.New()
				return pki.New(fs, "/keys", "agent")
			},
			wantEmpty: true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			m := tc.setup()
			got := m.Fingerprint()

			if tc.wantEmpty {
				assert.Empty(suite.T(), got)
			} else if tc.validateFunc != nil {
				tc.validateFunc(got)
			}
		})
	}
}

func (suite *KeypairPublicTestSuite) TestControllerPublicKey() {
	tests := []struct {
		name         string
		setKey       ed25519.PublicKey
		validateFunc func(m *pki.Manager)
	}{
		{
			name: "when controller key is not set returns nil",
			validateFunc: func(m *pki.Manager) {
				assert.Nil(suite.T(), m.ControllerPublicKey())
			},
		},
		{
			name: "when controller key is set returns the key",
			setKey: func() ed25519.PublicKey {
				pub, _, _ := ed25519.GenerateKey(rand.Reader)
				return pub
			}(),
			validateFunc: func(m *pki.Manager) {
				assert.Len(suite.T(), m.ControllerPublicKey(), ed25519.PublicKeySize)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()
			m := pki.New(fs, "/keys", "agent")

			if tc.setKey != nil {
				m.SetControllerPublicKey(tc.setKey)
			}

			tc.validateFunc(m)
		})
	}
}

func (suite *KeypairPublicTestSuite) TestParsePublicKeyPEM() {
	tests := []struct {
		name         string
		pemData      []byte
		wantErr      bool
		wantContains string
		validateFunc func(key ed25519.PublicKey)
	}{
		{
			name: "when valid PEM parses successfully",
			pemData: func() []byte {
				pub, _, _ := ed25519.GenerateKey(rand.Reader)
				return pem.EncodeToMemory(&pem.Block{
					Type:  "ED25519 PUBLIC KEY",
					Bytes: pub,
				})
			}(),
			validateFunc: func(key ed25519.PublicKey) {
				assert.Len(suite.T(), key, ed25519.PublicKeySize)
			},
		},
		{
			name:         "when PEM data is not valid returns error",
			pemData:      []byte("not valid pem data"),
			wantErr:      true,
			wantContains: "failed to decode PEM block",
		},
		{
			name: "when PEM has wrong block type returns error",
			pemData: func() []byte {
				pub, _, _ := ed25519.GenerateKey(rand.Reader)
				return pem.EncodeToMemory(&pem.Block{
					Type:  "RSA PUBLIC KEY",
					Bytes: pub,
				})
			}(),
			wantErr:      true,
			wantContains: "unexpected block type",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			key, err := pki.ParsePublicKeyPEM(tc.pemData)

			if tc.wantErr {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantContains)
			} else {
				require.NoError(suite.T(), err)
				if tc.validateFunc != nil {
					tc.validateFunc(key)
				}
			}
		})
	}
}

func TestKeypairPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(KeypairPublicTestSuite))
}
