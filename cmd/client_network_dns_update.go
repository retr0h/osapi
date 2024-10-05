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
	"log/slog"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/client"
)

// clientNetworkDNSUpdateCmd represents the clientNetworkDNSUpdate command.
var clientNetworkDNSUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the DNS configuration",
	Long: `Update the current DNS configuration with the supplied options.
`,
	Run: func(cmd *cobra.Command, _ []string) {
		servers, _ := cmd.Flags().GetStringSlice("servers")
		searchDomains, _ := cmd.Flags().GetStringSlice("search-domains")

		networkHandler := handler.(client.NetworkHandler)
		resp, err := networkHandler.PutNetworkDNS(context.TODO(), servers, searchDomains)
		if err != nil {
			logFatal("failed to update network dns endpoint", err)
		}

		errorMsg := "unknown error"
		switch resp.StatusCode() {
		case http.StatusAccepted:
			logger.Info(
				"network dns put",
				slog.String("search_domains", strings.Join(searchDomains, ",")),
				slog.String("servers", strings.Join(servers, ",")),
				slog.String("response", string(resp.Body)),
				slog.String("status", "ok"),
			)

		case http.StatusBadRequest:
			if resp.JSON400 != nil {
				errorMsg = resp.JSON400.Error
			}

			logger.Error(
				"bad request",
				slog.Int("code", resp.StatusCode()),
				slog.String("response", errorMsg),
			)

		default:
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
	clientNetworkDNSCmd.AddCommand(clientNetworkDNSUpdateCmd)

	clientNetworkDNSUpdateCmd.PersistentFlags().
		StringSlice("servers", []string{}, "List of DNS server IP addresses (comma-separated)")
	clientNetworkDNSUpdateCmd.PersistentFlags().
		StringSlice("search-domains", []string{}, "List of DNS search domains (comma-separated)")

	clientNetworkDNSUpdateCmd.MarkFlagsOneRequired("servers", "search-domains")
}
