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
	"github.com/retr0h/osapi/internal/task"
)

// CombinedHandler is a superset of all smaller handler interfaces.
type CombinedHandler interface {
	NetworkHandler
	TaskHandler
	SystemHandler
}

// NetworkHandler defines an interface for interacting with Network client operations.
type NetworkHandler interface {
	// GetNetworkDNSByInterface get the network dns get API endpoint.
	GetNetworkDNSByInterface(
		ctx context.Context,
		_ string,
	) (*gen.GetNetworkDNSByInterfaceResponse, error)

	// PutNetworkDNS put the network dns put API endpoint.
	PutNetworkDNS(
		ctx context.Context,
		servers []string,
		searchDomains []string,
		interfaceName string,
	) (*gen.PutNetworkDNSResponse, error)
	// PostNetworkPing post the network ping API endpoint.
	PostNetworkPing(
		ctx context.Context,
		address string,
	) (*gen.PostNetworkPingResponse, error)
}

// TaskHandler defines an interface for interacting with Task client operations.
type TaskHandler interface {
	// DeleteTaskByID deletes a single item through the task API endpoint.
	DeleteTaskByID(
		ctx context.Context,
		messageID uint64,
	) (*gen.DeleteTaskIDResponse, error)
	// GetTaskID fetches a single item through the task API endpoint.
	GetTaskByID(
		ctx context.Context,
		messageID uint64,
	) (*gen.GetTaskIDResponse, error)
	// GetTaskList get all items through the task API endpoint.
	GetTaskList(
		ctx context.Context,
	) (*gen.GetTaskResponse, error)
	// PostTask inserts a single item into the task API endpoint.
	PostTask(
		ctx context.Context,
		taskData task.Task,
	) (*gen.PostTaskResponse, error)
	// GetTaskStatus gets status through the task API endpoint.
	GetTaskStatus(ctx context.Context) (*gen.GetTaskStatusResponse, error)
}

// SystemHandler defines an interface for interacting with System client operations.
type SystemHandler interface {
	// GetSystemStatus get the system status API endpoint.
	GetSystemStatus(ctx context.Context) (*gen.GetSystemStatusResponse, error)
	// GetSystemHostname get the system hostname API endpoint.
	GetSystemHostname(ctx context.Context) (*gen.GetSystemHostnameResponse, error)
}
