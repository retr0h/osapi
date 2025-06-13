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

package task

import (
	"encoding/json"
)

// ActionType represents the type of task action.
type ActionType string

const (
	// ActionTypeDNS represents a DNS configuration change action.
	ActionTypeDNS ActionType = "dns"
	// ActionTypeShutdown represents a system shutdown or reboot action.
	ActionTypeShutdown ActionType = "shutdown"
)

// ShutdownActionType represents the type of shutdown action.
type ShutdownActionType string

const (
	// ShutdownActionTypeReboot represents a system reboot action.
	ShutdownActionTypeReboot ShutdownActionType = "reboot"
	// ShutdownActionTypeShutdown represents a system shutdown action.
	ShutdownActionTypeShutdown ShutdownActionType = "shutdown"
)

// Task represents a task that can be executed by the worker.
// It contains an action type and the associated action data.
type Task struct {
	// Type specifies the type of action to perform.
	Type ActionType `json:"type"`
	// Data contains the action-specific data as raw JSON.
	Data json.RawMessage `json:"data"`
}

// ChangeDNSAction represents an action to change DNS settings.
type ChangeDNSAction struct {
	// DNSServers is a list of DNS server IP addresses (IPv4 or IPv6).
	DNSServers []string `json:"dns_servers"`
	// SearchDomains is a list of search domains for DNS resolution.
	SearchDomains []string `json:"search_domains"`
	// InterfaceName is the name of the network interface to apply DNS settings to.
	InterfaceName string `json:"interface_name"`
}

// ShutdownAction represents an action to reboot or shutdown the system.
type ShutdownAction struct {
	// ActionType specifies whether to reboot or shutdown the system.
	ActionType ShutdownActionType `json:"action_type"`
	// DelaySeconds is an optional field to specify a delay in seconds before reboot/shutdown.
	DelaySeconds int32 `json:"delay_seconds,omitempty"`
	// Message is an optional message to log or display before reboot/shutdown.
	Message string `json:"message,omitempty"`
}
