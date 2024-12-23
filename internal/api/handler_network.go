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
	"github.com/retr0h/osapi/internal/exec"
	"github.com/retr0h/osapi/internal/provider/network/dns"
	"github.com/retr0h/osapi/internal/provider/network/ping"
	"github.com/retr0h/osapi/internal/task/client"
)

// GetNetworkHandler returns network handler for registration.
func (s *Server) GetNetworkHandler(
	_ afero.Fs,
	clientManager client.Manager,
) []func(e *echo.Echo) {
	var pingProvider ping.Provider
	var dnsProvider dns.Provider
	var execManager exec.Manager

	info, _ := host.Info()
	switch strings.ToLower(info.Platform) {
	case "ubuntu":
		execManager = exec.New(s.logger)
		pingProvider = ping.NewUbuntuProvider()
		dnsProvider = dns.NewUbuntuProvider(s.logger, execManager)
	default:
		pingProvider = ping.NewLinuxProvider()
		dnsProvider = dns.NewLinuxProvider()
	}

	return []func(e *echo.Echo){
		func(e *echo.Echo) {
			networkHandler := network.New(pingProvider, dnsProvider, clientManager)
			networkGen.RegisterHandlers(e, networkHandler)
		},
	}
}
