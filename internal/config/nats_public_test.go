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
		name         string
		nats         config.NATS
		validateFunc func([]config.KVBucketInfo)
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
			validateFunc: func(got []config.KVBucketInfo) {
				s.Len(got, len([]string{
					"job-queue",
					"job-responses",
					"registry",
					"facts",
					"state",
					"file-state",
					"enrollment",
				}))
				for i, info := range got {
					s.Equal([]string{
						"job-queue",
						"job-responses",
						"registry",
						"facts",
						"state",
						"file-state",
						"enrollment",
					}[i], info.Name)
					s.Equal([]string{
						"job-queue",
						"job-responses",
						"agent-registry",
						"agent-facts",
						"agent-state",
						"file-state",
						"agent-enrollment",
					}[i], info.Bucket)
				}
			},
		},
		{
			name: "empty config returns slice with empty bucket fields",
			nats: config.NATS{},
			validateFunc: func(got []config.KVBucketInfo) {
				s.Len(got, len([]string{
					"job-queue",
					"job-responses",
					"registry",
					"facts",
					"state",
					"file-state",
					"enrollment",
				}))
				for i, info := range got {
					s.Equal([]string{
						"job-queue",
						"job-responses",
						"registry",
						"facts",
						"state",
						"file-state",
						"enrollment",
					}[i], info.Name)
					s.Equal([]string{"", "", "", "", "", "", ""}[i], info.Bucket)
				}
			},
		},
		{
			name: "partial config — only KV buckets set",
			nats: config.NATS{
				KV: config.NATSKV{
					Bucket:         "job-queue",
					ResponseBucket: "job-responses",
				},
			},
			validateFunc: func(got []config.KVBucketInfo) {
				s.Len(got, len([]string{
					"job-queue",
					"job-responses",
					"registry",
					"facts",
					"state",
					"file-state",
					"enrollment",
				}))
				for i, info := range got {
					s.Equal([]string{
						"job-queue",
						"job-responses",
						"registry",
						"facts",
						"state",
						"file-state",
						"enrollment",
					}[i], info.Name)
					s.Equal([]string{
						"job-queue",
						"job-responses",
						"",
						"",
						"",
						"",
						"",
					}[i], info.Bucket)
				}
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(tt.nats.AllKVBuckets())
		})
	}
}

func (s *NATSPublicTestSuite) TestAllObjectStoreBuckets() {
	tests := []struct {
		name         string
		nats         config.NATS
		validateFunc func([]config.ObjectStoreBucketInfo)
	}{
		{
			name: "objects bucket populated",
			nats: config.NATS{
				Objects: config.NATSObjects{Bucket: "file-objects"},
			},
			validateFunc: func(got []config.ObjectStoreBucketInfo) {
				s.Len(got, len([]string{"file-objects"}))
				for i, info := range got {
					s.Equal([]string{"file-objects"}[i], info.Name)
					s.Equal([]string{"file-objects"}[i], info.Bucket)
				}
			},
		},
		{
			name: "empty config returns slice with empty bucket field",
			nats: config.NATS{},
			validateFunc: func(got []config.ObjectStoreBucketInfo) {
				s.Len(got, len([]string{"file-objects"}))
				for i, info := range got {
					s.Equal([]string{"file-objects"}[i], info.Name)
					s.Equal([]string{""}[i], info.Bucket)
				}
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(tt.nats.AllObjectStoreBuckets())
		})
	}
}

func TestNATSPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NATSPublicTestSuite))
}
