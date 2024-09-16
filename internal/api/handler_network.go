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
	"github.com/shirou/gopsutil/v4/host"
	"github.com/spf13/afero"

	"github.com/retr0h/osapi/internal/api/network"
	networkGen "github.com/retr0h/osapi/internal/api/network/gen"
	networkImpl "github.com/retr0h/osapi/internal/provider/network"
)

// GetNetworkHandler returns network handler for registration.
func (s *Server) GetNetworkHandler(
	appFs afero.Fs,
) []func(e *echo.Echo) {
	var networkProvider networkImpl.Provider

	info, _ := host.Info()
	switch strings.ToLower(info.Platform) {
	case "ubuntu":
		networkProvider = networkImpl.NewUbuntuProvider(appFs)
	default:
		networkProvider = networkImpl.NewDefaultLinuxProvider()
	}

	return []func(e *echo.Echo){
		func(e *echo.Echo) {
			networkHandler := network.New(networkProvider)
			networkGen.RegisterHandlers(e, networkHandler)
		},
	}
}
