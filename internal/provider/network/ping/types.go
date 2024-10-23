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

package ping

import (
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// Provider implements the methods to interact with various ping components.
type Provider interface {
	// Do pings the given host and returns the ping statistics or an error.
	Do(address string) (*Result, error)
}

// Pinger is the interface representing the pinging functionality.
type Pinger interface {
	// SetPrivileged sets whether the pinger will run in privileged mode (using raw sockets).
	// This may require elevated privileges or specific system configurations to work.
	SetPrivileged(privileged bool)
	// Run initiates the ping operation. This method blocks until the operation is completed.
	Run() error
	// Statistics returns the results of the ping operation, including packet loss, RTT, and other metrics.
	Statistics() *probing.Statistics
	// SetCount specifies the number of ping packets to send. A value of 3 means the ping will attempt 3 times.
	SetCount(count int)
}

// PingerWrapper wrapper struct around probing.Pinger.
type PingerWrapper struct {
	*probing.Pinger
}

// Result represents custom ping result details.
type Result struct {
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
