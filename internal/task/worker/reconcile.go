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

package worker

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shirou/gopsutil/v4/host"

	"github.com/retr0h/osapi/internal/exec"
	"github.com/retr0h/osapi/internal/provider/network/dns"
	"github.com/retr0h/osapi/internal/task"
)

// reconcile applies the desired state to the system by making necessary changes.
//
// This function compares the current system state with the desired state, which is
// serialized as JSON actions in the message queue, and performs any required actions
// to bring the system into alignment with the desired configuration.
// It may involve tasks such as modifying configurations, starting/stopping services, or
// other system-level operations. If any changes are applied, the function ensures they are
// completed successfully or logs errors if the changes fail.
//
// If an action does not complete successfully, the corresponding message is placed back
// on the queue and will be retried until successful, following a retry strategy.
//
// The reconciliation process follows the principle of idempotency, ensuring that repeated
// invocations of the function will produce the same outcome without unintended side effects.
func (w *Worker) reconcile(
	seq uint64,
	data []byte,
) error {
	var t task.Task

	err := task.UnmarshalJSON(data, &t)
	if err != nil {
		return fmt.Errorf("failed to unmarshal task id: %d %w", seq, err)
	}

	w.logger.Info(
		"reconciling message",
		slog.Uint64("id", seq),
		slog.String("type", string(t.Type)),
	)

	switch t.Type {
	case task.ActionTypeShutdown:
		return w.handleShutdownAction(t.Data)
	case task.ActionTypeDNS:
		return w.handleDNSAction(t.Data)
	default:
		return fmt.Errorf("unknown task action type: %s", t.Type)
	}
}

// handleShutdownAction processes shutdown/reboot actions.
func (w *Worker) handleShutdownAction(data json.RawMessage) error {
	var shutdownAction task.ShutdownAction

	err := json.Unmarshal(data, &shutdownAction)
	if err != nil {
		return fmt.Errorf("failed to unmarshal shutdown action: %w", err)
	}

	w.logger.Info(
		"processing shutdown action",
		slog.String("action_type", string(shutdownAction.ActionType)),
		slog.Int("delay_seconds", int(shutdownAction.DelaySeconds)),
		slog.String("message", shutdownAction.Message),
	)

	// TODO: implement shutdown/reboot logic
	return nil
}

// handleDNSAction processes DNS configuration change actions.
func (w *Worker) handleDNSAction(data json.RawMessage) error {
	var dnsAction task.ChangeDNSAction

	err := json.Unmarshal(data, &dnsAction)
	if err != nil {
		return fmt.Errorf("failed to unmarshal DNS action: %w", err)
	}

	w.logger.Info(
		"processing DNS action",
		slog.String("interface", dnsAction.InterfaceName),
		slog.Any("dns_servers", dnsAction.DNSServers),
		slog.Any("search_domains", dnsAction.SearchDomains),
	)

	dnsProvider := GetDNSProvider(w.logger)
	return dnsProvider.SetResolvConfByInterface(
		dnsAction.DNSServers,
		dnsAction.SearchDomains,
		dnsAction.InterfaceName,
	)
}

// GetDNSProvider returns a DNS provider handler based on the system platform.
// It selects the appropriate DNS provider implementation depending on the OS platform
// detected (e.g., Ubuntu-specific vs. general Linux). This allows platform-specific
// DNS configurations to be handled in a unified manner.
func GetDNSProvider(
	logger *slog.Logger,
) dns.Provider {
	var dnsProvider dns.Provider
	var execManager exec.Manager

	info, _ := host.Info()

	switch strings.ToLower(info.Platform) {
	case "ubuntu":
		execManager = exec.New(logger)
		dnsProvider = dns.NewUbuntuProvider(logger, execManager)
	default:
		dnsProvider = dns.NewLinuxProvider()
	}

	return dnsProvider
}
