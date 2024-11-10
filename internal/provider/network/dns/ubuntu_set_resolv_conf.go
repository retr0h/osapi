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
	"os"
	"strings"
)

// SetResolvConf updates the resolv.conf file with new DNS servers and search
// domains, preserving existing settings if new values are not provided.
// The function returns an error if the operation fails.
//
// Important Notice:
//   - This function writes directly to /run/systemd/resolve/resolv.conf, which
//     may be overwritten by systemd-resolved. In the future, consider moving to
//     resolvectl for better integration with systemd-resolved, as it manages
//     DNS settings dynamically and supports per-interface configurations.
//
// Mocking:
//   - Afero is used for file system abstraction, which allows for easier
//     testing and mocking of file reads.
//
// See `systemd-resolved.service(8)` manual page for further information.
func (u *Ubuntu) SetResolvConf(
	servers []string,
	searchDomains []string,
) error {
	const resolvConfFile = "/run/systemd/resolve/resolv.conf"
	const tmpResolvConfFile = "/run/systemd/resolve/resolv.conf.tmp"

	u.logger.Info(
		"setting resolv conf",
		slog.String("servers", strings.Join(servers, ", ")),
		slog.String("search_domains", strings.Join(searchDomains, ", ")),
	)

	if len(servers) == 0 && len(searchDomains) == 0 {
		return fmt.Errorf("no DNS servers or search domains provided; nothing to update")
	}

	existingConfig, err := u.GetResolvConf()
	if err != nil {
		return fmt.Errorf("failed to get current resolv.conf: %w", err)
	}

	// Use existing values if new values are not provided
	if len(servers) == 0 {
		servers = existingConfig.DNSServers
	}
	if len(searchDomains) == 0 {
		searchDomains = existingConfig.SearchDomains
	}

	file, err := u.appFs.OpenFile(tmpResolvConfFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open temporary file %s: %w", tmpResolvConfFile, err)
	}
	defer func() { _ = file.Close() }()

	// Write the DNS servers, if any are provided
	if len(servers) > 0 {
		for _, server := range servers {
			if _, err := file.WriteString(fmt.Sprintf("nameserver %s\n", server)); err != nil {
				return fmt.Errorf("failed to write nameserver %s to temp file: %w", server, err)
			}
		}
	}

	// Write the search domains, if any are provided
	if len(searchDomains) > 0 {
		searchLine := fmt.Sprintf("search %s\n", strings.Join(searchDomains, " "))
		if _, err := file.WriteString(searchLine); err != nil {
			return fmt.Errorf("failed to write search domains to temp file: %w", err)
		}
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close temp file %s: %w", tmpResolvConfFile, err)
	}

	// Atomically move the temporary file to replace the resolv.conf file
	if err := u.appFs.Rename(tmpResolvConfFile, resolvConfFile); err != nil {
		return fmt.Errorf("failed to move temp file to %s: %w", resolvConfFile, err)
	}

	return nil
}
