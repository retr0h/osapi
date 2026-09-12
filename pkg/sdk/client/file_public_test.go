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

package client_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
)

type FilePublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *FilePublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *FilePublicTestSuite) TestUpload() {
	fileContent := []byte("content")
	hash := sha256.Sum256(fileContent)
	contentSHA := fmt.Sprintf("%x", hash)

	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		file         io.Reader
		opts         []client.UploadOption
		validateFunc func(*client.Response[client.FileUpload], error)
	}{
		{
			name: "when uploading new file returns result",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error":"file not found"}`))
					return
				}
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write(
					[]byte(
						`{"name":"nginx.conf","sha256":"abc123","size":1024,"changed":true,"content_type":"raw"}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.FileUpload], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("nginx.conf", resp.Data.Name)
				suite.Equal("abc123", resp.Data.SHA256)
				suite.Equal(1024, resp.Data.Size)
				suite.True(resp.Data.Changed)
				suite.Equal("raw", resp.Data.ContentType)
			},
		},
		{
			name: "when pre-check SHA matches skips upload",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusOK)
					_, _ = fmt.Fprintf(
						w,
						`{"name":"nginx.conf","sha256":"%s","size":7,"content_type":"raw"}`,
						contentSHA,
					)
					return
				}
				// POST should NOT be called — fail if it is.
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"unexpected POST"}`))
			},
			validateFunc: func(resp *client.Response[client.FileUpload], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("nginx.conf", resp.Data.Name)
				suite.Equal(contentSHA, resp.Data.SHA256)
				suite.False(resp.Data.Changed)
				suite.Nil(resp.RawJSON())
			},
		},
		{
			name: "when force skips pre-check and uploads",
			opts: []client.UploadOption{client.WithForce()},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					// GET should NOT be called — fail if it is.
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"unexpected GET"}`))
					return
				}
				suite.Contains(r.URL.RawQuery, "force=true")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write(
					[]byte(
						`{"name":"nginx.conf","sha256":"abc123","size":7,"changed":true,"content_type":"raw"}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.FileUpload], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.True(resp.Data.Changed)
			},
		},
		{
			name: "when server returns 409 returns ConflictError",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error":"file not found"}`))
					return
				}
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":"file already exists"}`))
			},
			validateFunc: func(resp *client.Response[client.FileUpload], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ConflictError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusConflict, target.StatusCode)
			},
		},
		{
			name: "when server returns 400 returns ValidationError",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error":"not found"}`))
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"name is required"}`))
			},
			validateFunc: func(resp *client.Response[client.FileUpload], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusBadRequest, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.FileUpload], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.FileUpload], err error) {
				suite.Error(err)
				suite.Nil(resp)
			},
		},
		{
			name: "when server returns 201 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusCreated)
			},
			validateFunc: func(resp *client.Response[client.FileUpload], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusCreated, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
		{
			name: "when file reader returns error",
			file: &errReader{err: errors.New("read failed")},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
			},
			validateFunc: func(resp *client.Response[client.FileUpload], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "read file")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			file := tc.file
			if file == nil {
				file = bytes.NewReader(fileContent)
			}

			resp, err := sut.File.Upload(
				suite.ctx,
				"nginx.conf",
				"raw",
				file,
				tc.opts...,
			)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *FilePublicTestSuite) TestList() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.FileList], error)
	}{
		{
			name: "when listing files returns results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"files":[{"name":"file1.txt","sha256":"aaa","size":100,"content_type":"raw"}],"total":1}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.FileList], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Files, 1)
				suite.Equal(1, resp.Data.Total)
				suite.Equal("file1.txt", resp.Data.Files[0].Name)
				suite.Equal("raw", resp.Data.Files[0].ContentType)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.FileList], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.FileList], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "list files")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.FileList], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.File.List(suite.ctx)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *FilePublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		fileName     string
		validateFunc func(*client.Response[client.FileMetadata], error)
	}{
		{
			name:     "when getting file returns metadata",
			fileName: "nginx.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"name":"nginx.conf","sha256":"def456","size":512,"content_type":"raw"}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.FileMetadata], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("nginx.conf", resp.Data.Name)
				suite.Equal("def456", resp.Data.SHA256)
				suite.Equal(512, resp.Data.Size)
				suite.Equal("raw", resp.Data.ContentType)
			},
		},
		{
			name:     "when server returns 400 returns ValidationError",
			fileName: "nginx.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid file name"}`))
			},
			validateFunc: func(resp *client.Response[client.FileMetadata], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusBadRequest, target.StatusCode)
			},
		},
		{
			name:     "when server returns 404 returns NotFoundError",
			fileName: "missing.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"file not found"}`))
			},
			validateFunc: func(resp *client.Response[client.FileMetadata], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name:     "when server returns 403 returns AuthError",
			fileName: "nginx.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.FileMetadata], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			fileName:  "nginx.conf",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.FileMetadata], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "get file nginx.conf")
			},
		},
		{
			name:     "when server returns 200 with no JSON body returns UnexpectedStatusError",
			fileName: "nginx.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.FileMetadata], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.File.Get(suite.ctx, tc.fileName)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *FilePublicTestSuite) TestDelete() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		fileName     string
		validateFunc func(*client.Response[client.FileDelete], error)
	}{
		{
			name:     "when deleting file returns result",
			fileName: "old.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(`{"name":"old.conf","deleted":true}`),
				)
			},
			validateFunc: func(resp *client.Response[client.FileDelete], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("old.conf", resp.Data.Name)
				suite.True(resp.Data.Deleted)
			},
		},
		{
			name:     "when server returns 400 returns ValidationError",
			fileName: "old.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid file name"}`))
			},
			validateFunc: func(resp *client.Response[client.FileDelete], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusBadRequest, target.StatusCode)
			},
		},
		{
			name:     "when server returns 404 returns NotFoundError",
			fileName: "missing.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"file not found"}`))
			},
			validateFunc: func(resp *client.Response[client.FileDelete], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name:     "when server returns 403 returns AuthError",
			fileName: "old.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.FileDelete], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			fileName:  "old.conf",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.FileDelete], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "delete file old.conf")
			},
		},
		{
			name:     "when server returns 200 with no JSON body returns UnexpectedStatusError",
			fileName: "old.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.FileDelete], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.File.Delete(suite.ctx, tc.fileName)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *FilePublicTestSuite) TestStale() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.StaleList], error)
	}{
		{
			name: "when stale entries exist returns results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"stale": [
						{
							"object_name": "nginx.conf",
							"hostname": "web-01",
							"path": "/etc/nginx/nginx.conf",
							"deployed_sha": "aaa111",
							"current_sha": "bbb222",
							"deployed_at": "2026-01-15T10:30:00Z"
						}
					],
					"total": 1
				}`))
			},
			validateFunc: func(resp *client.Response[client.StaleList], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal(1, resp.Data.Total)
				suite.Len(resp.Data.Stale, 1)
				suite.Equal("nginx.conf", resp.Data.Stale[0].ObjectName)
				suite.Equal("web-01", resp.Data.Stale[0].Hostname)
				suite.Equal("/etc/nginx/nginx.conf", resp.Data.Stale[0].Path)
				suite.Equal("aaa111", resp.Data.Stale[0].DeployedSHA)
				suite.Equal("bbb222", resp.Data.Stale[0].CurrentSHA)
				suite.Equal("2026-01-15T10:30:00Z", resp.Data.Stale[0].DeployedAt)
			},
		},
		{
			name: "when no stale entries returns empty list",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"stale":[],"total":0}`))
			},
			validateFunc: func(resp *client.Response[client.StaleList], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal(0, resp.Data.Total)
				suite.Empty(resp.Data.Stale)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(resp *client.Response[client.StaleList], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.StaleList], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal server error"}`))
			},
			validateFunc: func(resp *client.Response[client.StaleList], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.StaleList], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.StaleList], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "list stale deployments")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.File.Stale(suite.ctx)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *FilePublicTestSuite) TestChanged() {
	fileContent := []byte("content")
	hash := sha256.Sum256(fileContent)
	contentSHA := fmt.Sprintf("%x", hash)

	differentContent := []byte("different")
	diffHash := sha256.Sum256(differentContent)
	diffSHA := fmt.Sprintf("%x", diffHash)

	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		file         io.Reader
		validateFunc func(*client.Response[client.FileChanged], error)
	}{
		{
			name: "when file does not exist returns changed true",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"file not found"}`))
			},
			validateFunc: func(resp *client.Response[client.FileChanged], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.True(resp.Data.Changed)
				suite.Equal("nginx.conf", resp.Data.Name)
				suite.Equal(contentSHA, resp.Data.SHA256)
			},
		},
		{
			name: "when SHA matches returns changed false",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(
					w,
					`{"name":"nginx.conf","sha256":"%s","size":7,"content_type":"raw"}`,
					contentSHA,
				)
			},
			validateFunc: func(resp *client.Response[client.FileChanged], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.False(resp.Data.Changed)
				suite.Equal(contentSHA, resp.Data.SHA256)
			},
		},
		{
			name: "when SHA differs returns changed true",
			file: bytes.NewReader(differentContent),
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(
					w,
					`{"name":"nginx.conf","sha256":"%s","size":7,"content_type":"raw"}`,
					contentSHA,
				)
			},
			validateFunc: func(resp *client.Response[client.FileChanged], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.True(resp.Data.Changed)
				suite.Equal(diffSHA, resp.Data.SHA256)
			},
		},
		{
			name: "when server returns 403 returns error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.FileChanged], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "check file nginx.conf")
			},
		},
		{
			name: "when file reader returns error",
			file: &errReader{err: errors.New("read failed")},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.FileChanged], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "read file")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			file := tc.file
			if file == nil {
				file = bytes.NewReader(fileContent)
			}

			resp, err := sut.File.Changed(
				suite.ctx,
				"nginx.conf",
				file,
			)
			tc.validateFunc(resp, err)
		})
	}
}

type errReader struct {
	err error
}

func (r *errReader) Read(
	_ []byte,
) (int, error) {
	return 0, r.err
}

func TestFilePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FilePublicTestSuite))
}
