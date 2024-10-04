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
)

// clientNetworkPingCmd represents the clientNetworkPing command.
var clientNetworkPingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping the specified server",
	Long: `Ping the specified server and return results.
`,
	Run: func(cmd *cobra.Command, _ []string) {
		address, _ := cmd.Flags().GetString("address")

		resp, err := handler.PostNetworkPing(context.TODO(), address)
		if err != nil {
			logFatal("failed to post network ping endpoint", err)
		}

		errorMsg := "unknown error"
		switch resp.StatusCode() {
		case http.StatusOK:
			if jsonOutput {
				logger.Info(
					"network ping",
					slog.String("response", string(resp.Body)),
				)
				return
			}

			if resp.JSON200 == nil {
				logFatal("failed response", fmt.Errorf("post network ping was nil"))
			}

			respRows := [][]string{}
			respRows = append(respRows, []string{
				safeString(resp.JSON200.AvgRtt),
				safeString(resp.JSON200.MaxRtt),
				safeString(resp.JSON200.MinRtt),
				float64ToString(resp.JSON200.PacketLoss),
				intToString(resp.JSON200.PacketsReceived),
				intToString(resp.JSON200.PacketsSent),
			})

			sections := []section{
				{
					Title: "Ping Response",
					Headers: []string{
						"AVG RTT",
						"MAX RTT",
						"MIN RTT",
						"PACKET LOSS",
						"PACKETS RECEIVED",
						"PACKETS SENT",
					},
					Rows: respRows,
				},
			}
			printStyledTable(sections)
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
	clientNetworkCmd.AddCommand(clientNetworkPingCmd)

	clientNetworkPingCmd.PersistentFlags().
		StringP("address", "a", "", "The address to ping")

	_ = clientNetworkPingCmd.MarkPersistentFlagRequired("address")
}
