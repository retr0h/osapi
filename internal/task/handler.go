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
	"fmt"

	taskpb "github.com/retr0h/osapi/internal/task/gen/proto/task"
)

const (
	// UnknownTaskType is a fallback for unrecognized task types.
	UnknownTaskType Type = iota
	// ShutdownTaskType represents a Shutdown task.
	ShutdownTaskType
	// ChangeDNSTaskType represents a Change DNS task.
	ChangeDNSTaskType
)

// String provides a string representation of the task type.
func (t Type) String() string {
	switch t {
	case ShutdownTaskType:
		return "Shutdown"
	case ChangeDNSTaskType:
		return "ChangeDNS"
	default:
		return "Unknown"
	}
}

// GetTaskType abstracts the logic for determining the task type.
func GetTaskType(t *taskpb.Task) (Type, error) {
	switch t.GetAction().(type) {
	case *taskpb.Task_ShutdownAction:
		return ShutdownTaskType, nil
	case *taskpb.Task_ChangeDnsAction:
		return ChangeDNSTaskType, nil
	default:
		return UnknownTaskType, fmt.Errorf("unknown task action type")
	}
}
