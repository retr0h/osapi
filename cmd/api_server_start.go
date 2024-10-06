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
	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/task/client"
)

// apiServerStartCmd represents the apiServerStart command.
var apiServerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server",
	Long: `Start the API server.
`,
	Run: func(_ *cobra.Command, _ []string) {
		var clientManager client.Manager = client.New(appConfig, logger)

		err := clientManager.Connect()
		if err != nil {
			logFatal("failed to set up client", err)
		}

		var sm api.ServerManager = api.New(appConfig, logger)
		handlers := sm.CreateHandlers(appFs, clientManager)
		sm.RegisterHandlers(handlers)

		err = sm.Start()
		if err != nil {
			logFatal("failed to start server", err)
		}
	},
}

func init() {
	apiServerCmd.AddCommand(apiServerStartCmd)
}
