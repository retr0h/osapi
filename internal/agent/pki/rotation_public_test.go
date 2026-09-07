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
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/agent/pki"
)

type RotationPublicTestSuite struct {
	suite.Suite
}

func (suite *RotationPublicTestSuite) TestVerifyWithGrace() {
	tests := []struct {
		name         string
		setup        func(m *pki.Manager) (data []byte, sig []byte)
		wantVerified bool
	}{
		{
			name: "when signed with current controller key returns true",
			setup: func(m *pki.Manager) ([]byte, []byte) {
				pub, priv, _ := ed25519.GenerateKey(rand.Reader)
				m.SetControllerPublicKey(pub)
				data := []byte("test data")
				sig := ed25519.Sign(priv, data)
				return data, sig
			},
			wantVerified: true,
		},
		{
			name: "when signed with previous controller key returns true",
			setup: func(m *pki.Manager) ([]byte, []byte) {
				oldPub, oldPriv, _ := ed25519.GenerateKey(rand.Reader)
				newPub, _, _ := ed25519.GenerateKey(rand.Reader)
				m.SetControllerPublicKey(oldPub)
				m.RotateControllerKey(newPub)
				data := []byte("test data")
				sig := ed25519.Sign(oldPriv, data)
				return data, sig
			},
			wantVerified: true,
		},
		{
			name: "when signed with both keys set and current key matches returns true",
			setup: func(m *pki.Manager) ([]byte, []byte) {
				oldPub, _, _ := ed25519.GenerateKey(rand.Reader)
				newPub, newPriv, _ := ed25519.GenerateKey(rand.Reader)
				m.SetControllerPublicKey(oldPub)
				m.RotateControllerKey(newPub)
				data := []byte("test data")
				sig := ed25519.Sign(newPriv, data)
				return data, sig
			},
			wantVerified: true,
		},
		{
			name: "when neither key is set returns false",
			setup: func(_ *pki.Manager) ([]byte, []byte) {
				_, priv, _ := ed25519.GenerateKey(rand.Reader)
				data := []byte("test data")
				sig := ed25519.Sign(priv, data)
				return data, sig
			},
			wantVerified: false,
		},
		{
			name: "when signature does not match any key returns false",
			setup: func(m *pki.Manager) ([]byte, []byte) {
				pub1, _, _ := ed25519.GenerateKey(rand.Reader)
				pub2, _, _ := ed25519.GenerateKey(rand.Reader)
				_, unrelatedPriv, _ := ed25519.GenerateKey(rand.Reader)
				m.SetControllerPublicKey(pub1)
				m.RotateControllerKey(pub2)
				data := []byte("test data")
				sig := ed25519.Sign(unrelatedPriv, data)
				return data, sig
			},
			wantVerified: false,
		},
		{
			name: "when only current key is set and signature does not match returns false",
			setup: func(m *pki.Manager) ([]byte, []byte) {
				pub, _, _ := ed25519.GenerateKey(rand.Reader)
				_, wrongPriv, _ := ed25519.GenerateKey(rand.Reader)
				m.SetControllerPublicKey(pub)
				data := []byte("test data")
				sig := ed25519.Sign(wrongPriv, data)
				return data, sig
			},
			wantVerified: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()
			m := pki.New(fs, "/keys", "agent")

			data, sig := tc.setup(m)
			got := m.VerifyWithGrace(data, sig)
			assert.Equal(suite.T(), tc.wantVerified, got)
		})
	}
}

func (suite *RotationPublicTestSuite) TestRotateControllerKey() {
	tests := []struct {
		name         string
		validateFunc func(m *pki.Manager, oldPub ed25519.PublicKey, newPub ed25519.PublicKey)
	}{
		{
			name: "when rotating moves current key to previous",
			validateFunc: func(m *pki.Manager, oldPub ed25519.PublicKey, newPub ed25519.PublicKey) {
				assert.Equal(suite.T(), newPub, m.ControllerPublicKey())
				assert.Equal(suite.T(), oldPub, m.PreviousControllerPublicKey())
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()
			m := pki.New(fs, "/keys", "agent")

			oldPub, _, _ := ed25519.GenerateKey(rand.Reader)
			newPub, _, _ := ed25519.GenerateKey(rand.Reader)

			m.SetControllerPublicKey(oldPub)
			m.RotateControllerKey(newPub)

			tc.validateFunc(m, oldPub, newPub)
		})
	}
}

func (suite *RotationPublicTestSuite) TestRotateControllerKeyChain() {
	tests := []struct {
		name         string
		validateFunc func(
			m *pki.Manager,
			key1 ed25519.PublicKey,
			key2 ed25519.PublicKey,
			key3 ed25519.PublicKey,
		)
	}{
		{
			name: "when rotating twice only keeps last two keys",
			validateFunc: func(
				m *pki.Manager,
				_ ed25519.PublicKey,
				key2 ed25519.PublicKey,
				key3 ed25519.PublicKey,
			) {
				assert.Equal(suite.T(), key3, m.ControllerPublicKey())
				assert.Equal(suite.T(), key2, m.PreviousControllerPublicKey())
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()
			m := pki.New(fs, "/keys", "agent")

			key1, _, _ := ed25519.GenerateKey(rand.Reader)
			key2, _, _ := ed25519.GenerateKey(rand.Reader)
			key3, _, _ := ed25519.GenerateKey(rand.Reader)

			m.SetControllerPublicKey(key1)
			m.RotateControllerKey(key2)
			m.RotateControllerKey(key3)

			tc.validateFunc(m, key1, key2, key3)
		})
	}
}

func (suite *RotationPublicTestSuite) TestPreviousControllerPublicKey() {
	tests := []struct {
		name         string
		setup        func(m *pki.Manager)
		validateFunc func(m *pki.Manager)
	}{
		{
			name:  "when no rotation has occurred returns nil",
			setup: func(_ *pki.Manager) {},
			validateFunc: func(m *pki.Manager) {
				assert.Nil(suite.T(), m.PreviousControllerPublicKey())
			},
		},
		{
			name: "when controller key set but not rotated returns nil",
			setup: func(m *pki.Manager) {
				pub, _, _ := ed25519.GenerateKey(rand.Reader)
				m.SetControllerPublicKey(pub)
			},
			validateFunc: func(m *pki.Manager) {
				assert.Nil(suite.T(), m.PreviousControllerPublicKey())
			},
		},
		{
			name: "when rotated returns previous key",
			setup: func(m *pki.Manager) {
				pub1, _, _ := ed25519.GenerateKey(rand.Reader)
				pub2, _, _ := ed25519.GenerateKey(rand.Reader)
				m.SetControllerPublicKey(pub1)
				m.RotateControllerKey(pub2)
			},
			validateFunc: func(m *pki.Manager) {
				require.NotNil(suite.T(), m.PreviousControllerPublicKey())
				assert.Len(suite.T(), m.PreviousControllerPublicKey(), ed25519.PublicKeySize)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()
			m := pki.New(fs, "/keys", "agent")

			tc.setup(m)
			tc.validateFunc(m)
		})
	}
}

func TestRotationPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(RotationPublicTestSuite))
}
