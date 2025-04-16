package page

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kujtimiihoxha/opencode/internal/tui/components/logs"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
)

var LogsPage PageID = "logs"

type logsPage struct {
	table   logs.TableComponent
	details logs.DetailComponent
}

func (p *logsPage) Init() tea.Cmd {
	return nil
}

func (p *logsPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return p, nil
}

func (p *logsPage) View() string {
	return p.table.View() + "\n" + p.details.View()
}

func NewLogsPage() tea.Model {
	return layout.NewBentoLayout(
		layout.BentoPanes{
			layout.BentoRightTopPane:    logs.NewLogsTable(),
			layout.BentoRightBottomPane: logs.NewLogsDetails(),
		},
		layout.WithBentoLayoutCurrentPane(layout.BentoRightTopPane),
		layout.WithBentoLayoutRightTopHeightRatio(0.5),
	)
}
