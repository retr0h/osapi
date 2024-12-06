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

package dns

import (
	"fmt"
	"log/slog"
	"strings"
)

// SetResolvConf updates the DNS configuration for a specific network interface
// using the `resolvectl` command. It applies new DNS servers and search domains
// if provided, while preserving existing settings for values that are not specified.
// The function returns an error if the operation fails.
//
// Cross-platform considerations:
//   - This function is designed specifically for Linux systems that utilize
//     `systemd-resolved` for managing DNS configurations.
//   - It relies on the `resolvectl` command, which is available on systems with
//     `systemd` version 237 or later. On non-systemd systems or older versions of
//     Linux, this functionality may not be available.
//
// Notes about the implementation:
//   - This function queries DNS information dynamically using `resolvectl`, which
//     supports per-interface configurations and reflects the live state of DNS
//     settings managed by `systemd-resolved`.
//   - If no search domains are configured for the interface, the function defaults
//     to returning `["."]` to indicate the root domain.
//
// Requirements:
//   - The `resolvectl` command must be installed and available in the system path.
//   - The caller must have sufficient privileges to query network settings for the
//     specified interface.
//
// See `systemd-resolved.service(8)` manual page for further information.
func (u *Ubuntu) SetResolvConf(
	servers []string,
	searchDomains []string,
) error {
	// TODO(retr0h): parameterize the interface
	const interfaceName = "wlp0s20f3"

	u.logger.Info(
		"setting resolvectl configuration",
		slog.String("servers", strings.Join(servers, ", ")),
		slog.String("search_domains", strings.Join(searchDomains, ", ")),
	)

	if len(servers) == 0 && len(searchDomains) == 0 {
		return fmt.Errorf("no DNS servers or search domains provided; nothing to update")
	}

	existingConfig, err := u.GetResolvConf()
	if err != nil {
		return fmt.Errorf("failed to get current resolvectl configuration: %w", err)
	}

	// Use existing values if new values are not provided
	if len(servers) == 0 {
		servers = existingConfig.DNSServers
	}
	if len(searchDomains) == 0 {
		searchDomains = existingConfig.SearchDomains
	}

	// Set DNS servers
	if len(servers) > 0 {
		cmd := "resolvectl"
		args := append([]string{"dns", interfaceName}, servers...)
		output, err := u.execManager.RunCmd(cmd, args)
		if err != nil {
			return fmt.Errorf("failed to set DNS servers with resolvectl: %w - %s", err, output)
		}
	}

	// Set search domains
	if len(searchDomains) > 0 {
		cmd := "resolvectl"
		args := append([]string{"domain", interfaceName}, searchDomains...)
		output, err := u.execManager.RunCmd(cmd, args)
		if err != nil {
			return fmt.Errorf("failed to set search domains with resolvectl: %w - %s", err, output)
		}
	}

	return nil
}
