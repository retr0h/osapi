// Copyright (c) 2024 John Dewey

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

package api

import (
	"github.com/labstack/echo/v4"
	"github.com/spf13/afero"

	"github.com/retr0h/osapi/internal/api/queue"
	queueGen "github.com/retr0h/osapi/internal/api/queue/gen"
	queueImpl "github.com/retr0h/osapi/internal/queue"
)

// CreateHandlers initializes handlers and returns a slice of functions to register them.
func (s *Server) CreateHandlers(
	appFs afero.Fs,
	queueManager queueImpl.Manager,
) []func(e *echo.Echo) {
	handlers := []func(e *echo.Echo){
		func(e *echo.Echo) {
			queueHandler := queue.New(queueManager)
			queueGen.RegisterHandlers(e, queueHandler)
		},
	}

	systemHandler := s.GetSystemHandler()
	networkHandler := s.GetNetworkHandler(appFs, queueManager)

	handlers = append(handlers, systemHandler...)
	handlers = append(handlers, networkHandler...)

	return handlers
}

// RegisterHandlers registers a list of handlers with the Echo instance.
func (s *Server) RegisterHandlers(handlers []func(e *echo.Echo)) {
	for _, handler := range handlers {
		handler(s.Echo)
	}
}
