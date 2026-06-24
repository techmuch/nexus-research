package tui

import (
	"fmt"
	"os"
	"reflect"
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
	if !strings.Contains(viewStr, "User Account Directory") {
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
	if !strings.Contains(viewStr, "Path:") {
		t.Errorf("expected status block to contain Database Path key")
	}

	// Go back using enter
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateMenu {
		t.Errorf("expected state to return to stateMenu on enter, got %v", m.state)
	}

	// 6. Test Quit Command Option
	m.cursor = 4
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
	if m.selectedUserCursor != 1 {
		t.Errorf("expected selectedUserCursor to follow newly created user to index 1, got %d", m.selectedUserCursor)
	}

	// 2. Test handleFormCompletion for stateUserDelete
	m.state = stateUserDelete
	m.targetUser = db.User{Username: "testformuser"}
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Key("confirmDelete"),
		),
	)
	setFormResults(m.form, map[string]any{
		"confirmDelete": true,
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
	m.targetUser = db.User{Username: "nonexistentuser"}
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Key("confirmDelete"),
		),
	)
	setFormResults(m.form, map[string]any{
		"confirmDelete": true,
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
	if !strings.Contains(m.statusMsg, "No user highlighted") {
		t.Errorf("expected statusMsg to show 'No user highlighted', got: %s", m.statusMsg)
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

func helperSetFormResults(form *huh.Form, results map[string]any) {
	formPtr := unsafe.Pointer(form)
	resultsMapPtr := (*map[string]any)(unsafe.Pointer(uintptr(formPtr) + 8))
	if *resultsMapPtr == nil {
		*resultsMapPtr = make(map[string]any)
	}
	for k, v := range results {
		(*resultsMapPtr)[k] = v
	}
}

func TestUserDetailFlow(t *testing.T) {
	_ = db.CloseDB()
	_ = db.InitDB(":memory:")
	_ = db.CreateUser("testdetail", "password123", false)
	_ = db.CreateUser("adminuser", "password123", true)

	m := NewModel(":memory:")
	m.state = stateUserList

	// 1. Check navigation
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)
	if m.selectedUserCursor != 1 {
		t.Errorf("expected selectedUserCursor to be 1, got %d", m.selectedUserCursor)
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)
	if m.selectedUserCursor != 0 {
		t.Errorf("expected selectedUserCursor to be 0, got %d", m.selectedUserCursor)
	}

	// 2. Check View rendering of split-pane
	viewStr := m.View()
	if !strings.Contains(viewStr, "User Account Directory") {
		t.Errorf("expected view to contain User Account Directory header, got:\n%s", viewStr)
	}
	if !strings.Contains(viewStr, "PROFILE: adminuser") {
		t.Errorf("expected view to contain profile panel for adminuser, got:\n%s", viewStr)
	}

	// 3. Test Search mode (/)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = newModel.(Model)
	if !m.searching {
		t.Errorf("expected searching to be true")
	}

	// Type search characters: "t", "e", "s", "t"
	for _, char := range "test" {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		m = newModel.(Model)
	}
	if m.searchQuery != "test" {
		t.Errorf("expected searchQuery to be 'test', got '%s'", m.searchQuery)
	}

	// Check that view is filtered (only testdetail matches)
	viewStr = m.View()
	if !strings.Contains(viewStr, "PROFILE: testdetail") {
		t.Errorf("expected view to select testdetail after filtering, got:\n%s", viewStr)
	}

	// Backspace search
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(Model)
	if m.searchQuery != "tes" {
		t.Errorf("expected query to be 'tes', got '%s'", m.searchQuery)
	}

	// Space key in search
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newModel.(Model)
	if m.searchQuery != "tes " {
		t.Errorf("expected query to have space, got '%s'", m.searchQuery)
	}

	// Exit search with Esc (clears query)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.searching || m.searchQuery != "" {
		t.Errorf("expected search to be cleared and deactivated")
	}

	// 4. Test Change Password direct key (p)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = newModel.(Model)
	if m.state != stateUserChangePassword {
		t.Errorf("expected state stateUserChangePassword, got %v", m.state)
	}
	// Abort password form
	m.form.State = huh.StateAborted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList on password abort, got %v", m.state)
	}

	// Re-trigger and complete password change
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = newModel.(Model)
	helperSetFormResults(m.form, map[string]any{
		"newPassword": "newpassword123",
	})
	m.form.State = huh.StateCompleted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList on password change completion, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "Password for 'adminuser' updated successfully") {
		t.Errorf("expected success status message, got: %s", m.statusMsg)
	}

	// 5. Test Rename User direct key (r)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = newModel.(Model)
	if m.state != stateUserRename {
		t.Errorf("expected state stateUserRename, got %v", m.state)
	}
	// Abort rename
	m.form.State = huh.StateAborted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList on rename abort, got %v", m.state)
	}

	// Re-trigger and complete rename
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = newModel.(Model)
	helperSetFormResults(m.form, map[string]any{
		"newUsername": "z_adminuser",
	})
	m.form.State = huh.StateCompleted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList on rename completion, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "User renamed from 'adminuser' to 'z_adminuser'") {
		t.Errorf("expected success status message, got: %s", m.statusMsg)
	}
	if m.selectedUserCursor != 1 {
		t.Errorf("expected cursor to follow renamed user to index 1, got %d", m.selectedUserCursor)
	}

	// 6. Test Toggle Status (t)
	// Currently active
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = newModel.(Model)
	if !m.targetUser.IsDisabled {
		t.Errorf("expected user to be disabled")
	}
	if !strings.Contains(m.statusMsg, "disabled") {
		t.Errorf("expected statusMsg to show disabled status, got: %s", m.statusMsg)
	}

	// Toggle back to active
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = newModel.(Model)
	if m.targetUser.IsDisabled {
		t.Errorf("expected user to be enabled")
	}

	// 7. Test Delete User (d)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = newModel.(Model)
	if m.state != stateUserDelete {
		t.Errorf("expected state stateUserDelete, got %v", m.state)
	}
	// Cancel deletion
	helperSetFormResults(m.form, map[string]any{
		"confirmDelete": false,
	})
	m.form.State = huh.StateCompleted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "deletion cancelled") {
		t.Errorf("expected cancellation status message, got: %s", m.statusMsg)
	}

	// Delete and confirm
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = newModel.(Model)
	helperSetFormResults(m.form, map[string]any{
		"confirmDelete": true,
	})
	m.form.State = huh.StateCompleted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "successfully deleted") {
		t.Errorf("expected deletion success message, got: %s", m.statusMsg)
	}

	// 8. Test Escape / q to exit user list
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateMenu {
		t.Errorf("expected state stateMenu on Esc key, got %v", m.state)
	}
}

func TestFormCompletionErrorPaths(t *testing.T) {
	_ = db.CloseDB()
	_ = db.InitDB(":memory:")
	_ = db.CreateUser("erroruser", "password123", false)

	m := NewModel(":memory:")
	m.targetUser = db.User{Username: "erroruser", IsDisabled: false}

	// 1. Password change DB error
	_ = db.CloseDB() // Force error on DB call
	m.state = stateUserChangePassword
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("newPassword"),
		),
	)
	helperSetFormResults(m.form, map[string]any{
		"newPassword": "newpassword123",
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateUserList {
		t.Errorf("expected state to transition to stateUserList on error, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "Error changing password") {
		t.Errorf("expected statusMsg to contain DB error, got: %s", m.statusMsg)
	}

	// 2. Rename user DB error
	m.state = stateUserRename
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("newUsername"),
		),
	)
	helperSetFormResults(m.form, map[string]any{
		"newUsername": "erroruser_new",
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateUserList {
		t.Errorf("expected state to transition to stateUserList on error, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "Error renaming user") {
		t.Errorf("expected statusMsg to contain DB error, got: %s", m.statusMsg)
	}

	// 3. Delete user DB error
	m.state = stateUserDelete
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Key("confirmDelete"),
		),
	)
	helperSetFormResults(m.form, map[string]any{
		"confirmDelete": true,
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateUserList {
		t.Errorf("expected state to transition to stateUserList on error, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "Error deleting user") {
		t.Errorf("expected statusMsg to contain DB error, got: %s", m.statusMsg)
	}

	// Restore DB for further testing if needed
	_ = db.InitDB(":memory:")
}

func TestTUIFormAbortionSubStates(t *testing.T) {
	m := NewModel(":memory:")
	m.targetUser = db.User{Username: "abortuser"}

	m.state = stateUserChangePassword
	m.handleFormAbortion()
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList on password form abort, got %v", m.state)
	}

	m.state = stateUserRename
	m.handleFormAbortion()
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList on rename form abort, got %v", m.state)
	}
}

func TestStatusViewInMemory(t *testing.T) {
	m := NewModel(":memory:")
	m.state = stateStatus
	viewStr := m.View()
	if !strings.Contains(viewStr, "In-Memory (Volatile)") {
		t.Errorf("expected view to display in-memory DB message, got:\n%s", viewStr)
	}
}

func TestTUIToggleStatusErrorPaths(t *testing.T) {
	m := NewModel(":memory:")
	m.state = stateUserList

	// Toggle status with no user selected (empty list)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = newModel.(Model)
	if m.statusMsg != "No user highlighted" {
		t.Errorf("expected 'No user highlighted' error status message, got: %s", m.statusMsg)
	}

	// Change password with no user selected
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = newModel.(Model)
	if m.statusMsg != "No user highlighted" {
		t.Errorf("expected 'No user highlighted' error status message, got: %s", m.statusMsg)
	}

	// Rename with no user selected
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = newModel.(Model)
	if m.statusMsg != "No user highlighted" {
		t.Errorf("expected 'No user highlighted' error status message, got: %s", m.statusMsg)
	}

	// Delete with no user selected
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = newModel.(Model)
	if m.statusMsg != "No user highlighted" {
		t.Errorf("expected 'No user highlighted' error status message, got: %s", m.statusMsg)
	}
}

func TestTUIMoreEdgeCases(t *testing.T) {
	// 1. Test clampCursor boundary cases
	m := NewModel(":memory:")
	
	// Test clampCursor(0)
	m.selectedUserCursor = 5
	m.clampCursor(0)
	if m.selectedUserCursor != 0 {
		t.Errorf("expected cursor to be 0 for numUsers = 0, got %d", m.selectedUserCursor)
	}

	// Test clampCursor with negative cursor
	m.selectedUserCursor = -3
	m.clampCursor(5)
	if m.selectedUserCursor != 0 {
		t.Errorf("expected cursor to clamp to 0, got %d", m.selectedUserCursor)
	}

	// Test clampCursor with cursor >= numUsers
	m.selectedUserCursor = 10
	m.clampCursor(5)
	if m.selectedUserCursor != 4 {
		t.Errorf("expected cursor to clamp to 4, got %d", m.selectedUserCursor)
	}

	// 2. Test empty users list view with query vs no query
	_ = db.CloseDB()
	_ = db.InitDB(":memory:") // empty db

	m.state = stateUserList
	m.searchQuery = ""
	viewStr := m.View()
	if !strings.Contains(viewStr, "No registered user accounts.") {
		t.Errorf("expected view to contain 'No registered user accounts.', got:\n%s", viewStr)
	}

	m.searchQuery = "test"
	viewStr = m.View()
	if !strings.Contains(viewStr, "No users match filter query.") {
		t.Errorf("expected view to contain 'No users match filter query.', got:\n%s", viewStr)
	}

	// 3. Test detail profile view when selectedUserCursor is out of bounds
	_ = db.CreateUser("user1", "password", false)
	m.selectedUserCursor = 10
	viewStr = m.View()
	if !strings.Contains(viewStr, "No user selected.") {
		t.Errorf("expected view to render 'No user selected.' in detail pane, got:\n%s", viewStr)
	}

	// Restore db for remaining tests
	_ = db.CloseDB()
	_ = db.InitDB(":memory:")
}

func TestFormValidators(t *testing.T) {
	// Reinit db
	_ = db.CloseDB()
	_ = db.InitDB(":memory:")
	_ = db.CreateUser("admin", "adminpassword", true)

	m := NewModel(":memory:")

	// Helper to extract validator function using unsafe.Pointer to bypass unexported fields
	getValidator := func(form *huh.Form, groupIdx, fieldIdx int) func(string) error {
		formVal := reflect.ValueOf(form).Elem()
		selectorVal := formVal.FieldByName("selector").Elem()
		groupsSlice := selectorVal.FieldByName("items")
		groupVal := groupsSlice.Index(groupIdx).Elem()
		groupSelectorVal := groupVal.FieldByName("selector").Elem()
		fieldsSlice := groupSelectorVal.FieldByName("items")
		fieldVal := fieldsSlice.Index(fieldIdx).Elem().Elem()
		validateFuncVal := fieldVal.FieldByName("validate")
		if !validateFuncVal.IsValid() || validateFuncVal.IsNil() {
			return nil
		}
		validateFuncPtr := (*func(string) error)(unsafe.Pointer(validateFuncVal.UnsafeAddr()))
		return *validateFuncPtr
	}

	// 1. Create User form validators
	m.state = stateUserList
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = newModel.(Model)
	if m.state != stateUserCreate || m.form == nil {
		t.Fatalf("expected form to be initialized for user creation")
	}
	
	usernameValidator := getValidator(m.form, 0, 0)
	if usernameValidator == nil {
		t.Fatal("username validator not found")
	}
	if err := usernameValidator("   "); err == nil || err.Error() != "username cannot be empty" {
		t.Errorf("expected username empty error, got %v", err)
	}
	if err := usernameValidator("testuser"); err != nil {
		t.Errorf("expected valid username, got %v", err)
	}
	if err := usernameValidator("admin"); err == nil || err.Error() != "user already exists" {
		t.Errorf("expected user already exists error for admin, got %v", err)
	}

	passwordValidator := getValidator(m.form, 0, 1)
	if passwordValidator == nil {
		t.Fatal("password validator not found")
	}
	if err := passwordValidator("123"); err == nil || err.Error() != "password must be at least 4 characters" {
		t.Errorf("expected short password error, got %v", err)
	}
	if err := passwordValidator("1234"); err != nil {
		t.Errorf("expected valid password, got %v", err)
	}

	// 2. Rename User form validators
	m.state = stateUserList
	m.selectedUserCursor = 0
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = newModel.(Model)
	if m.state != stateUserRename || m.form == nil {
		t.Fatalf("expected form to be initialized for user renaming")
	}

	renameValidator := getValidator(m.form, 0, 0)
	if renameValidator == nil {
		t.Fatal("rename validator not found")
	}
	if err := renameValidator("   "); err == nil || err.Error() != "username cannot be empty" {
		t.Errorf("expected rename empty error, got %v", err)
	}
	if err := renameValidator("newuser"); err != nil {
		t.Errorf("expected valid rename, got %v", err)
	}
	// admin is the current user, so renaming to admin is allowed
	if err := renameValidator("admin"); err != nil {
		t.Errorf("expected valid rename to own name, got %v", err)
	}
	// create another user to test rename clash
	_ = db.CreateUser("anotheruser", "pass", false)
	if err := renameValidator("anotheruser"); err == nil || err.Error() != "user already exists" {
		t.Errorf("expected user already exists error for anotheruser, got %v", err)
	}

	// 3. Change Password form validators
	m.state = stateUserList
	m.selectedUserCursor = 0
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = newModel.(Model)
	if m.state != stateUserChangePassword || m.form == nil {
		t.Fatalf("expected form to be initialized for password change")
	}

	changePwdValidator := getValidator(m.form, 0, 0)
	if changePwdValidator == nil {
		t.Fatal("change pwd validator not found")
	}
	if err := changePwdValidator("123"); err == nil || err.Error() != "password must be at least 4 characters" {
		t.Errorf("expected short password error, got %v", err)
	}
	if err := changePwdValidator("1234"); err != nil {
		t.Errorf("expected valid password, got %v", err)
	}

	// 4. Config DB Path form validators
	m.state = stateMenu
	m.cursor = 1
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateConfig || m.form == nil {
		t.Fatalf("expected form to be initialized for database config")
	}

	configValidator := getValidator(m.form, 0, 0)
	if configValidator == nil {
		t.Fatal("config validator not found")
	}
	if err := configValidator("   "); err == nil || err.Error() != "database path cannot be empty" {
		t.Errorf("expected config empty error, got %v", err)
	}
	if err := configValidator("nexus.db"); err != nil {
		t.Errorf("expected valid config path, got %v", err)
	}
}

func TestTUIResponsiveLayoutAndHelp(t *testing.T) {
	_ = db.CloseDB()
	_ = db.InitDB(":memory:")
	// No users initially

	m := NewModel(":memory:")
	m.state = stateUserList

	// 1. Verify contextual command help card (no users case)
	viewStr := m.View()
	if !strings.Contains(viewStr, "Add: a") || strings.Contains(viewStr, "Password: p") {
		t.Errorf("expected only add and esc keyboard commands in footer when empty, got:\n%s", viewStr)
	}

	// 2. Add a user and verify detail help changes
	_ = db.CreateUser("user1", "pass", false)
	viewStr = m.View()
	if !strings.Contains(viewStr, "Password: p") {
		t.Errorf("expected full user keyboard commands in footer when users exist, got:\n%s", viewStr)
	}

	// 3. Test clear status message on key navigation
	m.statusMsg = "Important notification"
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)
	if m.statusMsg != "" {
		t.Errorf("expected statusMsg to be cleared on navigation, got '%s'", m.statusMsg)
	}

	// Reset status message and test search typing clearing it
	m.statusMsg = "Some notification"
	m.searching = true
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = newModel.(Model)
	if m.statusMsg != "" {
		t.Errorf("expected statusMsg to be cleared on search typing, got '%s'", m.statusMsg)
	}
	m.searching = false

	// 4. Test WindowSizeMsg responsive layout updates
	// Normal wide layout
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = newModel.(Model)
	if m.width != 100 || m.height != 40 {
		t.Errorf("expected width=100 height=40, got width=%d height=%d", m.width, m.height)
	}
	viewStr = m.View()
	if viewStr == "" {
		t.Errorf("expected non-empty view string")
	}

	// Narrow vertical layout
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 40})
	m = newModel.(Model)
	viewStr = m.View()
	if viewStr == "" {
		t.Errorf("expected non-empty view string on narrow window size")
	}

	// Test status dashboard views under different sizes
	m.state = stateStatus
	// Wide status view (should contain DATABASE ENGINE and USER METRICS)
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(Model)
	viewStr = m.View()
	if !strings.Contains(viewStr, "DATABASE ENGINE") || !strings.Contains(viewStr, "USER METRICS") {
		t.Errorf("expected status cards in wide view, got:\n%s", viewStr)
	}

	// Narrow status view
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 50, Height: 40})
	m = newModel.(Model)
	viewStr = m.View()
	if !strings.Contains(viewStr, "DATABASE ENGINE") {
		t.Errorf("expected status cards in narrow view, got:\n%s", viewStr)
	}

	// Test menu views under different sizes
	m.state = stateMenu
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = newModel.(Model)
	viewStr = m.View()
	if !strings.Contains(viewStr, "User Management") {
		t.Errorf("expected menu options in wide menu view, got:\n%s", viewStr)
	}

	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 40})
	m = newModel.(Model)
	viewStr = m.View()
	if !strings.Contains(viewStr, "User Management") {
		t.Errorf("expected menu options in narrow menu view, got:\n%s", viewStr)
	}

	// Test esc key behavior clears searchQuery first in stateUserList
	m.state = stateUserList
	m.searchQuery = "filteredquery"
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.searchQuery != "" {
		t.Errorf("expected searchQuery to be cleared on first Esc, got '%s'", m.searchQuery)
	}
	if m.state != stateUserList {
		t.Errorf("expected state to remain stateUserList on first Esc, got %v", m.state)
	}

	// Pressing esc again goes to menu
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateMenu {
		t.Errorf("expected state to transition to stateMenu on second Esc, got %v", m.state)
	}
}

func TestTUIFormBordersAndErrorStyle(t *testing.T) {
	m := NewModel(":memory:")
	m.state = stateUserCreate
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("username"),
		),
	)

	// 1. Test form width styling with large width (clamps to 60)
	m.width = 100
	viewStr := m.View()
	if viewStr == "" {
		t.Errorf("expected view not to be empty")
	}

	// 2. Test form width styling with small width (clamps to 30)
	m.width = 20
	viewStr = m.View()
	if viewStr == "" {
		t.Errorf("expected view not to be empty")
	}

	// 3. Test status message with "Error:" prefix in stateMenu
	m.state = stateMenu
	m.statusMsg = "Error: something failed"
	viewStr = m.View()
	if !strings.Contains(viewStr, "Error: something failed") {
		t.Errorf("expected statusMsg to be rendered in menu")
	}

	// 4. Test status message with "Error:" prefix in stateUserList
	m.state = stateUserList
	viewStr = m.View()
	if !strings.Contains(viewStr, "Error: something failed") {
		t.Errorf("expected statusMsg to be rendered in user list")
	}
}

func TestConfigPathUnchanged(t *testing.T) {
	m := NewModel(":memory:")
	m.state = stateConfig
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("newPath"),
		),
	)
	helperSetFormResults(m.form, map[string]any{
		"newPath": ":memory:",
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateMenu {
		t.Errorf("expected state to transition to stateMenu, got %v", m.state)
	}
	if m.statusMsg != "Database path unchanged" {
		t.Errorf("expected statusMsg to be 'Database path unchanged', got '%s'", m.statusMsg)
	}
}

func TestUserRenameUnchanged(t *testing.T) {
	m := NewModel(":memory:")
	m.state = stateUserRename
	m.targetUser = db.User{Username: "testuser"}
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("newUsername"),
		),
	)
	helperSetFormResults(m.form, map[string]any{
		"newUsername": "testuser",
	})
	m.form.State = huh.StateCompleted
	m.handleFormCompletion()

	if m.state != stateUserList {
		t.Errorf("expected state to transition to stateUserList, got %v", m.state)
	}
	if m.statusMsg != "Username unchanged" {
		t.Errorf("expected statusMsg to be 'Username unchanged', got '%s'", m.statusMsg)
	}
}

func TestTUIFormFooter(t *testing.T) {
	m := NewModel(":memory:")
	m.state = stateUserCreate
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("username"),
		),
	)

	viewStr := m.View()
	if !strings.Contains(viewStr, "Next Field: Tab") {
		t.Errorf("expected form view to contain form footer navigation guide, got:\n%s", viewStr)
	}
}

func TestTUISearchSelectionPersistence(t *testing.T) {
	_ = db.CloseDB()
	_ = db.InitDB(":memory:")

	_ = db.CreateUser("admin", "pass", true)
	_ = db.CreateUser("bob", "pass", false)
	_ = db.CreateUser("testdetail", "pass", false)

	m := NewModel(":memory:")
	m.state = stateUserList

	// Highlight testdetail (sorted list: admin, bob, testdetail -> index 2)
	m.selectedUserCursor = 2
	m.updateLastHighlighted()

	// Type search query character: "t" (filtered list: only testdetail -> index 0)
	m.updateSearchQuery("t")
	if m.selectedUserCursor != 0 {
		t.Errorf("expected selectedUserCursor to remain focused on testdetail (index 0), got %d", m.selectedUserCursor)
	}

	// Filter out the selected user
	m.updateSearchQuery("tx")
	if m.selectedUserCursor != 0 {
		t.Errorf("expected selectedUserCursor to clamp to 0, got %d", m.selectedUserCursor)
	}

	// Restore search filter
	m.updateSearchQuery("t")
	if m.selectedUserCursor != 0 {
		t.Errorf("expected selectedUserCursor to focus on testdetail (index 0), got %d", m.selectedUserCursor)
	}
	
	m.updateSearchQuery("") // clear search

	if m.selectedUserCursor != 2 {
		t.Errorf("expected selectedUserCursor to restore focus to testdetail (index 2), got %d", m.selectedUserCursor)
	}
}

func TestTUIViewportScrolling(t *testing.T) {
	_ = db.CloseDB()
	_ = db.InitDB(":memory:")

	for i := 1; i <= 15; i++ {
		username := fmt.Sprintf("user%02d", i)
		_ = db.CreateUser(username, "password", false)
	}

	m := NewModel(":memory:")
	m.state = stateUserList
	
	m.height = 20
	m.width = 80

	m.clampCursor(15)
	if m.scrollOffset != 0 {
		t.Errorf("expected initial scrollOffset to be 0, got %d", m.scrollOffset)
	}

	m.selectedUserCursor = 5
	m.clampCursor(15)
	if m.scrollOffset != 0 {
		t.Errorf("expected scrollOffset to be 0 when cursor is 5, got %d", m.scrollOffset)
	}

	m.selectedUserCursor = 10
	m.clampCursor(15)
	if m.scrollOffset != 3 {
		t.Errorf("expected scrollOffset to be 3 when cursor is 10, got %d", m.scrollOffset)
	}

	viewStr := m.View()
	if !strings.Contains(viewStr, "Showing 4-11 of 15 accounts") {
		t.Errorf("expected view to contain Showing 4-11 of 15 accounts, got:\n%s", viewStr)
	}

	m.selectedUserCursor = 2
	m.clampCursor(15)
	if m.scrollOffset != 2 {
		t.Errorf("expected scrollOffset to slide up to 2 when cursor is 2, got %d", m.scrollOffset)
	}

	viewStr = m.View()
	if !strings.Contains(viewStr, "Showing 3-10 of 15 accounts") {
		t.Errorf("expected view to contain Showing 3-10 of 15 accounts, got:\n%s", viewStr)
	}
}



