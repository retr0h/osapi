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

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
)

// logFatal logs a fatal error message along with optional structured data
// and then exits the program with a status code of 1.
func logFatal(message string, err error, kvPairs ...any) {
	if err != nil {
		kvPairs = append(kvPairs, "error", err)
	}
	logger.Error(
		message,
		kvPairs...,
	)

	os.Exit(1)
}

// prettyPrintJSON unmarshals JSON from a byte slice, formats it with indentation,
// and prints it to the standard output.
func prettyPrintJSON(respBody []byte) {
	var jsonObj interface{}
	if err := json.Unmarshal(respBody, &jsonObj); err != nil {
		logFatal("failed to unmarshal json", err)
	}

	prettyJSON, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		logFatal("failed to marshal json", err)
	}

	fmt.Println(string(prettyJSON))
}
