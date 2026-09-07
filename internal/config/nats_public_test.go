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

package config_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/config"
)

type NATSPublicTestSuite struct {
	suite.Suite
}

func (s *NATSPublicTestSuite) TestAllKVBuckets() {
	tests := []struct {
		name            string
		nats            config.NATS
		expectedNames   []string
		expectedBuckets []string
	}{
		{
			name: "all buckets populated",
			nats: config.NATS{
				KV: config.NATSKV{
					Bucket:         "job-queue",
					ResponseBucket: "job-responses",
				},
				Registry:   config.NATSRegistry{Bucket: "agent-registry"},
				Facts:      config.NATSFacts{Bucket: "agent-facts"},
				State:      config.NATSState{Bucket: "agent-state"},
				FileState:  config.NATSFileState{Bucket: "file-state"},
				Enrollment: config.NATSEnrollment{Bucket: "agent-enrollment"},
			},
			expectedNames: []string{
				"job-queue",
				"job-responses",
				"registry",
				"facts",
				"state",
				"file-state",
				"enrollment",
			},
			expectedBuckets: []string{
				"job-queue",
				"job-responses",
				"agent-registry",
				"agent-facts",
				"agent-state",
				"file-state",
				"agent-enrollment",
			},
		},
		{
			name: "empty config returns slice with empty bucket fields",
			nats: config.NATS{},
			expectedNames: []string{
				"job-queue",
				"job-responses",
				"registry",
				"facts",
				"state",
				"file-state",
				"enrollment",
			},
			expectedBuckets: []string{"", "", "", "", "", "", ""},
		},
		{
			name: "partial config — only KV buckets set",
			nats: config.NATS{
				KV: config.NATSKV{
					Bucket:         "job-queue",
					ResponseBucket: "job-responses",
				},
			},
			expectedNames: []string{
				"job-queue",
				"job-responses",
				"registry",
				"facts",
				"state",
				"file-state",
				"enrollment",
			},
			expectedBuckets: []string{
				"job-queue",
				"job-responses",
				"",
				"",
				"",
				"",
				"",
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := tt.nats.AllKVBuckets()

			s.Len(got, len(tt.expectedNames))
			for i, info := range got {
				s.Equal(tt.expectedNames[i], info.Name)
				s.Equal(tt.expectedBuckets[i], info.Bucket)
			}
		})
	}
}

func (s *NATSPublicTestSuite) TestAllObjectStoreBuckets() {
	tests := []struct {
		name            string
		nats            config.NATS
		expectedNames   []string
		expectedBuckets []string
	}{
		{
			name: "objects bucket populated",
			nats: config.NATS{
				Objects: config.NATSObjects{Bucket: "file-objects"},
			},
			expectedNames:   []string{"file-objects"},
			expectedBuckets: []string{"file-objects"},
		},
		{
			name:            "empty config returns slice with empty bucket field",
			nats:            config.NATS{},
			expectedNames:   []string{"file-objects"},
			expectedBuckets: []string{""},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := tt.nats.AllObjectStoreBuckets()

			s.Len(got, len(tt.expectedNames))
			for i, info := range got {
				s.Equal(tt.expectedNames[i], info.Name)
				s.Equal(tt.expectedBuckets[i], info.Bucket)
			}
		})
	}
}

func TestNATSPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NATSPublicTestSuite))
}
