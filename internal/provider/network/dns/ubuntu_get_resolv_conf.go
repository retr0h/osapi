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
	"regexp"
	"strings"
)

// GetResolvConf retrieves the DNS configuration for a specific network interface
// using the `resolvectl` command. It returns a Config struct containing the DNS
// servers and search domains for the interface, and an error if something goes wrong.
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
func (u *Ubuntu) GetResolvConf() (*Config, error) {
	// TODO(retr0h): parameterize the interface
	const interfaceName = "wlp0s20f3"

	cmd := "resolvectl"
	args := []string{"status", interfaceName}
	output, err := u.execManager.RunCmd(cmd, args)
	if err != nil {
		return nil, fmt.Errorf("failed to run resolvectl: %w - %s", err, output)
	}

	config := &Config{}

	// Parse DNS Servers
	dnsServersRegex := regexp.MustCompile(`DNS Servers:\s+([^\n]+)`)
	if matches := dnsServersRegex.FindStringSubmatch(output); len(matches) > 1 {
		config.DNSServers = strings.Fields(matches[1])
	}

	// Parse Search Domains
	searchDomainRegex := regexp.MustCompile(`DNS Domain:\s+([^\n]+)`)
	if matches := searchDomainRegex.FindStringSubmatch(output); len(matches) > 1 {
		config.SearchDomains = strings.Fields(matches[1])
	} else {
		config.SearchDomains = []string{"."}
	}

	return config, nil
}
