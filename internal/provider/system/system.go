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

	"github.com/retr0h/osapi/internal/metadata"
	"github.com/retr0h/osapi/internal/metadata/sysinfo"
	"github.com/retr0h/osapi/internal/provider/system/linux"
)

// New factory to create a new instance.
func New(
	appFs afero.Fs,
) *System {
	return &System{
		appFs: appFs,
	}
}

// GetHostname gets the system's hostname.
func (s *System) GetHostname() (string, error) {
	return s.HostnameProvider.GetHostname()
}

// GetUptime gets the system's uptime.
func (s *System) GetUptime() (time.Duration, error) {
	return s.UptimeProvider.GetUptime()
}

// GetLoadAverage gets the system's load average.
func (s *System) GetLoadAverage() ([3]float64, error) {
	return s.LoadProvider.GetLoadAverage()
}

// RegisterProviders register system providers.
func (s *System) RegisterProviders() {
	var sim metadata.SysInfoManager = sysinfo.New(s.appFs)
	si := sim.GetSysInfo()

	switch si.OS.Distribution {
	case "ubuntu":
		// common hostname provider across linux distros
		s.HostnameProvider = linux.NewOSHostnameProvider(s.appFs)
		s.UptimeProvider = linux.NewOSUptimeProvider(s.appFs)
		s.LoadProvider = linux.NewOSLoadProvider(s.appFs)
	}
}
