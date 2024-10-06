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

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/client"
)

// clientNetworkDNSGetCmd represents the clientNetworkDNSGet command.
var clientNetworkDNSGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the DNS configuration",
	Long: `Get the servers current DNS configuration.
`,
	Run: func(_ *cobra.Command, _ []string) {
		networkHandler := handler.(client.NetworkHandler)
		resp, err := networkHandler.GetNetworkDNS(context.TODO())
		if err != nil {
			logFatal("failed to get network dns endpoint", err)
		}

		switch resp.StatusCode() {
		case http.StatusOK:
			if jsonOutput {
				logger.Info(
					"network dns get",
					slog.String("response", string(resp.Body)),
				)
				return
			}

			if resp.JSON200 == nil {
				logFatal("failed response", fmt.Errorf("get dns response was nil"))
			}

			var searchDomainsList, serversList []string
			if resp.JSON200.SearchDomains != nil {
				searchDomainsList = *resp.JSON200.SearchDomains
			}
			if resp.JSON200.Servers != nil {
				serversList = *resp.JSON200.Servers
			}

			dnsData := map[string]interface{}{
				"Search Domains": formatList(searchDomainsList),
				"Servers":        formatList(serversList),
			}
			printStyledMap(dnsData)

		default:
			errorMsg := "unknown error"
			if resp.JSON500 != nil {
				errorMsg = resp.JSON500.Error
			}

			logger.Error(
				"error in response",
				slog.Int("code", resp.StatusCode()),
				slog.String("error", errorMsg),
			)
		}
	},
}

func init() {
	clientNetworkDNSCmd.AddCommand(clientNetworkDNSGetCmd)
}
