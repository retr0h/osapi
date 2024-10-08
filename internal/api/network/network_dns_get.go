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

package network

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/retr0h/osapi/internal/api/network/gen"
)

// GetNetworkDNS get the network dns get API endpoint.
func (n Network) GetNetworkDNS(
	ctx echo.Context,
) error {
	dnsConfig, err := n.DNSProvider.GetResolvConf()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.NetworkErrorResponse{
			Error: err.Error(),
		})
	}

	searchDomains := dnsConfig.SearchDomains
	servers := dnsConfig.DNSServers

	return ctx.JSON(http.StatusOK, gen.DNSConfigResponse{
		SearchDomains: &searchDomains,
		Servers:       &servers,
	})
}
