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
	targetUser         db.User
	searchQuery        string
	searching          bool
	width              int
	height             int
	scrollOffset       int
	lastHighlightedUsername string
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

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f5f")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Padding(1, 2)

	masterPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Padding(1, 2).
			Width(42)

	detailPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Padding(1, 2).
			Width(38)

	statusCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Padding(1, 2).
			Width(36)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	menuSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00b52d")).
				Background(lipgloss.Color("#21262D")).
				Bold(true).
				Padding(0, 1)

	menuNormalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#161B22")).
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 1)

	badgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Padding(0, 1)
)

func NewModel(dbPath string) Model {
	return Model{
		state:              stateMenu,
		cursor:             0,
		dbPath:             dbPath,
		terminalOut:        os.Stdout,
		selectedUserCursor: 0,
		searchQuery:        "",
		searching:          false,
		scrollOffset:       0,
		lastHighlightedUsername: "",
	}
}

func (m Model) getFilteredUsers() ([]db.User, error) {
	users, err := db.ListUsers()
	if err != nil {
		return nil, err
	}
	if m.searchQuery == "" {
		return users, nil
	}
	var filtered []db.User
	query := strings.ToLower(m.searchQuery)
	for _, u := range users {
		if strings.Contains(strings.ToLower(u.Username), query) {
			filtered = append(filtered, u)
		}
	}
	return filtered, nil
}

func (m *Model) clampCursor(numUsers int) {
	pageSize := m.height - 12
	if pageSize < 5 {
		pageSize = 5
	}
	if m.height <= 0 {
		pageSize = 8
	}

	if numUsers == 0 {
		m.selectedUserCursor = 0
		m.scrollOffset = 0
		return
	}
	if m.selectedUserCursor >= numUsers {
		m.selectedUserCursor = numUsers - 1
	}
	if m.selectedUserCursor < 0 {
		m.selectedUserCursor = 0
	}

	if numUsers <= pageSize {
		m.scrollOffset = 0
		return
	}
	if m.selectedUserCursor < m.scrollOffset {
		m.scrollOffset = m.selectedUserCursor
	} else if m.selectedUserCursor >= m.scrollOffset+pageSize {
		m.scrollOffset = m.selectedUserCursor - pageSize + 1
	}
}

func (m *Model) updateLastHighlighted() {
	users, err := m.getFilteredUsers()
	if err == nil && len(users) > 0 && m.selectedUserCursor >= 0 && m.selectedUserCursor < len(users) {
		m.lastHighlightedUsername = users[m.selectedUserCursor].Username
	}
}

func (m *Model) updateSearchQuery(newQuery string) {
	m.searchQuery = newQuery

	newUsers, err := m.getFilteredUsers()
	if err == nil {
		found := false
		if m.lastHighlightedUsername != "" {
			for idx, u := range newUsers {
				if u.Username == m.lastHighlightedUsername {
					m.selectedUserCursor = idx
					found = true
					break
				}
			}
		}
		if !found {
			m.selectedUserCursor = 0
		}
		m.clampCursor(len(newUsers))
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Capture terminal size messages to keep dimensions updated
	if wMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wMsg.Width
		m.height = wMsg.Height
	}

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
		m.statusMsg = "" // Clear status message on any keypress

		// If in user list and searching, capture all inputs
		if m.state == stateUserList && m.searching {
			switch msg.Type {
			case tea.KeyEnter:
				m.searching = false
			case tea.KeyEsc:
				m.searching = false
				m.updateSearchQuery("")
			case tea.KeyBackspace:
				if len(m.searchQuery) > 0 {
					m.updateSearchQuery(m.searchQuery[:len(m.searchQuery)-1])
				}
			case tea.KeySpace:
				m.updateSearchQuery(m.searchQuery + " ")
			case tea.KeyRunes:
				m.updateSearchQuery(m.searchQuery + string(msg.Runes))
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.state == stateMenu {
				return m, tea.Quit
			}
			m.state = stateMenu
			m.searching = false
			m.searchQuery = ""
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
					m.updateLastHighlighted()
				}
			}
		case "down", "j":
			switch m.state {
			case stateMenu:
				if m.cursor < 3 {
					m.cursor++
				}
			case stateUserList:
				users, err := m.getFilteredUsers()
				if err == nil && len(users) > 0 {
					if m.selectedUserCursor < len(users)-1 {
						m.selectedUserCursor++
						m.updateLastHighlighted()
					}
				}
			}
		case "esc":
			if m.state == stateUserList && m.searchQuery != "" {
				m.updateSearchQuery("")
				return m, nil
			}
			m.state = stateMenu
			m.searching = false
			m.searchQuery = ""
			return m, nil
		case "enter":
			switch m.state {
			case stateMenu:
				switch m.cursor {
				case 0: // User Management
					m.state = stateUserList
					m.selectedUserCursor = 0
					m.searchQuery = ""
					m.searching = false
					m.updateLastHighlighted()
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
				case 3: // Exit
					return m, tea.Quit
				}
			case stateStatus:
				m.state = stateMenu
			}
		case "/":
			if m.state == stateUserList {
				m.searching = true
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
								trimmed := strings.TrimSpace(str)
								if trimmed == "" {
									return fmt.Errorf("username cannot be empty")
								}
								exists := false
								if db.DB != nil {
									_ = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", trimmed).Scan(&exists)
								}
								if exists {
									return fmt.Errorf("user already exists")
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
		case "p":
			if m.state == stateUserList {
				users, err := m.getFilteredUsers()
				if err != nil || len(users) == 0 {
					m.statusMsg = "No user highlighted"
					return m, nil
				}
				m.targetUser = users[m.selectedUserCursor]
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
			}
		case "r":
			if m.state == stateUserList {
				users, err := m.getFilteredUsers()
				if err != nil || len(users) == 0 {
					m.statusMsg = "No user highlighted"
					return m, nil
				}
				m.targetUser = users[m.selectedUserCursor]
				m.state = stateUserRename
				m.form = huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("New Username").
							Key("newUsername").
							Validate(func(str string) error {
								trimmed := strings.TrimSpace(str)
								if trimmed == "" {
									return fmt.Errorf("username cannot be empty")
								}
								if trimmed == m.targetUser.Username {
									return nil
								}
								exists := false
								if db.DB != nil {
									_ = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", trimmed).Scan(&exists)
								}
								if exists {
									return fmt.Errorf("user already exists")
								}
								return nil
							}),
					),
				)
				return m, m.form.Init()
			}
		case "t":
			if m.state == stateUserList {
				users, err := m.getFilteredUsers()
				if err != nil || len(users) == 0 {
					m.statusMsg = "No user highlighted"
					return m, nil
				}
				m.targetUser = users[m.selectedUserCursor]
				err = db.SetDisabled(m.targetUser.Username, !m.targetUser.IsDisabled)
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
			}
		case "d", "x":
			if m.state == stateUserList {
				users, err := m.getFilteredUsers()
				if err != nil || len(users) == 0 {
					m.statusMsg = "No user highlighted"
					return m, nil
				}
				m.targetUser = users[m.selectedUserCursor]
				m.state = stateUserDelete
				m.form = huh.NewForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title(fmt.Sprintf("Are you sure you want to delete '%s'?", m.targetUser.Username)).
							Key("confirmDelete"),
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
		var formWidth int = 60
		if m.width > 0 {
			formWidth = m.width - 6
			if formWidth < 30 {
				formWidth = 30
			}
			if formWidth > 60 {
				formWidth = 60
			}
		}
		title := titleStyle.Render("NEXUS RESEARCH STATION")
		badge := badgeStyle.Render("TUI v1.1")
		header := lipgloss.JoinHorizontal(lipgloss.Top, title, " ", badge)
		formContent := header + "\n\n" + borderStyle.Copy().Width(formWidth).Render(m.form.View()) + "\n\n"
		
		footerText := " ➜ Next Field: Tab • Submit: Enter • Cancel: Esc "
		totalW := m.width
		if totalW <= 0 {
			totalW = 76
		}
		if len(footerText) < totalW {
			footerText = footerText + strings.Repeat(" ", totalW-len(footerText))
		} else if len(footerText) > totalW {
			footerText = footerText[:totalW]
		}
		return formContent + footerStyle.Render(footerText)
	}

	title := titleStyle.Render("NEXUS RESEARCH STATION")
	badge := badgeStyle.Render("TUI v1.1")
	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, title, " ", badge))
	s.WriteString("\n\n")

	var footerText string

	switch m.state {
	case stateMenu:
		var menuSb strings.Builder
		menuSb.WriteString("Select an option:\n\n")
		options := []string{
			"User Management (List, Add, Delete)",
			"Database Configuration (Edit path)",
			"View System Status",
			"Exit",
		}
		
		var menuWidth int = 42
		if m.width > 0 {
			menuWidth = m.width - 6
			if menuWidth < 25 {
				menuWidth = 25
			}
			if menuWidth > 60 {
				menuWidth = 60
			}
		}
		
		innerWidth := menuWidth - 6
		if innerWidth < 20 {
			innerWidth = 20
		}
		
		for i, opt := range options {
			displayStr := fmt.Sprintf("  %s", opt)
			if i == m.cursor {
				displayStr = fmt.Sprintf("➜ %s", opt)
			}
			if len(displayStr) < innerWidth {
				displayStr = displayStr + strings.Repeat(" ", innerWidth-len(displayStr))
			} else if len(displayStr) > innerWidth {
				displayStr = displayStr[:innerWidth]
			}
			
			if i == m.cursor {
				menuSb.WriteString(menuSelectedStyle.Render(displayStr))
			} else {
				menuSb.WriteString(menuNormalStyle.Render(displayStr))
			}
			menuSb.WriteString("\n")
		}

		s.WriteString(borderStyle.Copy().Width(menuWidth).Render(menuSb.String()))
		s.WriteString("\n\n")

		if m.statusMsg != "" {
			style := selectedStyle
			if strings.HasPrefix(strings.ToLower(m.statusMsg), "error") {
				style = errorStyle
			}
			s.WriteString(style.Render(m.statusMsg) + "\n\n")
		}
		
		footerText = " ➜ Navigate: ↑/↓ • Select: Enter • Quit: q "

	case stateUserList:
		users, err := m.getFilteredUsers()
		allUsers, _ := db.ListUsers()
		totalCount := 0
		if err == nil {
			totalCount = len(allUsers)
		}

		if err != nil {
			s.WriteString("User Account Directory (error listing)\n\n")
		} else if len(users) == 0 {
			s.WriteString("User Account Directory (0 accounts)\n\n")
		} else {
			pageSize := m.height - 12
			if pageSize < 5 {
				pageSize = 5
			}
			if m.height <= 0 {
				pageSize = 8
			}
			startIndex := m.scrollOffset
			endIndex := startIndex + pageSize
			if endIndex > len(users) {
				endIndex = len(users)
			}

			if m.searchQuery != "" {
				s.WriteString(fmt.Sprintf("User Account Directory (Showing %d-%d of %d matches, %d total)\n\n", startIndex+1, endIndex, len(users), totalCount))
			} else {
				s.WriteString(fmt.Sprintf("User Account Directory (Showing %d-%d of %d accounts)\n\n", startIndex+1, endIndex, len(users)))
			}
		}

		var leftContent string
		var rightContent string

		// Calculate responsive widths
		var masterWidth int
		var detailWidth int
		if m.width > 0 {
			masterWidth = int(float64(m.width) * 0.55) - 6
			detailWidth = int(float64(m.width) * 0.40) - 6
			if masterWidth < 25 {
				masterWidth = 25
			}
			if detailWidth < 20 {
				detailWidth = 20
			}
		} else {
			masterWidth = 36
			detailWidth = 32
		}

		tableInnerWidth := masterWidth - 6
		if tableInnerWidth < 25 {
			tableInnerWidth = 25
		}

		if err != nil {
			leftContent = fmt.Sprintf("Error listing users: %v\n", err)
			rightContent = "No user profile available"
			footerText = " ➜ Back: Esc "
		} else {
			// Construct Master Pane
			var leftSb strings.Builder
			if len(users) == 0 {
				if m.searchQuery != "" {
					leftSb.WriteString("No users match filter query.\n")
				} else {
					leftSb.WriteString("No registered user accounts.\n")
				}
			} else {
				roleWidth := 8
				statusWidth := 10
				userWidth := tableInnerWidth - roleWidth - statusWidth - 4
				if userWidth < 12 {
					userWidth = 12
				}

				pageSize := m.height - 12
				if pageSize < 5 {
					pageSize = 5
				}
				if m.height <= 0 {
					pageSize = 8
				}
				startIndex := m.scrollOffset
				endIndex := startIndex + pageSize
				if endIndex > len(users) {
					endIndex = len(users)
				}

				t := table.New().
					Border(lipgloss.RoundedBorder()).
					BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#30363D"))).
					Headers("USERNAME", "ROLE", "STATUS")

				for i := startIndex; i < endIndex; i++ {
					u := users[i]
					role := "User"
					if u.IsAdmin {
						role = "Admin"
					}
					statusText := "Active"
					if u.IsDisabled {
						statusText = "Disabled"
					}
					t.Row(u.Username, role, statusText)
				}

				t.StyleFunc(func(row, col int) lipgloss.Style {
					var w int
					switch col {
					case 0:
						w = userWidth
					case 1:
						w = roleWidth
					case 2:
						w = statusWidth
					}
					style := lipgloss.NewStyle().Width(w)

					if row == 0 {
						return style.Foreground(lipgloss.Color("#00b52d")).Bold(true)
					}
					// Highlight selected user row
					if row-1 == m.selectedUserCursor-startIndex {
						return style.
							Foreground(lipgloss.Color("#00b52d")).
							Bold(true).
							Background(lipgloss.Color("#21262D"))
					}
					return style.Foreground(lipgloss.Color("#ffffff"))
				})
				leftSb.WriteString(t.Render())
				leftSb.WriteString("\n")
			}

			// Add search bar details at bottom of Left Pane
			leftSb.WriteString("\n")
			if m.searching {
				searchBoxStyle := lipgloss.NewStyle().
					Border(lipgloss.NormalBorder()).
					BorderForeground(lipgloss.Color("#00b52d")).
					Padding(0, 1).
					Width(tableInnerWidth)
				leftSb.WriteString(searchBoxStyle.Render(fmt.Sprintf("🔍 %s█", m.searchQuery)))
			} else if m.searchQuery != "" {
				searchBoxStyle := lipgloss.NewStyle().
					Border(lipgloss.NormalBorder()).
					BorderForeground(lipgloss.Color("#30363D")).
					Padding(0, 1).
					Width(tableInnerWidth)
				leftSb.WriteString(searchBoxStyle.Render(fmt.Sprintf("🔍 Filter: %s", m.searchQuery)))
			} else {
				leftSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Press [/] to filter users"))
			}

			leftContent = leftSb.String()

			// Construct Detail Pane
			var rightSb strings.Builder
			detailInnerWidth := detailWidth - 6
			if detailInnerWidth < 20 {
				detailInnerWidth = 20
			}

			if len(users) == 0 || m.selectedUserCursor < 0 || m.selectedUserCursor >= len(users) {
				rightSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00b52d")).Bold(true).Render("👤 USER PROFILE"))
				rightSb.WriteString("\n")
				rightSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#30363D")).Render(strings.Repeat("─", detailInnerWidth)))
				rightSb.WriteString("\n\nNo user selected.\n")
			} else {
				u := users[m.selectedUserCursor]
				
				roleText := "👤 User"
				if u.IsAdmin {
					roleText = "🛡️ Admin"
				}
				
				statusLabel := "🟢 Active"
				if u.IsDisabled {
					statusLabel = "🔴 Disabled"
				}

				rightSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00b52d")).Bold(true).Render(fmt.Sprintf("👤 PROFILE: %s", u.Username)))
				rightSb.WriteString("\n")
				rightSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#30363D")).Render(strings.Repeat("─", detailInnerWidth)))
				rightSb.WriteString("\n\n")
				
				rightSb.WriteString(fmt.Sprintf("Username :  %s\n", u.Username))
				rightSb.WriteString(fmt.Sprintf("Role     :  %s\n", roleText))
				rightSb.WriteString(fmt.Sprintf("Status   :  %s\n", statusLabel))
				rightSb.WriteString(fmt.Sprintf("Joined   :  📅 %s\n", u.CreatedAt.Format("2006-01-02")))
			}

			rightContent = rightSb.String()

			if len(users) == 0 {
				footerText = " ➜ Add: a • Back: Esc "
			} else if m.searching {
				footerText = " ➜ Apply Filter: Enter • Cancel: Esc "
			} else {
				footerText = " ➜ Add: a • Password: p • Rename: r • Toggle Status: t • Delete: d • Filter: / • Back: Esc "
			}
		}

		var panes string
		if m.width > 0 && m.width < 75 {
			fullPaneWidth := m.width - 6
			if fullPaneWidth < 25 {
				fullPaneWidth = 25
			}
			panes = lipgloss.JoinVertical(
				lipgloss.Left,
				masterPaneStyle.Copy().Width(fullPaneWidth).Render(leftContent),
				detailPaneStyle.Copy().Width(fullPaneWidth).Render(rightContent),
			)
		} else {
			panes = lipgloss.JoinHorizontal(
				lipgloss.Top,
				masterPaneStyle.Copy().Width(masterWidth).Render(leftContent),
				"  ",
				detailPaneStyle.Copy().Width(detailWidth).Render(rightContent),
			)
		}
		s.WriteString(panes)
		s.WriteString("\n\n")
		if m.statusMsg != "" {
			style := selectedStyle
			if strings.HasPrefix(strings.ToLower(m.statusMsg), "error") {
				style = errorStyle
			}
			s.WriteString(style.Render(m.statusMsg) + "\n\n")
		}

	case stateStatus:
		s.WriteString("System Metrics:\n\n")
		var userCount int
		var adminCount int
		var activeCount int
		var disabledCount int
		users, err := db.ListUsers()
		if err == nil {
			userCount = len(users)
			for _, u := range users {
				if u.IsAdmin {
					adminCount++
				}
				if u.IsDisabled {
					disabledCount++
				} else {
					activeCount++
				}
			}
		}

		// DB file size
		var dbSizeStr string = "0 bytes"
		fileInfo, err := os.Stat(m.dbPath)
		if err == nil {
			dbSizeStr = fmt.Sprintf("%.2f KB", float64(fileInfo.Size())/1024.0)
		} else if m.dbPath == ":memory:" {
			dbSizeStr = "In-Memory (Volatile)"
		}

		dbCardContent := fmt.Sprintf(
			"🗄️ DATABASE ENGINE\n\nPath: %s\nSize: %s\nType: SQLite 3",
			m.dbPath, dbSizeStr,
		)
		userCardContent := fmt.Sprintf(
			"👥 USER METRICS\n\nTotal : %d users\nAdmin : 🛡️ %d\nActive: 🟢 %d / 🔴 %d",
			userCount, adminCount, activeCount, disabledCount,
		)
		sysCardContent := fmt.Sprintf(
			"💚 SYSTEM HEALTH\n\nStatus: %s\nPlatform: %s\nEngine: Bubble Tea",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#00b52d")).Bold(true).Render("ONLINE"),
			"Go CLI",
		)

		var statusLayout string
		if m.width > 0 && m.width < 80 {
			fullCardWidth := m.width - 6
			if fullCardWidth < 25 {
				fullCardWidth = 25
			}
			statusLayout = lipgloss.JoinVertical(
				lipgloss.Left,
				statusCardStyle.Copy().Width(fullCardWidth).Render(dbCardContent),
				statusCardStyle.Copy().Width(fullCardWidth).Render(userCardContent),
				statusCardStyle.Copy().Width(fullCardWidth).Render(sysCardContent),
			)
		} else {
			cardWidth := 36
			if m.width > 0 {
				cardWidth = (m.width - 10) / 3
				if cardWidth < 22 {
					cardWidth = 22
				}
				if cardWidth > 36 {
					cardWidth = 36
				}
			}
			statusLayout = lipgloss.JoinHorizontal(
				lipgloss.Top,
				statusCardStyle.Copy().Width(cardWidth).Render(dbCardContent),
				"  ",
				statusCardStyle.Copy().Width(cardWidth).Render(userCardContent),
				"  ",
				statusCardStyle.Copy().Width(cardWidth).Render(sysCardContent),
			)
		}
		s.WriteString(statusLayout)
		s.WriteString("\n\n")
		
		footerText = " ➜ Back: Esc/Enter "
	}

	if footerText != "" {
		displayFooter := footerText
		totalW := m.width
		if totalW <= 0 {
			totalW = 76
		}
		if len(displayFooter) < totalW {
			displayFooter = displayFooter + strings.Repeat(" ", totalW-len(displayFooter))
		} else if len(displayFooter) > totalW {
			displayFooter = displayFooter[:totalW]
		}
		s.WriteString(footerStyle.Render(displayFooter))
	}

	return s.String()
}

func (m *Model) handleFormCompletion() {
	switch m.state {
	case stateUserCreate:
		username := strings.TrimSpace(m.form.GetString("username"))
		password := m.form.GetString("password")
		isAdmin := m.form.GetBool("isAdmin")
		err := db.CreateUser(username, password, isAdmin)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("User '%s' successfully created", username)
			if users, err := m.getFilteredUsers(); err == nil {
				for idx, u := range users {
					if u.Username == username {
						m.selectedUserCursor = idx
						m.updateLastHighlighted()
						break
					}
				}
			}
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
		m.state = stateUserList
	case stateUserRename:
		newUsername := strings.TrimSpace(m.form.GetString("newUsername"))
		if newUsername == m.targetUser.Username {
			m.statusMsg = "Username unchanged"
			m.state = stateUserList
			break
		}
		err := db.RenameUser(m.targetUser.Username, newUsername)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error renaming user: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("User renamed from '%s' to '%s'", m.targetUser.Username, newUsername)
			if users, err := m.getFilteredUsers(); err == nil {
				for idx, u := range users {
					if u.Username == newUsername {
						m.selectedUserCursor = idx
						m.updateLastHighlighted()
						break
					}
				}
			}
		}
		m.state = stateUserList
	case stateUserDelete:
		if m.form.GetBool("confirmDelete") {
			err := db.DeleteUser(m.targetUser.Username)
			if err != nil {
				m.statusMsg = fmt.Sprintf("Error deleting user: %v", err)
			} else {
				m.statusMsg = fmt.Sprintf("User '%s' successfully deleted", m.targetUser.Username)
				m.selectedUserCursor = 0
				m.updateLastHighlighted()
			}
		} else {
			m.statusMsg = "User deletion cancelled"
		}
		m.state = stateUserList
	case stateConfig:
		newPath := m.form.GetString("newPath")
		if newPath == m.dbPath {
			m.statusMsg = "Database path unchanged"
			m.state = stateMenu
			break
		}
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
	case stateUserCreate, stateUserDelete, stateUserChangePassword, stateUserRename:
		m.state = stateUserList
	case stateConfig:
		m.state = stateMenu
	}
}
