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
	"strings"

	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/token"
)

// tokenGenerateCmd represents the tokenGenerate command.
var tokenGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new token",
	Long: `Generate a new API token with specific roles, expiration, and issuer details.
This command allows you to customize the token properties for various use cases.
`,
	Run: func(cmd *cobra.Command, _ []string) {
		signingKey := appConfig.API.Server.Security.SigningKey
		roles, _ := cmd.Flags().GetStringSlice("roles")
		subject, _ := cmd.Flags().GetString("subject")

		var tm token.Manager = token.New(logger)
		tokin, err := tm.Generate(signingKey, roles, subject)
		if err != nil {
			logFatal("failed to generate token", err)
		}

		logger.Info(
			"generated token",
			slog.String("token", tokin),
			slog.String("roles", strings.Join(roles, ",")),
			slog.String("subject", subject),
		)
	},
}

func init() {
	tokenCmd.AddCommand(tokenGenerateCmd)
	allowedRoles := token.GenerateAllowedRoles(token.RoleHierarchy)
	usage := fmt.Sprintf("Roles for the token (allowed: %s)", strings.Join(allowedRoles, ", "))

	tokenGenerateCmd.PersistentFlags().
		StringSliceP("roles", "r", []string{}, usage)
	tokenGenerateCmd.PersistentFlags().
		StringP("subject", "u", "", "Subject for the token (e.g., user ID or unique identifier)")

	_ = tokenGenerateCmd.MarkPersistentFlagRequired("roles")
	_ = tokenGenerateCmd.MarkPersistentFlagRequired("subject")

	tokenGenerateCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		roles, _ := cmd.Flags().GetStringSlice("roles")

		if err := validateRoles(roles); err != nil {
			logFatal("invalid roles", err, "allowed", allowedRoles)
		}
	}
}

func validateRoles(roles []string) error {
	allowedRoles := token.GenerateAllowedRoles(token.RoleHierarchy)
	allowedRolesMap := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowedRolesMap[role] = struct{}{}
	}

	for _, role := range roles {
		if _, ok := allowedRolesMap[role]; !ok {
			return fmt.Errorf("unsupported role: %s", role)
		}
	}
	return nil
}
