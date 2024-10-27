package images

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/dustin/go-humanize"
	"github.com/mintoolkit/mint/pkg/app"

	"github.com/mintoolkit/mint/pkg/app/master/command"
	"github.com/mintoolkit/mint/pkg/app/master/tui/common"
	"github.com/mintoolkit/mint/pkg/app/master/tui/keys"
	"github.com/mintoolkit/mint/pkg/crt"
	"github.com/mintoolkit/mint/pkg/crt/docker/dockerutil"

	tea "github.com/charmbracelet/bubbletea"
)

// TUI represents the internal state of the terminal user interface.
type TUI struct {
	table      table.Table
	width      int
	height     int
	standalone bool
	loading    bool
	// runtime selection controls
	choice int
	chosen bool
}

// Styles - move to `common`
const (
	gray      = lipgloss.Color("#737373")
	lightGray = lipgloss.Color("#d3d3d3")
	white     = lipgloss.Color("#ffffff")
)

var (
	// HeaderStyle is the lipgloss style used for the table headers.
	HeaderStyle = lipgloss.NewStyle().Foreground(white).Bold(true).Align(lipgloss.Center)
	// CellStyle is the base lipgloss style used for the table rows.
	CellStyle = lipgloss.NewStyle().Padding(0, 1).Width(14)
	// OddRowStyle is the lipgloss style used for odd-numbered table rows.
	OddRowStyle = CellStyle.Foreground(gray)
	// EvenRowStyle is the lipgloss style used for even-numbered table rows.
	EvenRowStyle = CellStyle.Foreground(lightGray)
	// BorderStyle is the lipgloss style used for the table border.
	BorderStyle = lipgloss.NewStyle().Foreground(white)
	// CheckboxStyle is the lipgloss style used for the runtime selector
	CheckboxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
)

// End styles

func LoadTUI() *TUI {
	m := &TUI{
		width:   20,
		height:  15,
		loading: true,
	}
	return m
}

func generateTable(images map[string]crt.BasicImageInfo) table.Table {
	var rows [][]string
	for k, v := range images {
		imageRow := []string{k, dockerutil.CleanImageID(v.ID)[:12], humanize.Time(time.Unix(v.Created, 0)), humanize.Bytes(uint64(v.Size))}
		rows = append(rows, imageRow)
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(BorderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			var style lipgloss.Style

			switch {
			case row == 0:
				return HeaderStyle
			case row%2 == 0:
				style = EvenRowStyle
			default:
				style = OddRowStyle
			}

			return style
		}).
		Headers("Name", "ID", "Created", "Size").
		Rows(rows...)

	return *t
}

// InitialTUI returns the initial state of the TUI.
func InitialTUI(images map[string]crt.BasicImageInfo, standalone bool) *TUI {
	m := &TUI{
		width:      20,
		height:     15,
		standalone: standalone,
	}
	m.table = generateTable(images)
	return m
}

func (m TUI) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

// Update is called to handle user input and update the TUI's state.
func (m TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case common.Event:
		imagesCh := make(chan interface{})
		imagesChannelMap := map[string]chan interface{}{
			// TODO - move the name of this channel to a centralized location
			// to be reused by `handler.go`'s `OnCommand` function.
			"images": imagesCh,
		}
		xc := app.NewExecutionContext(
			"tui",
			true,
			"json",
			imagesChannelMap,
		)

		cparams := &CommandParams{
			Runtime:   crt.AutoRuntime,
			GlobalTUI: true,
		}

		gcValue, ok := msg.Data.(*command.GenericParams)
		if !ok || gcValue == nil {
			return nil, nil
		}

		go OnCommand(xc, gcValue, cparams)
		imagesData := <-imagesCh
		images, ok := imagesData.(map[string]crt.BasicImageInfo)
		if !ok || images == nil {
			return nil, nil
		}
		m.table = generateTable(images)
		return m, nil
	case tea.WindowSizeMsg:
		m.table.Width(msg.Width)
		m.table.Height(msg.Height)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Global.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Global.Back):
			return common.TUIsInstance.Home, nil
		}
	}
	// If the user has not made a choice, handle choice updates
	if !m.chosen {
		return updateChoices(msg, m)
	}
	// Otherwise...
	// TODO - loading state after a user has selected a choice
	return m, nil
}

func updateChoices(msg tea.Msg, m TUI) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.choice++
			if m.choice > 3 {
				m.choice = 3
			}
		case "k", "up":
			m.choice--
			if m.choice < 0 {
				m.choice = 0
			}
		case "enter":
			m.chosen = true
			return m, nil
		}
	}

	return m, nil
}

func choicesView(m TUI) string {
	choice := m.choice

	template := "Filter by runtime\n\n"
	template += "%s\n\n"
	choices := fmt.Sprintf(
		"%s\n%s\n%s",
		checkbox("Docker", choice == 0),
		checkbox("Containerd", choice == 1),
		checkbox("Podman", choice == 2),
	)
	return fmt.Sprintf(template, choices)
}

func checkbox(label string, checked bool) string {
	if checked {
		return CheckboxStyle.Render("[x] " + label)
	}
	return fmt.Sprintf("[ ] %s", label)
}

const (
	dockerRuntime     = "docker"
	containerdRuntime = "containerd"
	podmanRuntime     = "podman"
)

func chosenView(m TUI) string {
	var runtime string

	switch m.choice {
	case 0:
		runtime = dockerRuntime
	case 1:
		runtime = containerdRuntime
	case 2:
		runtime = podmanRuntime
	}

	return fmt.Sprintf("You picked - %s:)", runtime)
}

// View returns the view that should be displayed.
func (m TUI) View() string {
	var components []string

	tableContent := m.table.String()

	var runtimeSelectorContent string
	if !m.chosen {
		runtimeSelectorContent = choicesView(m)
	} else {
		runtimeSelectorContent = chosenView(m)
	}

	components = append(components, tableContent, runtimeSelectorContent)

	components = append(components, m.help())

	return lipgloss.JoinVertical(lipgloss.Left,
		components...,
	)
}

func (m TUI) help() string {
	if m.standalone {
		return common.HelpStyle("• j/k, up/down: select • enter: choose • q: quit")
	}
	return common.HelpStyle("• j/k, up/down: select • enter: choose • esc: back • q: quit")
}
