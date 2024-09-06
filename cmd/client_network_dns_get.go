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
	"fmt"
	"log/slog"
	"net/http"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// clientNetworkDNSGetCmd represents the clientNetworkDNSGetCmd command.
var clientNetworkDNSGetCmd = &cobra.Command{
	Use:   "get",
	Short: "DNS of the server",
	Long: `Obtain the current DNS configuration.
`,
	Run: func(_ *cobra.Command, _ []string) {
		resp, err := c.GetNetworkDNS()
		if err != nil {
			logFatal("failed to get network dns endpoint", err)
		}

		switch resp.StatusCode() {
		case http.StatusOK:
			if jsonOutput {
				prettyPrintJSON(resp.Body)
				return
			}

			var searchDomainsList, serversList []string
			if resp.JSON200.SearchDomains != nil {
				searchDomainsList = *resp.JSON200.SearchDomains
			}
			if resp.JSON200.Servers != nil {
				serversList = *resp.JSON200.Servers
			}

			sections := []section{{}}
			dnsConfig := fmt.Sprintf(
				"\n%s: %s\n%s: %s",
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Search Domains"),
				formatList(searchDomainsList),
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Servers"),
				formatList(serversList),
			)

			printStyledTable(sections, dnsConfig)

		default:
			if jsonOutput {
				prettyPrintJSON(resp.Body)
				return
			}

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
