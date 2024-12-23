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

package client

import (
	"log/slog"
	"net/http"

	"github.com/retr0h/osapi/internal/client/gen"
	"github.com/retr0h/osapi/internal/config"
)

// New factory to create a new instance.
func New(
	logger *slog.Logger,
	appConfig config.Config,
	client *gen.ClientWithResponses,
) *Client {
	return &Client{
		Client:    client,
		logger:    logger,
		appConfig: appConfig,
	}
}

// NewClientWithResponses factory to create new instance.
// hate returning an error here... feels like a design flaw.
func NewClientWithResponses(
	appConfig config.Config,
) (*gen.ClientWithResponses, error) {
	transport := &authTransport{
		base:       http.DefaultTransport,
		authHeader: "Bearer " + appConfig.Client.Security.BearerToken,
	}

	hc := http.Client{
		Transport: transport,
	}

	return gen.NewClientWithResponses(appConfig.Client.URL, gen.WithHTTPClient(&hc))
}

// RoundTrip implements the http.RoundTripper interface.
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.authHeader)
	return t.base.RoundTrip(req)
}
