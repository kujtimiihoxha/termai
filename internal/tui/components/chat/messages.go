package chat

import tea "github.com/charmbracelet/bubbletea"

type messagesCmp struct{}

func (m *messagesCmp) Init() tea.Cmd {
	return nil
}

func (m *messagesCmp) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *messagesCmp) View() string {
	return "Messages"
}

func NewMessagesCmp() tea.Model {
	return &messagesCmp{}
}
