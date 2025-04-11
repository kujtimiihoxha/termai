package chat

import tea "github.com/charmbracelet/bubbletea"

type sidebarCmp struct{}

func (m *sidebarCmp) Init() tea.Cmd {
	return nil
}

func (m *sidebarCmp) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *sidebarCmp) View() string {
	return "Sidebar"
}

func NewSidebarCmp() tea.Model {
	return &sidebarCmp{}
}
