// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
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

// GetResolvConfByInterface retrieves the DNS configuration for a specific
// network interface using the `scutil --dns` command on macOS.
//
// It parses resolver blocks from scutil output, matching by interface name
// via the `if_index` field. If no resolver matches the requested interface,
// it returns an error.
//
// Example scutil --dns output:
//
//	resolver #1
//	  search domain[0] : example.com
//	  nameserver[0] : 192.168.1.1
//	  nameserver[1] : 8.8.8.8
//	  if_index : 6 (en0)
func (d *Darwin) GetResolvConfByInterface(
	interfaceName string,
) (*GetResult, error) {
	output, err := d.execManager.RunCmd("scutil", []string{"--dns"})
	if err != nil {
		return nil, fmt.Errorf("failed to run scutil --dns: %w - %s", err, output)
	}

	return parseScutilDNS(output, interfaceName)
}

// resolverBlock represents a parsed resolver block from scutil --dns output.
type resolverBlock struct {
	nameservers   []string
	searchDomains []string
	ifaceName     string
}

// parseScutilDNS parses `scutil --dns` output and returns DNS configuration
// for the requested interface.
func parseScutilDNS(
	output string,
	interfaceName string,
) (*GetResult, error) {
	blocks := splitResolverBlocks(output)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no resolver blocks found in scutil output")
	}

	// Look for a resolver matching the requested interface
	for _, block := range blocks {
		if block.ifaceName == interfaceName {
			return &GetResult{
				DNSServers:    block.nameservers,
				SearchDomains: block.searchDomains,
			}, nil
		}
	}

	return nil, fmt.Errorf("interface %q does not exist", interfaceName)
}

var (
	nameserverRegex   = regexp.MustCompile(`nameserver\[\d+\]\s*:\s*(\S+)`)
	searchDomainRegex = regexp.MustCompile(`search domain\[\d+\]\s*:\s*(\S+)`)
	ifIndexRegex      = regexp.MustCompile(`if_index\s*:\s*\d+\s*\((\S+)\)`)
)

// splitResolverBlocks splits scutil --dns output into individual resolver blocks.
func splitResolverBlocks(
	output string,
) []resolverBlock {
	// Split on "resolver #" to get individual blocks
	parts := strings.Split(output, "resolver #")
	var blocks []resolverBlock

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}

		block := resolverBlock{}

		for _, match := range nameserverRegex.FindAllStringSubmatch(part, -1) {
			block.nameservers = append(block.nameservers, match[1])
		}

		for _, match := range searchDomainRegex.FindAllStringSubmatch(part, -1) {
			block.searchDomains = append(block.searchDomains, match[1])
		}

		if match := ifIndexRegex.FindStringSubmatch(part); len(match) > 1 {
			block.ifaceName = match[1]
		}

		// Only include blocks that have nameservers
		if len(block.nameservers) > 0 {
			blocks = append(blocks, block)
		}
	}

	return blocks
}
