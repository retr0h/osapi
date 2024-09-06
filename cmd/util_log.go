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

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
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

// section represents a header with its corresponding rows.
type section struct {
	Title   string
	Headers []string
	Rows    [][]string
}

// printStyledTable renders a styled table with padding.
func printStyledTable(sections []section, additionalInfo string) {
	re := lipgloss.NewRenderer(os.Stdout)

	var (
		HeaderStyle  = re.NewStyle().Foreground(purple).Bold(true).Align(lipgloss.Center)
		CellStyle    = re.NewStyle().Padding(0, 1).Width(20)
		OddRowStyle  = CellStyle.Foreground(gray)
		EvenRowStyle = CellStyle.Foreground(lightGray)
		BorderStyle  = re.NewStyle().Foreground(purple)
		PaddingStyle = re.NewStyle().Padding(0, 2)
		TitleStyle   = re.NewStyle().Bold(true).Foreground(purple).MarginBottom(1).Padding(0, 2)
	)

	// Render and apply padding to the system information first.
	fmt.Println(PaddingStyle.Render(additionalInfo))

	// Iterate over each section to render titles and tables.
	for _, section := range sections {
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
