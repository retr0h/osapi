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
	"github.com/spf13/afero"

	"github.com/retr0h/osapi/internal/manager/system/ubuntu"
	"github.com/retr0h/osapi/internal/metadata"
	"github.com/retr0h/osapi/internal/metadata/sysinfo"
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

// RegisterProviders register system providers.
func (s *System) RegisterProviders() {
	var sim metadata.SysInfoManager = sysinfo.New(s.appFs)
	si := sim.GetSysInfo()

	switch si.OS.Distribution {
	case "ubuntu":
		s.HostnameProvider = ubuntu.NewOSHostnameProvider()
	case "":
		// TODO(retr0h): remove
		s.HostnameProvider = ubuntu.NewOSHostnameProvider()
	}
}