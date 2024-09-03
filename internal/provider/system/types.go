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

package system

import (
	"time"

	"github.com/spf13/afero"
)

// System implementation of the System operations.
type System struct {
	HostnameProvider HostnameProvider
	UptimeProvider   UptimeProvider
	LoadProvider     LoadProvider
	MemoryProvider   MemoryProvider

	appFs afero.Fs
}

// HostnameProvider is an internal only interface that abstracts the hostname function.
type HostnameProvider interface {
	GetHostname() (string, error)
}

// UptimeProvider is an internal only interface that abstracts the uptime function.
type UptimeProvider interface {
	GetUptime() (time.Duration, error)
}

// LoadProvider is an internal only interface that abstracts the getloadavg function.
type LoadProvider interface {
	GetLoadAverage() ([3]float32, error)
}

// MemoryProvider is an internal only interface that abstracts the meminfo function.
type MemoryProvider interface {
	GetMemory() ([]uint64, error)
}
