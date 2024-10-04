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

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	taskpb "github.com/retr0h/osapi/internal/task/gen/proto/task"
)

// MarshalProto is a generic function to marshal any proto.Message.
func MarshalProto(message proto.Message) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}
	return data, nil
}

// UnmarshalProto is a generic function to unmarshal any proto.Message.
func UnmarshalProto(data []byte, message proto.Message) error {
	err := proto.Unmarshal(data, message)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return nil
}

// MarshalTextString is a helper function that formats a proto.Message into a multi-line string.
func MarshalTextString(message proto.Message) (string, error) {
	marshaler := prototext.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}

	protoBytes, err := marshaler.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal proto message to text: %w", err)
	}

	return string(protoBytes), nil
}

// SafeMarshalTaskToString safely converts the provided proto data into a
// string representation of taskpb.Task.
func SafeMarshalTaskToString(body *[]byte) string {
	var taskMessage taskpb.Task

	if body == nil {
		return "N/A"
	}

	err := UnmarshalProto(*body, &taskMessage)
	if err != nil {
		return err.Error()
	}

	protoString, err := MarshalTextString(&taskMessage)
	if err != nil {
		return err.Error()
	}

	return protoString
}
