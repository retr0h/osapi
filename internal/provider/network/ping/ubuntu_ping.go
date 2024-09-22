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
	"context"
	"fmt"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// PingHost pings the given host and returns the ping statistics or an error.
// It returns a PingResult struct, and an error if something goes wrong.
//
// On Linux, it attempts an "unprivileged" ping via UDP by default.
// To enable unprivileged pings, run the following command:
//
//	sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
//
// Alternatively, call pinger.SetPrivileged(true) to use raw sockets.
// After doing so, you may either:
//  1. Run the binary as root, or
//  2. Set capabilities on the binary to allow raw socket usage:
//     sudo setcap cap_net_raw=+ep /path/to/your/compiled/binary
func (u *UbuntuPing) PingHost(address string) (*PingResult, error) {
	pinger, err := probing.NewPinger(address)
	timeout := 5 * time.Second

	if err != nil {
		return nil, fmt.Errorf("failed to initialize pinger: %w", err)
	}

	pinger.Count = 3

	// Create a context with a timeout for the entire ping operation
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Run the ping in a separate goroutine to avoid blocking
	resultChan := make(chan *PingResult)
	errorChan := make(chan error)

	go func() {
		err = pinger.Run() // This blocks until pinging is finished
		if err != nil {
			errorChan <- fmt.Errorf("failed to run pinger: %w", err)
			return
		}

		stats := pinger.Statistics()
		result := &PingResult{
			PacketsSent:     stats.PacketsSent,
			PacketsReceived: stats.PacketsRecv,
			PacketLoss:      stats.PacketLoss,
			MinRTT:          stats.MinRtt,
			AvgRTT:          stats.AvgRtt,
			MaxRTT:          stats.MaxRtt,
		}

		resultChan <- result
	}()

	// Use a select statement to enforce the timeout
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("ping operation timed out after %s", timeout)
	case err := <-errorChan:
		return nil, err
	case result := <-resultChan:
		return result, nil
	}
}
