package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/techmuch/nexus-research/db"
)

type state int

const (
	stateMenu state = iota
	stateUserList
	stateUserDetail
	stateUserCreate
	stateUserDelete
	stateConfig
	stateStatus
	stateUserChangePassword
	stateUserRename
)

type Model struct {
	state              state
	cursor             int
	dbPath             string
	statusMsg          string
	terminalOut        io.Writer
	form               *huh.Form
	selectedUserCursor int
	detailCursor       int
	targetUser         db.User
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
		state:              stateMenu,
		cursor:             0,
		dbPath:             dbPath,
		terminalOut:        os.Stdout,
		selectedUserCursor: 0,
		detailCursor:       0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If a form is currently active, forward the message to the form
	if m.state == stateUserCreate || m.state == stateUserDelete || m.state == stateConfig || m.state == stateUserChangePassword || m.state == stateUserRename {
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
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.state == stateMenu {
				return m, tea.Quit
			}
			if m.state == stateUserDetail {
				m.state = stateUserList
				m.statusMsg = ""
				return m, nil
			}
			m.state = stateMenu
			m.statusMsg = ""
			return m, nil
		case "up", "k":
			switch m.state {
			case stateMenu:
				if m.cursor > 0 {
					m.cursor--
				}
			case stateUserList:
				if m.selectedUserCursor > 0 {
					m.selectedUserCursor--
				}
			case stateUserDetail:
				if m.detailCursor > 0 {
					m.detailCursor--
				}
			}
		case "down", "j":
			switch m.state {
			case stateMenu:
				if m.cursor < 3 {
					m.cursor++
				}
			case stateUserList:
				users, err := db.ListUsers()
				if err == nil && len(users) > 0 {
					if m.selectedUserCursor < len(users)-1 {
						m.selectedUserCursor++
					}
				}
			case stateUserDetail:
				if m.detailCursor < 4 {
					m.detailCursor++
				}
			}
		case "esc":
			if m.state == stateUserDetail {
				m.state = stateUserList
				m.statusMsg = ""
			} else {
				m.state = stateMenu
				m.statusMsg = ""
			}
			return m, nil
		case "enter":
			switch m.state {
			case stateMenu:
				switch m.cursor {
				case 0: // User Management
					m.state = stateUserList
					m.statusMsg = ""
					m.selectedUserCursor = 0
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
			case stateUserList:
				users, err := db.ListUsers()
				if err == nil && len(users) > 0 && m.selectedUserCursor >= 0 && m.selectedUserCursor < len(users) {
					m.targetUser = users[m.selectedUserCursor]
					m.state = stateUserDetail
					m.detailCursor = 0
					m.statusMsg = ""
				}
			case stateUserDetail:
				switch m.detailCursor {
				case 0: // Change Password
					m.state = stateUserChangePassword
					m.form = huh.NewForm(
						huh.NewGroup(
							huh.NewInput().
								Title("New Password").
								Password(true).
								Key("newPassword").
								Validate(func(str string) error {
									if len(str) < 4 {
										return fmt.Errorf("password must be at least 4 characters")
									}
									return nil
								}),
						),
					)
					return m, m.form.Init()
				case 1: // Rename User
					m.state = stateUserRename
					m.form = huh.NewForm(
						huh.NewGroup(
							huh.NewInput().
								Title("New Username").
								Key("newUsername").
								Validate(func(str string) error {
									if strings.TrimSpace(str) == "" {
										return fmt.Errorf("username cannot be empty")
									}
									return nil
								}),
						),
					)
					return m, m.form.Init()
				case 2: // Toggle Status
					err := db.SetDisabled(m.targetUser.Username, !m.targetUser.IsDisabled)
					if err != nil {
						m.statusMsg = fmt.Sprintf("Error toggling status: %v", err)
					} else {
						m.targetUser.IsDisabled = !m.targetUser.IsDisabled
						if m.targetUser.IsDisabled {
							m.statusMsg = fmt.Sprintf("User '%s' is now disabled", m.targetUser.Username)
						} else {
							m.statusMsg = fmt.Sprintf("User '%s' is now enabled", m.targetUser.Username)
						}
					}
				case 3: // Delete User
					m.state = stateUserDelete
					m.form = huh.NewForm(
						huh.NewGroup(
							huh.NewConfirm().
								Title(fmt.Sprintf("Are you sure you want to delete '%s'?", m.targetUser.Username)).
								Key("confirmDelete"),
						),
					)
					return m, m.form.Init()
				case 4: // Back to List
					m.state = stateUserList
					m.statusMsg = ""
				}
			case stateStatus:
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
						huh.NewConfirm().
							Title("Give admin permissions?").
							Key("isAdmin"),
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
				options = append(options, huh.NewOption("(Cancel)", "CANCEL"))
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

	if m.state == stateUserCreate || m.state == stateUserDelete || m.state == stateConfig || m.state == stateUserChangePassword || m.state == stateUserRename {
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
			t := table.New().
				Border(lipgloss.RoundedBorder()).
				BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#30363D"))).
				Headers("USERNAME", "ROLE", "STATUS", "CREATED AT")

			for _, u := range users {
				role := "User"
				if u.IsAdmin {
					role = "Admin"
				}
				statusText := "Active"
				if u.IsDisabled {
					statusText = "Disabled"
				}
				t.Row(u.Username, role, statusText, u.CreatedAt.Format("2006-01-02 15:04"))
			}

			t.StyleFunc(func(row, col int) lipgloss.Style {
				if row == 0 {
					return lipgloss.NewStyle().Foreground(lipgloss.Color("#00b52d")).Bold(true)
				}
				// Highlight selected user row
				if row-1 == m.selectedUserCursor {
					return lipgloss.NewStyle().
						Foreground(lipgloss.Color("#00b52d")).
						Bold(true).
						Background(lipgloss.Color("#161B22"))
				}
				return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
			})

			s.WriteString(t.Render())
			s.WriteString("\n")
		}
		s.WriteString("\n")
		if m.statusMsg != "" {
			s.WriteString(selectedStyle.Render(m.statusMsg) + "\n\n")
		}
		s.WriteString("[up/down: navigate, enter: select, a: add user, d: delete, esc/q: back]\n")

	case stateUserDetail:
		s.WriteString(fmt.Sprintf("User Profile: %s\n\n", m.targetUser.Username))

		statusText := "Active"
		if m.targetUser.IsDisabled {
			statusText = "Disabled"
		}
		roleText := "User"
		if m.targetUser.IsAdmin {
			roleText = "Admin"
		}

		profileBlock := fmt.Sprintf(
			"Username   : %s\nRole       : %s\nStatus     : %s\nRegistered : %s",
			m.targetUser.Username, roleText, statusText, m.targetUser.CreatedAt.Format("2006-01-02 15:04"),
		)
		s.WriteString(borderStyle.Render(profileBlock))
		s.WriteString("\n\nSelect action:\n\n")

		toggleText := "Disable User Account"
		if m.targetUser.IsDisabled {
			toggleText = "Enable User Account"
		}

		actions := []string{
			"Change Password",
			"Rename User",
			toggleText,
			"Delete User Account",
			"Back to User List",
		}

		for i, act := range actions {
			if i == m.detailCursor {
				s.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", act)))
			} else {
				s.WriteString(normalStyle.Render(fmt.Sprintf("  %s", act)))
			}
			s.WriteString("\n")
		}
		s.WriteString("\n")
		if m.statusMsg != "" {
			s.WriteString(selectedStyle.Render(m.statusMsg) + "\n\n")
		}
		s.WriteString("[up/down: navigate, enter: select, esc/q: back]\n")

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
		isAdmin := m.form.GetBool("isAdmin")
		err := db.CreateUser(username, password, isAdmin)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("User '%s' successfully created", username)
		}
		m.state = stateUserList
	case stateUserChangePassword:
		newPassword := m.form.GetString("newPassword")
		err := db.ChangePassword(m.targetUser.Username, newPassword)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error changing password: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("Password for '%s' updated successfully", m.targetUser.Username)
		}
		m.state = stateUserDetail
	case stateUserRename:
		newUsername := m.form.GetString("newUsername")
		err := db.RenameUser(m.targetUser.Username, newUsername)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error renaming user: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("User renamed from '%s' to '%s'", m.targetUser.Username, newUsername)
			m.targetUser.Username = newUsername
		}
		m.state = stateUserDetail
	case stateUserDelete:
		confirmDelete := m.form.Get("confirmDelete")
		if confirmDelete != nil {
			if m.form.GetBool("confirmDelete") {
				err := db.DeleteUser(m.targetUser.Username)
				if err != nil {
					m.statusMsg = fmt.Sprintf("Error deleting user: %v", err)
					m.state = stateUserDetail
				} else {
					m.statusMsg = fmt.Sprintf("User '%s' successfully deleted", m.targetUser.Username)
					m.state = stateUserList
					m.selectedUserCursor = 0
				}
			} else {
				m.statusMsg = "User deletion cancelled"
				m.state = stateUserDetail
			}
		} else {
			targetUser := m.form.GetString("targetUser")
			if targetUser == "CANCEL" || targetUser == "" {
				m.statusMsg = "User deletion cancelled"
				m.state = stateUserList
				return
			}
			err := db.DeleteUser(targetUser)
			if err != nil {
				m.statusMsg = fmt.Sprintf("Error deleting user: %v", err)
			} else {
				m.statusMsg = fmt.Sprintf("User '%s' successfully deleted", targetUser)
			}
			m.state = stateUserList
		}
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
		if m.targetUser.Username != "" {
			m.state = stateUserDetail
		} else {
			m.state = stateUserList
		}
	case stateUserChangePassword, stateUserRename:
		m.state = stateUserDetail
	case stateConfig:
		m.state = stateMenu
	}
}
