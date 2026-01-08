package tui

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ImportExportMenuModel struct {
	choices       []string
	cursor        int
	width         int
	height        int
	inputMode     string // "", "export-path", "import-path"
	pathInput     textinput.Model
	exportType    []string
	importType    []string
	statusMessage string
}

func NewImportExportMenu(width, height int) ImportExportMenuModel {
	ti := textinput.New()
	ti.Placeholder = "Enter file path..."
	ti.CharLimit = 256
	ti.Width = 60

	return ImportExportMenuModel{
		choices: []string{
			"Export Data",
			"Import Data",
			"Back to Main Menu",
		},
		cursor:     0,
		width:      width,
		height:     height,
		pathInput:  ti,
		exportType: []string{"both"},
		importType: []string{"both"},
	}
}

func (m ImportExportMenuModel) Init() tea.Cmd {
	return nil
}

func (m ImportExportMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ExportDataMsg:
		// Execute export command
		exportCmd := exec.Command(os.Args[0], "export", "-o", msg.OutputPath)
		if len(msg.ExportType) > 0 && msg.ExportType[0] != "both" {
			for _, t := range msg.ExportType {
				exportCmd.Args = append(exportCmd.Args, "-t", t)
			}
		}

		output, err := exportCmd.CombinedOutput()
		if err != nil {
			m.statusMessage = fmt.Sprintf("❌ Export failed: %v\n%s", err, string(output))
		} else {
			m.statusMessage = fmt.Sprintf("✓ Export successful to: %s", msg.OutputPath)
		}
		m.inputMode = ""
		return m, nil

	case ImportDataMsg:
		// Execute import command
		importCmd := exec.Command(os.Args[0], "import", "-f", msg.FilePath)
		if len(msg.ImportType) > 0 && msg.ImportType[0] != "both" {
			for _, t := range msg.ImportType {
				importCmd.Args = append(importCmd.Args, "-t", t)
			}
		}

		output, err := importCmd.CombinedOutput()
		if err != nil {
			m.statusMessage = fmt.Sprintf("❌ Import failed: %v\n%s", err, string(output))
		} else {
			m.statusMessage = fmt.Sprintf("✓ Import successful from: %s", msg.FilePath)
		}
		m.inputMode = ""
		return m, nil

	case tea.KeyMsg:
		// Handle input mode
		if m.inputMode != "" {
			switch msg.String() {
			case "esc":
				m.inputMode = ""
				m.pathInput.Blur()
				m.pathInput.SetValue("")
				m.statusMessage = ""
				return m, nil

			case "enter":
				path := m.pathInput.Value()
				if path == "" {
					m.statusMessage = "❌ Path cannot be empty"
					return m, nil
				}

				// Return appropriate message based on mode
				if m.inputMode == "export-path" {
					m.inputMode = ""
					m.pathInput.Blur()
					m.pathInput.SetValue("")
					return m, func() tea.Msg {
						return ExportDataMsg{
							OutputPath: path,
							ExportType: m.exportType,
						}
					}
				} else if m.inputMode == "import-path" {
					m.inputMode = ""
					m.pathInput.Blur()
					m.pathInput.SetValue("")
					return m, func() tea.Msg {
						return ImportDataMsg{
							FilePath:   path,
							ImportType: m.importType,
						}
					}
				}
			}

			m.pathInput, cmd = m.pathInput.Update(msg)
			return m, cmd
		}

		// Handle menu navigation
		switch msg.String() {
		case "ctrl+c", "q":
			return m, func() tea.Msg {
				return BackToDashboardMsg{}
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", "l", "right", " ":
			switch m.cursor {
			case 0: // Export Data
				m.inputMode = "export-path"
				m.pathInput.SetValue("")
				m.pathInput.Focus()
				m.statusMessage = "Enter output path for export (e.g., backup.zip)"
				return m, textinput.Blink

			case 1: // Import Data
				m.inputMode = "import-path"
				m.pathInput.SetValue("")
				m.pathInput.Focus()
				m.statusMessage = "Enter path to ZIP file to import"
				return m, textinput.Blink

			case 2: // Back to Main Menu
				return m, func() tea.Msg {
					return BackToDashboardMsg{}
				}
			}

		case "esc":
			return m, func() tea.Msg {
				return BackToDashboardMsg{}
			}
		}
	}

	return m, nil
}

func (m ImportExportMenuModel) View() string {
	var s string

	if m.inputMode != "" {
		// Show input mode
		s = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render("📦 Import/Export") + "\n\n"
		s += m.statusMessage + "\n\n"
		s += m.pathInput.View() + "\n\n"
		s += helpStyle.Render("enter: confirm • esc: cancel")
	} else {
		// Show menu
		s = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render("📦 Import/Export") + "\n\n"

		if m.statusMessage != "" {
			s += m.statusMessage + "\n\n"
		}

		for i, choice := range m.choices {
			cursor := "  "
			if m.cursor == i {
				cursor = "▶ "
				s += selectedItemStyle.Render(cursor+choice) + "\n"
			} else {
				s += menuItemStyle.Render(cursor+choice) + "\n"
			}
		}

		s += "\n" + helpStyle.Render("↑/k: up • ↓/j: down • enter/l: select • esc/q: back")
	}

	// Center the content
	if m.width > 0 && m.height > 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center)
		return style.Render(s)
	}

	return s
}

// Message types
type ExportDataMsg struct {
	OutputPath string
	ExportType []string
}

type ImportDataMsg struct {
	FilePath   string
	ImportType []string
}
