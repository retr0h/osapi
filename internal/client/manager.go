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
	"context"

	"github.com/retr0h/osapi/internal/client/gen"
)

// Manager defines an interface for interacting with various client
// services and operations.
type Manager interface {
	// GetNetworkDNS get the network dns get API endpoint.
	GetNetworkDNS(
		ctx context.Context,
	) (*gen.GetNetworkDNSResponse, error)
	// PostNetworkPing post the network ping API endpoint.
	PostNetworkPing(
		ctx context.Context,
		address string,
	) (*gen.PostNetworkPingResponse, error)

	// DeleteQueueByID deletes a single item through the queue API endpoint.
	DeleteQueueByID(
		ctx context.Context,
		messageID string,
	) (*gen.DeleteQueueIDResponse, error)
	// GetQueueID fetches a single item through the queue API endpoint.
	GetQueueByID(
		ctx context.Context,
		messageID string,
	) (*gen.GetQueueIDResponse, error)
	// GetQueueAll gets all items through the queue API endpoint.
	GetQueueAll(
		ctx context.Context,
		limit int,
		offset int,
	) (*gen.GetQueueResponse, error)
	// PostQueue inserts a single item into the queue API endpoint.
	PostQueue(
		ctx context.Context,
		messageBody string,
	) (*gen.PostQueueResponse, error)

	// GetSystemStatus get the system status API endpoint.
	GetSystemStatus(
		ctx context.Context,
	) (*gen.GetSystemStatusResponse, error)
	// GetSystemHostname get the system hostname API endpoint.
	GetSystemHostname(
		ctx context.Context,
	) (*gen.GetSystemHostnameResponse, error)
}
