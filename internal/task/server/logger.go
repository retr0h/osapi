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

package server

import (
	"fmt"
	"log/slog"
)

// SlogWrapper wraps an slog.Logger to implement the NATS Logger interface.
// Note: slog is designed for structured logging, using key-value pairs.
// However, since the NATS logger interface uses printf-style formatting
// (e.g., "%s", "%q"), we use fmt.Sprintf to manually format the message
// before passing it to slog. This ensures compatibility between NATS's
// formatting needs and slog's structured logging.
type SlogWrapper struct {
	logger *slog.Logger
}

// Noticef logs a formatted notice message.
func (l *SlogWrapper) Noticef(format string, v ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, v...))
}

// Warnf logs a formatted warning message.
func (l *SlogWrapper) Warnf(format string, v ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, v...))
}

// Fatalf logs a formatted fatal error message.
func (l *SlogWrapper) Fatalf(format string, v ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, v...))
}

// Errorf logs a formatted error message.
func (l *SlogWrapper) Errorf(format string, v ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, v...))
}

// Debugf logs a formatted debug message.
func (l *SlogWrapper) Debugf(format string, v ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}

// Tracef logs a formatted trace message (NATS requires this for tracing).
func (l *SlogWrapper) Tracef(format string, v ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, v...)) // Map Trace to Debug in slog
}
