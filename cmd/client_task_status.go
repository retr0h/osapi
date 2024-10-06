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

	"github.com/retr0h/osapi/internal/client"
)

// TODO(retr0h): move to cmd.Flags.GetInt() once simplified
var pollIntervalSeconds int

type model struct {
	taskStatus string
	lastUpdate time.Time
	isLoading  bool
}

func initialModel() model {
	return model{
		taskStatus: "Fetching task status...",
		lastUpdate: time.Now(),
		isLoading:  true,
	}
}

// Poll every 30 seconds
func tickCmd() tea.Cmd {
	pollInterval := time.Duration(pollIntervalSeconds) * time.Second

	return tea.Tick(pollInterval, func(t time.Time) tea.Msg {
		return t
	})
}

func fetchTaskCmd() tea.Cmd {
	return func() tea.Msg {
		return fetchTaskStatus()
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchTaskCmd(), tickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	case string:
		m.taskStatus = msg
		m.lastUpdate = time.Now()
		m.isLoading = false
		return m, tickCmd()
	case time.Time:
		// timer ticks, fetch new task status
		return m, fetchTaskCmd()
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

	title := titleStyle.Render("Task Status")
	status := statusStyle.Render(m.taskStatus)
	lastUpdated := timeStyle.Render(
		fmt.Sprintf("Last Updated: %v", m.lastUpdate.Format(time.RFC1123)),
	)
	quitInstruction := timeStyle.Render("Press 'q' to quit")

	return borderStyle.Render(
		fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", title, status, lastUpdated, quitInstruction),
	)
}

func fetchTaskStatus() string {
	taskHandler := handler.(client.TaskHandler)
	resp, err := taskHandler.GetTaskStatus(context.TODO())
	if err != nil {
		logFatal("failed to get task status endpoint", err)
	}

	if resp.JSON200 == nil {
		logFatal("failed response", fmt.Errorf("staus task response was nil"))
	}

	if resp.JSON200.TotalItems == nil {
		logFatal("failed response", fmt.Errorf("total messages in response was nil"))
	}

	messageCount := *resp.JSON200.TotalItems
	return fmt.Sprintf("Currently processing %d tasks.", messageCount)
}

// clientTaskStatusCmd represents the clientTaskStatus command.
var clientTaskStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display the task status",
	Long: `Displays the task status with automatic updates every 30 secods.
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
	clientTaskCmd.AddCommand(clientTaskStatusCmd)

	clientTaskStatusCmd.PersistentFlags().
		IntVarP(&pollIntervalSeconds, "poll-interval-seconds", "", 60, "The interval (in seconds) between polling operations")
}
