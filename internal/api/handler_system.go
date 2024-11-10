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
	"strings"

	"github.com/labstack/echo/v4"
	sysHost "github.com/shirou/gopsutil/v4/host"

	"github.com/retr0h/osapi/internal/api/system"
	systemGen "github.com/retr0h/osapi/internal/api/system/gen"
	"github.com/retr0h/osapi/internal/provider/system/disk"
	"github.com/retr0h/osapi/internal/provider/system/host"
	"github.com/retr0h/osapi/internal/provider/system/load"
	"github.com/retr0h/osapi/internal/provider/system/mem"
)

// GetSystemHandler returns system handler for registration.
func (s *Server) GetSystemHandler() []func(e *echo.Echo) {
	var memProvider mem.Provider
	var loadProvider load.Provider
	var hostProvider host.Provider
	var diskProvider disk.Provider

	info, _ := sysHost.Info()
	switch strings.ToLower(info.Platform) {
	case "ubuntu":
		memProvider = mem.NewUbuntuProvider()
		loadProvider = load.NewUbuntuProvider()
		hostProvider = host.NewUbuntuProvider()
		diskProvider = disk.NewUbuntuProvider(s.logger)
	default:
		memProvider = mem.NewLinuxProvider()
		loadProvider = load.NewLinuxProvider()
		hostProvider = host.NewLinuxProvider()
		diskProvider = disk.NewLinuxProvider()
	}

	return []func(e *echo.Echo){
		func(e *echo.Echo) {
			systemHandler := system.New(
				memProvider,
				loadProvider,
				hostProvider,
				diskProvider,
			)
			systemGen.RegisterHandlers(e, systemHandler)
		},
	}
}
