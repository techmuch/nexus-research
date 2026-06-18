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
	
	// Test navigation and boundary of user list cursor
	// Down movement
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)
	if m.selectedUserCursor != 1 {
		t.Errorf("expected selectedUserCursor to be 1, got %d", m.selectedUserCursor)
	}
	// Up movement
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)
	if m.selectedUserCursor != 0 {
		t.Errorf("expected selectedUserCursor to be 0, got %d", m.selectedUserCursor)
	}

	// 1. Enter detail view for non-admin
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserDetail {
		t.Fatalf("expected state to transition to stateUserDetail, got %v", m.state)
	}
	if m.targetUser.Username != "adminuser" { // users listed alphabetically, adminuser first
		t.Errorf("expected targetUser username to be 'adminuser', got '%s'", m.targetUser.Username)
	}

	// 2. View in stateUserDetail for admin user
	viewStr := m.View()
	if !strings.Contains(viewStr, "User Profile: adminuser") {
		t.Errorf("expected view to contain profile header, got:\n%s", viewStr)
	}
	if !strings.Contains(viewStr, "Role       : Admin") {
		t.Errorf("expected view to show admin role, got:\n%s", viewStr)
	}

	// Press ESC to go back to list
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Fatalf("expected state to return to stateUserList, got %v", m.state)
	}

	// Move to testdetail and select
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model) // cursor is 1, which is testdetail
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.targetUser.Username != "testdetail" {
		t.Fatalf("expected targetUser username to be 'testdetail', got '%s'", m.targetUser.Username)
	}

	// Check view for non-admin
	viewStr = m.View()
	if !strings.Contains(viewStr, "Role       : User") {
		t.Errorf("expected view to show user role, got:\n%s", viewStr)
	}

	// 3. Navigation inside detail menu (up/down check boundaries)
	if m.detailCursor != 0 {
		t.Errorf("expected detailCursor to start at 0, got %d", m.detailCursor)
	}
	
	// Move down
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(Model)
	if m.detailCursor != 1 {
		t.Errorf("expected detailCursor to be 1, got %d", m.detailCursor)
	}

	// Move down to boundary
	for i := 0; i < 5; i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = newModel.(Model)
	}
	if m.detailCursor != 4 {
		t.Errorf("expected detailCursor to cap at 4, got %d", m.detailCursor)
	}

	// Move up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(Model)
	if m.detailCursor != 3 {
		t.Errorf("expected detailCursor to be 3, got %d", m.detailCursor)
	}

	// Move back up to 0
	for i := 0; i < 5; i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		m = newModel.(Model)
	}
	if m.detailCursor != 0 {
		t.Errorf("expected detailCursor to cap at 0, got %d", m.detailCursor)
	}

	// 4. Test Change Password select
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserChangePassword {
		t.Errorf("expected state to transition to stateUserChangePassword, got %v", m.state)
	}
	if m.form == nil {
		t.Fatalf("expected form to be initialized")
	}
	
	// Abort form and verify state returns to detail
	m.form.State = huh.StateAborted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateUserDetail {
		t.Errorf("expected state to return to stateUserDetail on abort, got %v", m.state)
	}

	// Re-select Change Password
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	// Complete form
	helperSetFormResults(m.form, map[string]any{
		"newPassword": "newpassword123",
	})
	m.form.State = huh.StateCompleted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserDetail {
		t.Errorf("expected state to return to stateUserDetail on password change completion, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "Password for 'testdetail' updated successfully") {
		t.Errorf("expected success status message, got: %s", m.statusMsg)
	}

	// Verify password was indeed changed
	ok, err := db.AuthenticateUser("testdetail", "newpassword123")
	if err != nil || !ok {
		t.Errorf("expected authentication with new password to succeed")
	}

	// 5. Test Rename User select
	m.detailCursor = 1
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserRename {
		t.Errorf("expected state to transition to stateUserRename, got %v", m.state)
	}
	// Abort
	m.form.State = huh.StateAborted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateUserDetail {
		t.Errorf("expected state to return to stateUserDetail on rename abort, got %v", m.state)
	}

	// Re-select Rename User
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	// Complete Rename
	helperSetFormResults(m.form, map[string]any{
		"newUsername": "testdetail_new",
	})
	m.form.State = huh.StateCompleted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserDetail {
		t.Errorf("expected state to return to stateUserDetail on rename completion, got %v", m.state)
	}
	if m.targetUser.Username != "testdetail_new" {
		t.Errorf("expected targetUser username to be updated to 'testdetail_new'")
	}
	if !strings.Contains(m.statusMsg, "User renamed from 'testdetail' to 'testdetail_new'") {
		t.Errorf("expected success status message, got: %s", m.statusMsg)
	}

	// Verify old username fails and new username succeeds
	ok, _ = db.AuthenticateUser("testdetail", "newpassword123")
	if ok {
		t.Errorf("expected old username authentication to fail")
	}
	ok, _ = db.AuthenticateUser("testdetail_new", "newpassword123")
	if !ok {
		t.Errorf("expected new username authentication to succeed")
	}

	// 6. Test Toggle Status
	m.detailCursor = 2
	// Verify view before toggling displays action as "Disable User Account"
	viewStr = m.View()
	if !strings.Contains(viewStr, "Disable User Account") {
		t.Errorf("expected detail view to offer disabling account")
	}
	// Select Toggle Status
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserDetail {
		t.Errorf("expected state to remain stateUserDetail, got %v", m.state)
	}
	if !m.targetUser.IsDisabled {
		t.Errorf("expected targetUser to be disabled")
	}
	if !strings.Contains(m.statusMsg, "User 'testdetail_new' is now disabled") {
		t.Errorf("expected disabled status message, got: %s", m.statusMsg)
	}
	// Verify authentication fails for disabled user
	ok, _ = db.AuthenticateUser("testdetail_new", "newpassword123")
	if ok {
		t.Errorf("expected disabled user authentication to fail")
	}

	// Verify view now displays action as "Enable User Account"
	viewStr = m.View()
	if !strings.Contains(viewStr, "Enable User Account") {
		t.Errorf("expected detail view to offer enabling account")
	}

	// Select Toggle Status again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.targetUser.IsDisabled {
		t.Errorf("expected targetUser to be enabled")
	}
	if !strings.Contains(m.statusMsg, "User 'testdetail_new' is now enabled") {
		t.Errorf("expected enabled status message, got: %s", m.statusMsg)
	}
	// Verify authentication succeeds for enabled user
	ok, _ = db.AuthenticateUser("testdetail_new", "newpassword123")
	if !ok {
		t.Errorf("expected enabled user authentication to succeed")
	}

	// 7. Test Delete User Cancel
	m.detailCursor = 3
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserDelete {
		t.Errorf("expected state to transition to stateUserDelete, got %v", m.state)
	}
	// Set confirmDelete to false
	helperSetFormResults(m.form, map[string]any{
		"confirmDelete": false,
	})
	m.form.State = huh.StateCompleted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserDetail {
		t.Errorf("expected state to return to stateUserDetail, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "User deletion cancelled") {
		t.Errorf("expected cancellation status message, got: %s", m.statusMsg)
	}

	// 8. Test Delete User Confirm
	m.detailCursor = 3
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserDelete {
		t.Errorf("expected state to transition to stateUserDelete, got %v", m.state)
	}
	// Set confirmDelete to true
	helperSetFormResults(m.form, map[string]any{
		"confirmDelete": true,
	})
	m.form.State = huh.StateCompleted
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList after deletion, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "User 'testdetail_new' successfully deleted") {
		t.Errorf("expected deletion success message, got: %s", m.statusMsg)
	}

	// Verify user is gone from db
	users, _ := db.ListUsers()
	for _, u := range users {
		if u.Username == "testdetail_new" {
			t.Errorf("expected deleted user to not exist in database")
		}
	}

	// 9. Back to List button
	m.state = stateUserDetail
	m.detailCursor = 4 // Back to User List
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList, got %v", m.state)
	}

	// 10. Test Escape/q keys inside stateUserDetail
	m.state = stateUserDetail
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList on Esc key, got %v", m.state)
	}

	m.state = stateUserDetail
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = newModel.(Model)
	if m.state != stateUserList {
		t.Errorf("expected state to return to stateUserList on 'q' key, got %v", m.state)
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

	if m.state != stateUserDetail {
		t.Errorf("expected state to transition to stateUserDetail on error, got %v", m.state)
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

	if m.state != stateUserDetail {
		t.Errorf("expected state to transition to stateUserDetail on error, got %v", m.state)
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

	if m.state != stateUserDetail {
		t.Errorf("expected state to transition to stateUserDetail on error, got %v", m.state)
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
	if m.state != stateUserDetail {
		t.Errorf("expected state to return to stateUserDetail on password form abort, got %v", m.state)
	}

	m.state = stateUserRename
	m.handleFormAbortion()
	if m.state != stateUserDetail {
		t.Errorf("expected state to return to stateUserDetail on rename form abort, got %v", m.state)
	}
}

func TestStatusViewInMemory(t *testing.T) {
	m := NewModel(":memory:")
	m.state = stateStatus
	viewStr := m.View()
	if !strings.Contains(viewStr, "In-memory database (volatile)") {
		t.Errorf("expected view to display in-memory DB message, got:\n%s", viewStr)
	}
}

func TestTUIToggleStatusDBError(t *testing.T) {
	_ = db.CloseDB() // Force error on DB call
	m := NewModel(":memory:")
	m.targetUser = db.User{Username: "testtoggle", IsDisabled: false}
	m.state = stateUserDetail
	m.detailCursor = 2 // Toggle Status

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if !strings.Contains(m.statusMsg, "Error toggling status") {
		t.Errorf("expected statusMsg to contain DB error, got: %s", m.statusMsg)
	}
	_ = db.InitDB(":memory:") // restore
}

