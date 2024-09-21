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
	"bufio"
	"fmt"
	"strings"
)

// GetResolvConf reads the DNS configuration from /run/systemd/resolve/resolv.conf.
// It returns a Config struct, and an error if something goes wrong.
//
// Cross-platform considerations: This function is designed specifically for Linux systems
// that utilize systemd-resolved for managing DNS configurations. It reads from
// /run/systemd/resolve/resolv.conf, which is commonly used when systemd-resolved is active.
// For non-systemd Linux systems or non-Linux Unix-like systems (e.g., BSD or macOS), this
// file may not exist or may be irrelevant.
//
// Important Notice:
//   - This function requires privilege escalation to operate correctly.
//
// About /run/systemd/resolve/resolv.conf:
//   - This file is managed by systemd-resolved and contains the current DNS servers and
//     search domains configured on the system.
//   - Unlike /etc/resolv.conf, this file is specific to systemd-resolved and reflects its
//     settings, rather than being a general resolv.conf file used by other resolvers.
//   - When systemd-resolved is running, /etc/resolv.conf might be symlinked to this file,
//     making it a reliable location for DNS information on systems using systemd.
//
// Mocking:
//   - Afero is used for file system abstraction, which allows for easier testing and mocking
//     of file reads.
//
// See `systemd-resolved.service(8)` manual page for further information.
func (u *UbuntuDNS) GetResolvConf() (*Config, error) {
	const resolvConfFile = "/run/systemd/resolve/resolv.conf"

	file, err := u.appFs.Open(resolvConfFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", resolvConfFile, err)
	}
	defer func() { _ = file.Close() }()

	config := &Config{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "nameserver":
			config.DNSServers = append(config.DNSServers, fields[1])
		case "search", "domain":
			config.SearchDomains = append(config.SearchDomains, fields[1:]...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", resolvConfFile, err)
	}

	return config, nil
}
