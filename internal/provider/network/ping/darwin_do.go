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

package ping

import (
	"context"
	"fmt"
	"time"
)

// Do pings the given host and returns the ping statistics or an error.
//
// On macOS, it uses privileged mode (raw sockets) for ICMP. This may
// require running the binary as root or with appropriate entitlements.
func (d *Darwin) Do(
	address string,
) (*Result, error) {
	pinger, err := d.NewPingerFn(address)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pinger: %w", err)
	}

	pinger.SetCount(3)
	pinger.SetPrivileged(true)

	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultChan := make(chan *Result)
	errorChan := make(chan error)

	go func() {
		err = pinger.Run()
		if err != nil {
			errorChan <- fmt.Errorf("failed to run pinger: %w", err)
			return
		}

		stats := pinger.Statistics()
		result := &Result{
			PacketsSent:     stats.PacketsSent,
			PacketsReceived: stats.PacketsRecv,
			PacketLoss:      stats.PacketLoss,
			MinRTT:          stats.MinRtt,
			AvgRTT:          stats.AvgRtt,
			MaxRTT:          stats.MaxRtt,
		}

		resultChan <- result
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("ping operation timed out after %s", timeout)
	case err := <-errorChan:
		return nil, err
	case result := <-resultChan:
		return result, nil
	}
}
