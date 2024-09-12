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
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

var (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
)

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

// section represents a header with its corresponding rows.
type section struct {
	Title   string
	Headers []string
	Rows    [][]string
}

// printStyledTable renders a styled table with padding.
// printStyledTable renders a styled table with dynamic column widths.
func printStyledTable(sections []section, additionalInfo string) {
	re := lipgloss.NewRenderer(os.Stdout)

	// Measure terminal width dynamically
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 80 // Default to 80 if unable to get terminal size
	}

	// Set a maximum width for the table
	maxWidth := 100 // Adjust this to your preferred maximum width

	for _, section := range sections {
		// Calculate the maximum header length for the current section
		maxHeaderLength := getMaxHeaderLength(section.Headers)

		// Calculate the dynamic width per cell, ensuring it does not exceed the max width
		dynamicWidth := (termWidth - 10) / len(section.Headers)
		if dynamicWidth < maxHeaderLength {
			dynamicWidth = maxHeaderLength
		}
		if dynamicWidth > maxWidth {
			dynamicWidth = maxWidth
		}

		var (
			HeaderStyle  = re.NewStyle().Foreground(purple).Bold(true).Align(lipgloss.Center)
			CellStyle    = re.NewStyle().Padding(0, 1).Width(dynamicWidth)
			OddRowStyle  = CellStyle.Foreground(gray)
			EvenRowStyle = CellStyle.Foreground(lightGray)
			BorderStyle  = re.NewStyle().Foreground(purple)
			PaddingStyle = re.NewStyle().Padding(0, 2)
			TitleStyle   = re.NewStyle().Bold(true).Foreground(purple).MarginBottom(1).Padding(0, 2)
		)

		// Render and apply padding to the system information first.
		fmt.Println(PaddingStyle.Render(additionalInfo))

		// Render the section title if it exists.
		if section.Title != "" {
			fmt.Println(TitleStyle.Render(section.Title))
		}

		// Create the table and apply styles.
		t := table.New().
			Border(lipgloss.ThickBorder()).
			BorderStyle(BorderStyle).
			StyleFunc(func(row, _ int) lipgloss.Style {
				switch {
				case row == 0:
					return HeaderStyle
				case row%2 == 0:
					return EvenRowStyle
				default:
					return OddRowStyle
				}
			})

		// Add headers and rows for the current section to the table.
		t.Headers(section.Headers...)
		t.Rows(section.Rows...)

		// Render the styled table.
		fmt.Println(PaddingStyle.Render(t.String()))
	}
}

// formatList helper function to convert []string to a formatted string.
func formatList(list []string) string {
	if len(list) == 0 {
		return "None"
	}
	return strings.Join(list, ", ")
}

// getMaxHeaderLength calculates the maximum length of the given headers.
func getMaxHeaderLength(headers []string) int {
	maxLen := 0
	for _, header := range headers {
		if len(header) > maxLen {
			maxLen = len(header)
		}
	}
	return maxLen
}

// safeInt returns a default value when the input *int is nil.
func safeInt(i *int) int {
	if i != nil {
		return *i
	}
	return 0
}

// safeString function to safely dereference string pointers
func safeString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// safeTime function to safely dereference time.Time pointers
func safeTime(t *time.Time) string {
	if t != nil {
		return t.Format(time.RFC3339)
	}
	return ""
}
