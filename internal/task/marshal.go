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
	"fmt"
)

// MarshalJSON is a generic function to marshal any data structure to JSON.
func MarshalJSON(data interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	return jsonData, nil
}

// UnmarshalJSON is a generic function to unmarshal JSON data.
func UnmarshalJSON(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// MarshalJSONIndent formats a data structure into a multi-line indented JSON string.
func MarshalJSONIndent(data interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to indented JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// SafeMarshalTaskToString safely converts the provided JSON data into a
// string representation of Task.
func SafeMarshalTaskToString(body *[]byte) string {
	var taskMessage Task

	if body == nil {
		return "N/A"
	}

	err := UnmarshalJSON(*body, &taskMessage)
	if err != nil {
		return err.Error()
	}

	jsonString, err := MarshalJSONIndent(taskMessage)
	if err != nil {
		return err.Error()
	}

	return jsonString
}

// SafeMarshalTaskToSummary safely converts the provided JSON data into a
// concise human-readable summary of the Task.
func SafeMarshalTaskToSummary(body *[]byte) string {
	var taskMessage Task

	if body == nil {
		return "N/A"
	}

	err := UnmarshalJSON(*body, &taskMessage)
	if err != nil {
		return "Error: " + err.Error()
	}

	switch taskMessage.Type {
	case ActionTypeDNS:
		var dnsAction ChangeDNSAction
		err = UnmarshalJSON(taskMessage.Data, &dnsAction)
		if err != nil {
			return "DNS: Error parsing data"
		}

		servers := "none"
		if len(dnsAction.DNSServers) > 0 {
			if len(dnsAction.DNSServers) == 1 {
				servers = dnsAction.DNSServers[0]
			} else {
				servers = fmt.Sprintf("%s (+%d more)", dnsAction.DNSServers[0], len(dnsAction.DNSServers)-1)
			}
		}

		domains := "none"
		if len(dnsAction.SearchDomains) > 0 {
			if len(dnsAction.SearchDomains) == 1 {
				domains = dnsAction.SearchDomains[0]
			} else {
				domains = fmt.Sprintf("%s (+%d more)", dnsAction.SearchDomains[0], len(dnsAction.SearchDomains)-1)
			}
		}

		return fmt.Sprintf("DNS: %s → %s | %s", dnsAction.InterfaceName, servers, domains)

	case ActionTypeShutdown:
		var shutdownAction ShutdownAction
		err = UnmarshalJSON(taskMessage.Data, &shutdownAction)
		if err != nil {
			return "Shutdown: Error parsing data"
		}

		delay := ""
		if shutdownAction.DelaySeconds > 0 {
			delay = fmt.Sprintf(" (delay: %ds)", shutdownAction.DelaySeconds)
		}

		message := ""
		if shutdownAction.Message != "" {
			message = fmt.Sprintf(" - %s", shutdownAction.Message)
		}

		return fmt.Sprintf("Shutdown: %s%s%s", shutdownAction.ActionType, delay, message)

	default:
		return fmt.Sprintf("Unknown: %s", taskMessage.Type)
	}
}
