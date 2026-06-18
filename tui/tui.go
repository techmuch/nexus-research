package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/techmuch/nexus-research/db"
)

type state int

const (
	stateMenu state = iota
	stateUserList
	stateUserCreate
	stateUserDelete
	stateConfig
	stateStatus
)

type Model struct {
	state       state
	cursor      int
	dbPath      string
	statusMsg   string
	terminalOut io.Writer
	form        *huh.Form
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00b52d")).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00b52d")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Padding(1, 2)
)

func NewModel(dbPath string) Model {
	return Model{
		state:       stateMenu,
		cursor:      0,
		dbPath:      dbPath,
		terminalOut: os.Stdout,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If a form is currently active, forward the message to the form
	if m.state == stateUserCreate || m.state == stateUserDelete || m.state == stateConfig {
		var cmd tea.Cmd
		var newForm tea.Model
		newForm, cmd = m.form.Update(msg)
		m.form = newForm.(*huh.Form)

		if m.form.State == huh.StateCompleted {
			m.handleFormCompletion()
			m.form = nil
		} else if m.form.State == huh.StateAborted {
			m.handleFormAbortion()
			m.form = nil
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == stateMenu {
				return m, tea.Quit
			}
			m.state = stateMenu
			m.statusMsg = ""
			return m, nil
		case "up", "k":
			if m.state == stateMenu {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			if m.state == stateMenu {
				if m.cursor < 3 {
					m.cursor++
				}
			}
		case "esc":
			m.state = stateMenu
			m.statusMsg = ""
			return m, nil
		case "enter":
			if m.state == stateMenu {
				switch m.cursor {
				case 0: // User Management
					m.state = stateUserList
					m.statusMsg = ""
				case 1: // Edit Database Config
					m.state = stateConfig
					m.form = huh.NewForm(
						huh.NewGroup(
							huh.NewInput().
								Title("SQLite Database Path").
								Key("newPath").
								Value(&m.dbPath).
								Validate(func(str string) error {
									if strings.TrimSpace(str) == "" {
										return fmt.Errorf("database path cannot be empty")
									}
									return nil
								}),
						),
					)
					return m, m.form.Init()
				case 2: // View System Status
					m.state = stateStatus
					m.statusMsg = ""
				case 3: // Exit
					return m, tea.Quit
				}
			} else if m.state == stateStatus {
				m.state = stateMenu
			}
		case "a":
			if m.state == stateUserList {
				m.state = stateUserCreate
				m.form = huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Username").
							Key("username").
							Validate(func(str string) error {
								if strings.TrimSpace(str) == "" {
									return fmt.Errorf("username cannot be empty")
								}
								return nil
							}),
						huh.NewInput().
							Title("Password").
							Password(true).
							Key("password").
							Validate(func(str string) error {
								if len(str) < 4 {
									return fmt.Errorf("password must be at least 4 characters")
								}
								return nil
							}),
					),
				)
				return m, m.form.Init()
			}
		case "d":
			if m.state == stateUserList {
				users, err := db.ListUsers()
				if err != nil || len(users) == 0 {
					m.statusMsg = "No users available to delete"
					return m, nil
				}
				var options []huh.Option[string]
				for _, u := range users {
					options = append(options, huh.NewOption(u.Username, u.Username))
				}
				m.state = stateUserDelete
				m.form = huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[string]().
							Title("Select user to delete").
							Options(options...).
							Key("targetUser"),
					),
				)
				return m, m.form.Init()
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder

	if m.state == stateUserCreate || m.state == stateUserDelete || m.state == stateConfig {
		return titleStyle.Render("NEXUS RESEARCH STATION - TUI PANEL") + "\n\n" + m.form.View()
	}

	s.WriteString(titleStyle.Render("NEXUS RESEARCH STATION - TUI PANEL"))
	s.WriteString("\n")

	switch m.state {
	case stateMenu:
		s.WriteString("Select an option:\n\n")
		options := []string{
			"User Management (List, Add, Delete)",
			"Database Configuration (Edit path)",
			"View System Status",
			"Exit",
		}
		for i, opt := range options {
			if i == m.cursor {
				s.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", opt)))
			} else {
				s.WriteString(normalStyle.Render(fmt.Sprintf("  %s", opt)))
			}
			s.WriteString("\n")
		}
		s.WriteString("\n")
		if m.statusMsg != "" {
			s.WriteString(selectedStyle.Render(m.statusMsg) + "\n\n")
		}
		s.WriteString("[up/down: navigate, enter: select, q: quit]\n")

	case stateUserList:
		s.WriteString("Registered User Accounts:\n\n")
		users, err := db.ListUsers()
		if err != nil {
			s.WriteString(fmt.Sprintf("Error listing users: %v\n", err))
		} else if len(users) == 0 {
			s.WriteString("No users found. Admin must create a user.\n")
		} else {
			for _, u := range users {
				s.WriteString(normalStyle.Render(fmt.Sprintf("- %-20s (Created: %s)\n", u.Username, u.CreatedAt.Format("2006-01-02 15:04"))))
			}
		}
		s.WriteString("\n")
		if m.statusMsg != "" {
			s.WriteString(selectedStyle.Render(m.statusMsg) + "\n\n")
		}
		s.WriteString("[a: add user, d: delete user, esc/q: back]\n")

	case stateStatus:
		s.WriteString("System Metrics:\n\n")
		var userCount int
		users, err := db.ListUsers()
		if err == nil {
			userCount = len(users)
		}
		
		// DB file size
		var dbSizeStr string = "0 bytes"
		fileInfo, err := os.Stat(m.dbPath)
		if err == nil {
			dbSizeStr = fmt.Sprintf("%.2f KB", float64(fileInfo.Size())/1024.0)
		} else if m.dbPath == ":memory:" {
			dbSizeStr = "In-memory database (volatile)"
		}

		statusBlock := fmt.Sprintf(
			"Database Path : %s\nDatabase Size : %s\nTotal Users   : %d\nSystem State  : ONLINE",
			m.dbPath, dbSizeStr, userCount,
		)
		s.WriteString(borderStyle.Render(statusBlock))
		s.WriteString("\n\n")
		s.WriteString("[enter/esc: back]\n")
	}

	return s.String()
}

func (m *Model) handleFormCompletion() {
	switch m.state {
	case stateUserCreate:
		username := m.form.GetString("username")
		password := m.form.GetString("password")
		err := db.CreateUser(username, password)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("User '%s' successfully created", username)
		}
		m.state = stateUserList
	case stateUserDelete:
		targetUser := m.form.GetString("targetUser")
		err := db.DeleteUser(targetUser)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error deleting user: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("User '%s' successfully deleted", targetUser)
		}
		m.state = stateUserList
	case stateConfig:
		newPath := m.form.GetString("newPath")
		err := db.CloseDB()
		if err == nil {
			err = db.InitDB(newPath)
		}
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error changing DB config: %v", err)
			_ = db.InitDB(m.dbPath)
		} else {
			m.dbPath = newPath
			m.statusMsg = fmt.Sprintf("Database path configured to '%s'", newPath)
		}
		m.state = stateMenu
	}
}

func (m *Model) handleFormAbortion() {
	switch m.state {
	case stateUserCreate, stateUserDelete:
		m.state = stateUserList
	case stateConfig:
		m.state = stateMenu
	}
}
