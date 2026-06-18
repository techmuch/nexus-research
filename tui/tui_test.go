package tui

import (
	"os"
	"strings"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/techmuch/nexus-research/db"
)

func TestMain(m *testing.M) {
	_ = db.InitDB(":memory:")
	_ = db.CreateUser("admin", "adminpassword", true)
	code := m.Run()
	_ = db.CloseDB()
	os.Exit(code)
}

func TestModelLifecycle(t *testing.T) {
	m := NewModel(":memory:")

	// 1. Test Init()
	cmd := m.Init()
	if cmd != nil {
		t.Errorf("expected Init to return nil, got %T", cmd)
	}

	// 2. Test Default State and View
	if m.state != stateMenu {
		t.Errorf("expected initial state to be stateMenu, got %v", m.state)
	}
	viewStr := m.View()
	if !strings.Contains(viewStr, "NEXUS RESEARCH STATION") {
		t.Errorf("expected view to contain header, got:\n%s", viewStr)
	}
	if !strings.Contains(viewStr, "User Management") {
		t.Errorf("expected view to contain 'User Management' menu option")
	}

	// 3. Test Navigation Down
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor to be 1 after moving down, got %d", m.cursor)
	}

	// Test Navigation Up
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor to be 0 after moving up, got %d", m.cursor)
	}

	// Test Boundary Navigation Up (should not go below 0)
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m.cursor)
	}

	// 4. Test Selection of User List
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to transition to stateUserList, got %v", m.state)
	}

	// Verify User List view
	viewStr = m.View()
	if !strings.Contains(viewStr, "Registered User Accounts") {
		t.Errorf("expected view to contain User List header, got:\n%s", viewStr)
	}
	if !strings.Contains(viewStr, "admin") {
		t.Errorf("expected view to contain registered 'admin' user, got:\n%s", viewStr)
	}

	// Go Back using esc
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateMenu {
		t.Errorf("expected state to return to stateMenu on esc, got %v", m.state)
	}

	// 5. Test System Status view
	m.cursor = 2
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateStatus {
		t.Errorf("expected state to transition to stateStatus, got %v", m.state)
	}

	// Verify System Status view
	viewStr = m.View()
	if !strings.Contains(viewStr, "System Metrics") {
		t.Errorf("expected view to contain Status metrics, got:\n%s", viewStr)
	}
	if !strings.Contains(viewStr, "Database Path") {
		t.Errorf("expected status block to contain Database Path key")
	}

	// Go back using enter
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateMenu {
		t.Errorf("expected state to return to stateMenu on enter, got %v", m.state)
	}

	// 6. Test Quit Command Option
	m.cursor = 3
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Errorf("expected quit cmd to be returned on exit select")
	}

	// 7. Test Quit key 'q'
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Errorf("expected quit cmd to be returned on key 'q'")
	}
}

func TestTUIFormsTransition(t *testing.T) {
	m := NewModel(":memory:")

	// Transition to stateUserList
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	// Press 'a' to enter user creation state
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = newModel.(Model)
	if m.state != stateUserCreate {
		t.Errorf("expected state to transition to stateUserCreate, got %v", m.state)
	}
	if m.form == nil {
		t.Errorf("expected form to be initialized for user creation")
	}
	if cmd == nil {
		t.Errorf("expected form init cmd to be returned")
	}

	// Go back using esc
	m.state = stateUserList
	m.form = nil

	// Press 'd' to enter user deletion state
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = newModel.(Model)
	if m.state != stateUserDelete {
		t.Errorf("expected state to transition to stateUserDelete, got %v", m.state)
	}
	if m.form == nil {
		t.Errorf("expected form to be initialized for user deletion")
	}

	// Go back to menu
	m.state = stateMenu
	m.form = nil
	m.cursor = 1 // config option

	// Press Enter to edit database configuration
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateConfig {
		t.Errorf("expected state to transition to stateConfig, got %v", m.state)
	}
	if m.form == nil {
		t.Errorf("expected form to be initialized for database config")
	}
}

func TestTUIFormHandlers(t *testing.T) {
	// Helper to set unexported results in huh.Form via unsafe pointers
	setFormResults := func(form *huh.Form, results map[string]any) {
		formPtr := unsafe.Pointer(form)
		resultsMapPtr := (*map[string]any)(unsafe.Pointer(uintptr(formPtr) + 8))
		if *resultsMapPtr == nil {
			*resultsMapPtr = make(map[string]any)
		}
		for k, v := range results {
			(*resultsMapPtr)[k] = v
		}
	}

	// 1. Test handleFormCompletion for stateUserCreate
	m := NewModel(":memory:")
	m.state = stateUserCreate
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("username"),
			huh.NewInput().Key("password"),
			huh.NewConfirm().Key("isAdmin"),
		),
	)
	setFormResults(m.form, map[string]any{
		"username": "testformuser",
		"password": "testformpass",
		"isAdmin":  true,
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "testformuser") {
		t.Errorf("expected statusMsg to contain username, got: %s", m.statusMsg)
	}

	// 2. Test handleFormCompletion for stateUserDelete
	m.state = stateUserDelete
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("targetUser"),
		),
	)
	setFormResults(m.form, map[string]any{
		"targetUser": "testformuser",
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "successfully deleted") {
		t.Errorf("expected statusMsg to show success, got: %s", m.statusMsg)
	}

	// Test handleFormCompletion for stateUserDelete failure (non-existent user)
	m.state = stateUserDelete
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("targetUser"),
		),
	)
	setFormResults(m.form, map[string]any{
		"targetUser": "nonexistentuser",
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList on delete failure, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "Error deleting user") {
		t.Errorf("expected statusMsg to show error, got: %s", m.statusMsg)
	}

	// 3. Test handleFormCompletion for stateConfig
	m.state = stateConfig
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("newPath"),
		),
	)
	setFormResults(m.form, map[string]any{
		"newPath": ":memory:",
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateMenu {
		t.Errorf("expected state to return to stateMenu, got %v", m.state)
	}

	// Test handleFormCompletion for stateConfig failure
	m.state = stateConfig
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("newPath"),
		),
	)
	setFormResults(m.form, map[string]any{
		"newPath": "/nonexistentdir/nexus.db",
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateMenu {
		t.Errorf("expected state to return to stateMenu on config failure, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "Error changing DB config") {
		t.Errorf("expected statusMsg to show config error, got: %s", m.statusMsg)
	}

	// 4. Test handleFormAbortion
	m.state = stateUserCreate
	m.handleFormAbortion()
	if m.state != stateUserList {
		t.Errorf("expected stateUserList after user create abort, got %v", m.state)
	}

	m.state = stateConfig
	m.handleFormAbortion()
	if m.state != stateMenu {
		t.Errorf("expected stateMenu after config abort, got %v", m.state)
	}

	// 5. Test handleFormCompletion for stateUserDelete cancel
	m.state = stateUserDelete
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("targetUser"),
		),
	)
	setFormResults(m.form, map[string]any{
		"targetUser": "CANCEL",
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList on delete cancel, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "User deletion cancelled") {
		t.Errorf("expected statusMsg to show cancellation, got: %s", m.statusMsg)
	}
}

func TestUpdateFormForwarding(t *testing.T) {
	setFormResults := func(form *huh.Form, results map[string]any) {
		formPtr := unsafe.Pointer(form)
		resultsMapPtr := (*map[string]any)(unsafe.Pointer(uintptr(formPtr) + 8))
		if *resultsMapPtr == nil {
			*resultsMapPtr = make(map[string]any)
		}
		for k, v := range results {
			(*resultsMapPtr)[k] = v
		}
	}

	// 1. Form Completion via Update
	m := NewModel(":memory:")
	m.state = stateUserCreate
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("username"),
			huh.NewInput().Key("password"),
			huh.NewConfirm().Key("isAdmin"),
		),
	)
	
	setFormResults(m.form, map[string]any{
		"username": "userfromupdate",
		"password": "pwdfromupdate",
		"isAdmin":  true,
	})
	
	m.form.State = huh.StateCompleted
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList after form completed via Update, got %v", m.state)
	}
	if m.form != nil {
		t.Errorf("expected form to be cleared after completion")
	}
	
	// 2. Form Abortion via Update
	m.state = stateUserCreate
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("username"),
		),
	)
	m.form.State = huh.StateAborted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList after form aborted via Update, got %v", m.state)
	}
	if m.form != nil {
		t.Errorf("expected form to be cleared after abortion")
	}
}

func TestExtraKeys(t *testing.T) {
	m := NewModel(":memory:")
	
	// Test navigation with Arrow keys "down" and "up"
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor to be 1, got %d", m.cursor)
	}
	
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor to be 0, got %d", m.cursor)
	}
	
	// Test ctrl+c when in stateMenu (should quit)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Errorf("expected quit cmd from ctrl+c in stateMenu")
	}
	
	// Test q when in stateUserList (should return to stateMenu)
	m.state = stateUserList
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = newModel.(Model)
	if m.state != stateMenu {
		t.Errorf("expected state to return to stateMenu, got %v", m.state)
	}
	
	// Test esc key behavior from status state
	m.state = stateStatus
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateMenu {
		t.Errorf("expected state to return to stateMenu, got %v", m.state)
	}
}

func TestDeleteNoUsers(t *testing.T) {
	_ = db.CloseDB()
	_ = db.InitDB(":memory:")

	m := NewModel(":memory:")
	m.state = stateUserList
	
	// Press 'd' to delete when there are no users
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to remain stateUserList, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "No users available to delete") {
		t.Errorf("expected statusMsg to show 'No users available to delete', got: %s", m.statusMsg)
	}
	if cmd != nil {
		t.Errorf("expected nil cmd, got %T", cmd)
	}
	
	// Reinitialize DB for other tests
	_ = db.CloseDB()
	_ = db.InitDB(":memory:")
	_ = db.CreateUser("admin", "adminpassword", true)
}

func TestUserListViewError(t *testing.T) {
	_ = db.CloseDB() // Force DB error
	m := NewModel(":memory:")
	m.state = stateUserList
	viewStr := m.View()
	if !strings.Contains(viewStr, "Error listing users") {
		t.Errorf("expected view to contain 'Error listing users', got:\n%s", viewStr)
	}
	_ = db.InitDB(":memory:") // restore
	_ = db.CreateUser("admin", "adminpassword", true)
}

func TestStatusViewNonExistentDB(t *testing.T) {
	m := NewModel("/nonexistentdir/nexus.db")
	m.state = stateStatus
	viewStr := m.View()
	if !strings.Contains(viewStr, "0 bytes") {
		t.Errorf("expected view to display 0 bytes for non-existent database file, got:\n%s", viewStr)
	}
}
