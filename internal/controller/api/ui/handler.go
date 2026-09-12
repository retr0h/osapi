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

// Package ui provides the HTTP handler for serving the embedded React UI.
package ui

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

// Handler returns registration functions for the embedded UI. It follows
// the same pattern as other domain handlers in this package. distFS must
// be rooted at the directory containing index.html (the caller is
// responsible for any fs.Sub calls).
func Handler(
	distFS fs.FS,
) []func(e *echo.Echo) {
	return []func(e *echo.Echo){
		func(e *echo.Echo) {
			Register(e, distFS)
		},
	}
}

// Register mounts the UI assets on the Echo router. Static files are served
// directly; all other non-/api paths fall back to index.html so React Router
// can handle client-side routing.
func Register(
	e *echo.Echo,
	distFS fs.FS,
) {
	fileServer := http.FileServer(http.FS(distFS))

	serveIndex := func(w http.ResponseWriter, _ *http.Request) {
		index, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	}

	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			serveIndex(w, r)
			return
		}

		f, err := distFS.Open(path)
		if err != nil {
			serveIndex(w, r)
			return
		}
		_ = f.Close()

		fileServer.ServeHTTP(w, r)
	})))
}
