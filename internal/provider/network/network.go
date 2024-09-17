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

package network

import (
	"time"
)

// DNSConfig represents the DNS configuration with servers and search domains.
type DNSConfig struct {
	// List of DNS server IP addresses (IPv4 or IPv6)
	DNSServers []string
	// List of search domains for DNS resolution
	SearchDomains []string
}

// PingResult represents custom ping result details.
type PingResult struct {
	// Number of packets sent
	PacketsSent int
	// Number of packets received
	PacketsReceived int
	// Percentage of packet loss
	PacketLoss float64
	// Minimum round-trip time
	MinRTT time.Duration
	// Average round-trip time
	AvgRTT time.Duration
	// Maximum round-trip time
	MaxRTT time.Duration
}
