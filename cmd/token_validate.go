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
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/token"
)

// tokenValidateCmd represents the tokenValidate command.
var tokenValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a token for authenticity and claims",
	Long: `Validate a JSON Web Token (JWT) by checking its signature, expiration, and claims.
This command ensures that the token is authentic, has not expired, and conforms to the expected roles and audience.
`,
	Run: func(cmd *cobra.Command, _ []string) {
		signingKey := appConfig.API.Server.Security.SigningKey
		tokenString, _ := cmd.Flags().GetString("token")

		var tm token.Manager = token.New(logger)
		claims, err := tm.Validate(tokenString, signingKey)
		if err != nil {
			logFatal("failed to validate token", err)
		}

		claimsData := map[string]interface{}{
			"Roles":    strings.Join(claims.Roles, ","),
			"Subject":  claims.Subject,
			"Audience": strings.Join(claims.RegisteredClaims.Audience, ","),
			"Expires":  claims.RegisteredClaims.ExpiresAt.Format(time.RFC3339),
			"Issued":   claims.RegisteredClaims.IssuedAt.Format(time.RFC3339),
		}
		printStyledMap(claimsData)
	},
}

func init() {
	tokenCmd.AddCommand(tokenValidateCmd)

	tokenValidateCmd.PersistentFlags().StringP("token", "t", "", "The Token string")

	_ = tokenValidateCmd.MarkPersistentFlagRequired("token")
}
