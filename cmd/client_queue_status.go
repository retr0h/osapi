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
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type model struct {
	queueStatus string
	lastUpdate  time.Time
	isLoading   bool
}

func initialModel() model {
	return model{
		queueStatus: "Fetching queue status...",
		lastUpdate:  time.Now(),
		isLoading:   true,
	}
}

// Poll every 30 seconds
func tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return t
	})
}

func fetchQueueCmd() tea.Cmd {
	return func() tea.Msg {
		return fetchQueueStatus()
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchQueueCmd(), tickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	case string:
		m.queueStatus = msg
		m.lastUpdate = time.Now()
		m.isLoading = false
		return m, tickCmd()
	case time.Time:
		// timer ticks, fetch new queue status
		return m, fetchQueueCmd()
	}
	return m, nil
}

func (m model) View() string {
	var (
		titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#15"))
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
		timeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")).Italic(true)
		borderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Padding(1).
				Margin(2).
				BorderForeground(purple)
	)

	title := titleStyle.Render("Queue Status")
	status := statusStyle.Render(m.queueStatus)
	lastUpdated := timeStyle.Render(
		fmt.Sprintf("Last Updated: %v", m.lastUpdate.Format(time.RFC1123)),
	)
	quitInstruction := timeStyle.Render("Press 'q' to quit")

	return borderStyle.Render(
		fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", title, status, lastUpdated, quitInstruction),
	)
}

func fetchQueueStatus() string {
	resp, err := handler.GetQueueStatus(context.TODO())
	if err != nil {
		logFatal("failed to get queue status endpoint", err)
	}

	if resp.JSON200 == nil {
		logFatal("failed response", fmt.Errorf("staus queue response was nil"))
	}

	if resp.JSON200.TotalItems == nil {
		logFatal("failed response", fmt.Errorf("total items in response was nil"))
	}

	itemCount := *resp.JSON200.TotalItems
	return fmt.Sprintf("Queue is currently processing %d tasks.", itemCount)
}

// clientQueueStatusCmd represents the clientQueueStatus command.
var clientQueueStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display the queue's status",
	Long: `Displays the queue's status with automatic updates every 30 secods.
`,
	Run: func(_ *cobra.Command, _ []string) {
		p := tea.NewProgram(initialModel())
		_, err := p.Run()
		if err != nil {
			logFatal("failed running the program: %v", err)
		}
	},
}

func init() {
	clientQueueCmd.AddCommand(clientQueueStatusCmd)
}
