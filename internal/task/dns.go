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

package task

import (
	"encoding/json"
)

// CreateAndMarshalChangeDNSAction creates a ChangeDNSAction message with the
// given DNS servers and search domains, wraps it in a Task, and marshals it
// to JSON bytes.
func CreateAndMarshalChangeDNSAction(
	servers []string,
	searchDomains []string,
	interfaceName string,
) ([]byte, error) {
	dnsAction := ChangeDNSAction{
		DNSServers:    servers,
		SearchDomains: searchDomains,
		InterfaceName: interfaceName,
	}

	dnsActionData, err := json.Marshal(dnsAction)
	if err != nil {
		return nil, err
	}

	// Wrap ChangeDNSAction into a Task message
	task := Task{
		Type: ActionTypeDNS,
		Data: dnsActionData,
	}

	data, err := json.Marshal(task)
	if err != nil {
		return nil, err
	}

	return data, nil
}
