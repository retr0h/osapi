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

package token

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Generate generates a signed JWT with the given roles.
func (t *Token) Generate(
	signingKey string,
	roles []string,
	subject string,
) (string, error) {
	claims := CustomClaims{
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "osapi",
			// NOTE(retr0h):
			// - The Audience claim (aud) in a JWT is used to specify the intended
			//   recipients of the token. It ensures the token is only processed by
			//   the services or APIs for which it was issued.
			// - Currently, the Audience claim is included in the token but not
			//   validated during use. In the future, Audience will be used to
			//   restrict tokens to specific services or APIs.
			Audience: jwt.ClaimStrings([]string{
				"https://localhost",
				"http://localhost",
			}),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(0, 3, 0)),
			Subject:   subject,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(signingKey))
}
