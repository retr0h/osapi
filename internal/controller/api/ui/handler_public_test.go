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

package ui_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	uihandler "github.com/osapi-io/osapi/internal/controller/api/ui"
)

// populatedFS returns an in-memory filesystem shaped like the output of
// fs.Sub(embeddedAssets, "dist") — callers pass it directly to Register or
// Handler with no "dist/" prefix.
func populatedFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":       {Data: []byte("<html>app</html>")},
		"assets/index.js":  {Data: []byte("console.log('app')")},
		"assets/index.css": {Data: []byte("body{}")},
		"favicon.ico":      {Data: []byte("icon")},
	}
}

// emptyFS returns a filesystem with no files, used to exercise the
// "index.html not found" error path in serveIndex.
func emptyFS() fstest.MapFS {
	return fstest.MapFS{}
}

type HandlerPublicTestSuite struct {
	suite.Suite

	echo *echo.Echo
}

func (s *HandlerPublicTestSuite) SetupTest() {
	s.echo = echo.New()
}

func (s *HandlerPublicTestSuite) SetupSubTest() {
	s.SetupTest()
}

func (s *HandlerPublicTestSuite) TestHandler() {
	tests := []struct {
		name         string
		distFS       fs.FS
		validateFunc func(funcs []func(e *echo.Echo))
	}{
		{
			name:   "returns a single registration func that mounts the UI",
			distFS: populatedFS(),
			validateFunc: func(funcs []func(e *echo.Echo)) {
				s.Require().Len(funcs, 1)

				// Execute the registration func and verify it wired the
				// handler into Echo.
				funcs[0](s.echo)

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				rec := httptest.NewRecorder()
				s.echo.ServeHTTP(rec, req)

				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "<html>app</html>")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			funcs := uihandler.Handler(tc.distFS)
			tc.validateFunc(funcs)
		})
	}
}

func (s *HandlerPublicTestSuite) TestRegister() {
	tests := []struct {
		name         string
		distFS       fs.FS
		setupRoute   func(e *echo.Echo)
		method       string
		path         string
		validateFunc func(rec *httptest.ResponseRecorder)
	}{
		{
			name:   "serves a static asset by path",
			distFS: populatedFS(),
			method: http.MethodGet,
			path:   "/assets/index.js",
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "console.log")
			},
		},
		{
			name:   "serves index.html for root path",
			distFS: populatedFS(),
			method: http.MethodGet,
			path:   "/",
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "<html>app</html>")
				s.Equal("text/html; charset=utf-8", rec.Header().Get("Content-Type"))
			},
		},
		{
			name:   "serves index.html for SPA client-side route /configure",
			distFS: populatedFS(),
			method: http.MethodGet,
			path:   "/configure",
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "<html>app</html>")
			},
		},
		{
			name:   "serves index.html for SPA client-side route /roles",
			distFS: populatedFS(),
			method: http.MethodGet,
			path:   "/roles",
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "<html>app</html>")
			},
		},
		{
			name:   "falls back to index.html for unknown asset under non-/api path",
			distFS: populatedFS(),
			method: http.MethodGet,
			path:   "/assets/missing.js",
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "<html>app</html>")
			},
		},
		{
			name:   "does not intercept API routes registered before the handler",
			distFS: populatedFS(),
			setupRoute: func(e *echo.Echo) {
				e.GET("/api/health", func(c echo.Context) error {
					return c.String(http.StatusOK, "healthy")
				})
			},
			method: http.MethodGet,
			path:   "/api/health",
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Equal("healthy", rec.Body.String())
			},
		},
		{
			name:   "returns 404 for unmatched /api path",
			distFS: populatedFS(),
			method: http.MethodGet,
			path:   "/api/missing",
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusNotFound, rec.Code)
			},
		},
		{
			name:   "returns 500 when index.html is missing from the filesystem",
			distFS: emptyFS(),
			method: http.MethodGet,
			path:   "/",
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusInternalServerError, rec.Code)
				s.Contains(rec.Body.String(), "index.html not found")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			if tc.setupRoute != nil {
				tc.setupRoute(s.echo)
			}
			uihandler.Register(s.echo, tc.distFS)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			s.echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestHandlerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HandlerPublicTestSuite))
}
