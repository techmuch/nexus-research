package db

import (
	"errors"
	"os"
	"testing"
)

func TestDatabaseFlow(t *testing.T) {
	// 1. Test InitDB in memory
	err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to initialize db in memory: %v", err)
	}
	defer CloseDB()

	// Verify DB ping succeeds
	err = DB.Ping()
	if err != nil {
		t.Errorf("expected db to ping, got error: %v", err)
	}

	// 2. Test CreateUser
	err = CreateUser("admin", "adminpassword", true)
	if err != nil {
		t.Errorf("failed to create user: %v", err)
	}

	// Test user duplicate creation returns error
	err = CreateUser("admin", "anotherpassword", false)
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}

	// Test empty credentials return error
	err = CreateUser("", "password", false)
	if err == nil {
		t.Errorf("expected error when username is empty, got nil")
	}

	err = CreateUser("user", "", false)
	if err == nil {
		t.Errorf("expected error when password is empty, got nil")
	}

	// 3. Test AuthenticateUser
	// Valid login
	valid, err := AuthenticateUser("admin", "adminpassword")
	if err != nil {
		t.Errorf("failed to authenticate: %v", err)
	}
	if !valid {
		t.Errorf("expected authentication to succeed")
	}

	// Invalid password
	valid, err = AuthenticateUser("admin", "wrongpassword")
	if err != nil {
		t.Errorf("failed to authenticate: %v", err)
	}
	if valid {
		t.Errorf("expected authentication to fail on wrong password")
	}

	// Non-existent user
	valid, err = AuthenticateUser("guest", "guestpassword")
	if err != nil {
		t.Errorf("failed to authenticate: %v", err)
	}
	if valid {
		t.Errorf("expected authentication to fail for non-existent user")
	}

	// Empty credentials
	valid, err = AuthenticateUser("", "password")
	if err != nil {
		t.Errorf("failed to authenticate: %v", err)
	}
	if valid {
		t.Errorf("expected authentication to fail for empty username")
	}
}

func TestInitDBError(t *testing.T) {
	err := InitDB("/nonexistentdir/nexus.db")
	if err == nil {
		t.Errorf("expected error when initializing DB under a non-existent directory, got nil")
	}
}

func TestCloseDB(t *testing.T) {
	err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	
	err = CloseDB()
	if err != nil {
		t.Errorf("expected CloseDB to succeed, got %v", err)
	}

	// Calling CloseDB on already closed DB
	err = CloseDB()
	if err != nil {
		t.Errorf("expected CloseDB on nil DB to succeed, got %v", err)
	}
}

func TestListAndDeleteUsers(t *testing.T) {
	err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer CloseDB()

	// Initial empty list
	users, err := ListUsers()
	if err != nil {
		t.Errorf("failed to list users: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}

	// Create user
	_ = CreateUser("testuser", "testpassword", true)

	// List should return 1 user
	users, err = ListUsers()
	if err != nil {
		t.Errorf("failed to list users: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
	if users[0].Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", users[0].Username)
	}
	if !users[0].IsAdmin {
		t.Errorf("expected users[0].IsAdmin to be true")
	}

	// Delete user
	err = DeleteUser("testuser")
	if err != nil {
		t.Errorf("failed to delete user: %v", err)
	}

	// Try to delete non-existent user returns ErrUserNotFound
	err = DeleteUser("nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}

	// Try to delete with empty username returns error
	err = DeleteUser("")
	if err == nil {
		t.Errorf("expected error when deleting empty username, got nil")
	}

	// List should be empty again
	users, err = ListUsers()
	if err != nil {
		t.Errorf("failed to list users: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users after deletion, got %d", len(users))
	}
}

func TestDBClosedErrors(t *testing.T) {
	_ = InitDB(":memory:")
	_ = CloseDB()

	err := CreateUser("user", "pass", false)
	if err == nil {
		t.Errorf("expected error when DB is closed")
	}

	_, err = AuthenticateUser("user", "pass")
	if err == nil {
		t.Errorf("expected error when DB is closed")
	}

	_, err = ListUsers()
	if err == nil {
		t.Errorf("expected error when DB is closed")
	}

	err = DeleteUser("user")
	if err == nil {
		t.Errorf("expected error when DB is closed")
	}

	err = ChangePassword("user", "newpassword")
	if err == nil {
		t.Errorf("expected error when DB is closed")
	}

	err = RenameUser("user", "newname")
	if err == nil {
		t.Errorf("expected error when DB is closed")
	}

	err = SetDisabled("user", true)
	if err == nil {
		t.Errorf("expected error when DB is closed")
	}

	// Reinitialize
	_ = InitDB(":memory:")
}

func TestChangePassword(t *testing.T) {
	err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer CloseDB()

	// 1. Create a user
	err = CreateUser("pwduser", "oldpassword", false)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 2. Try with empty username/password
	err = ChangePassword("", "newpwd")
	if err == nil {
		t.Errorf("expected error with empty username")
	}
	err = ChangePassword("pwduser", "")
	if err == nil {
		t.Errorf("expected error with empty password")
	}

	// 3. Try with short password
	err = ChangePassword("pwduser", "123")
	if err == nil {
		t.Errorf("expected error with short password")
	}

	// 4. Change password successfully
	err = ChangePassword("pwduser", "newpassword")
	if err != nil {
		t.Errorf("failed to change password: %v", err)
	}

	// 5. Authenticate with old password (should fail)
	ok, err := AuthenticateUser("pwduser", "oldpassword")
	if err != nil {
		t.Errorf("error authenticating: %v", err)
	}
	if ok {
		t.Errorf("expected old password to be invalid")
	}

	// 6. Authenticate with new password (should succeed)
	ok, err = AuthenticateUser("pwduser", "newpassword")
	if err != nil {
		t.Errorf("error authenticating: %v", err)
	}
	if !ok {
		t.Errorf("expected new password to be valid")
	}
}

func TestRenameUser(t *testing.T) {
	err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer CloseDB()

	// 1. Create two users
	err = CreateUser("user1", "password", false)
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}
	err = CreateUser("user2", "password", false)
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	// 2. Empty inputs
	err = RenameUser("", "newname")
	if err == nil {
		t.Errorf("expected error with empty oldUsername")
	}
	err = RenameUser("user1", "")
	if err == nil {
		t.Errorf("expected error with empty newUsername")
	}

	// 3. Rename to existing name (should fail)
	err = RenameUser("user1", "user2")
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}

	// 4. Rename successfully
	err = RenameUser("user1", "user1_new")
	if err != nil {
		t.Errorf("failed to rename user: %v", err)
	}

	// 5. Verify lookup of old name fails, new name succeeds
	ok, err := AuthenticateUser("user1", "password")
	if err != nil {
		t.Errorf("error authenticating: %v", err)
	}
	if ok {
		t.Errorf("expected old username to be unauthenticatable")
	}

	ok, err = AuthenticateUser("user1_new", "password")
	if err != nil {
		t.Errorf("error authenticating: %v", err)
	}
	if !ok {
		t.Errorf("expected new username to authenticate successfully")
	}
}

func TestSetDisabledAndAuthentication(t *testing.T) {
	err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer CloseDB()

	// 1. Create user
	err = CreateUser("lockuser", "password", false)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 2. Try empty username
	err = SetDisabled("", true)
	if err == nil {
		t.Errorf("expected error with empty username")
	}

	// 3. Check status is initially active
	users, err := ListUsers()
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	found := false
	for _, u := range users {
		if u.Username == "lockuser" {
			found = true
			if u.IsDisabled {
				t.Errorf("expected user to be active initially")
			}
		}
	}
	if !found {
		t.Fatalf("user not found in list")
	}

	// 4. Disable user
	err = SetDisabled("lockuser", true)
	if err != nil {
		t.Errorf("failed to disable user: %v", err)
	}

	// 5. Check status in list
	users, err = ListUsers()
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	for _, u := range users {
		if u.Username == "lockuser" {
			if !u.IsDisabled {
				t.Errorf("expected user to be disabled")
			}
		}
	}

	// 6. Try authenticating disabled user (should fail)
	ok, err := AuthenticateUser("lockuser", "password")
	if err != nil {
		t.Errorf("error authenticating: %v", err)
	}
	if ok {
		t.Errorf("expected disabled user authentication to fail")
	}

	// 7. Enable user
	err = SetDisabled("lockuser", false)
	if err != nil {
		t.Errorf("failed to enable user: %v", err)
	}

	// 8. Try authenticating enabled user (should succeed)
	ok, err = AuthenticateUser("lockuser", "password")
	if err != nil {
		t.Errorf("error authenticating: %v", err)
	}
	if !ok {
		t.Errorf("expected enabled user authentication to succeed")
	}
}

func TestMigrations(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_nexus_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	// InitDB first time (creates schema and table)
	err = InitDB(tempPath)
	if err != nil {
		t.Fatalf("failed to init DB first time: %v", err)
	}
	
	err = CreateUser("migrateduser", "password", true)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	
	err = CloseDB()
	if err != nil {
		t.Fatalf("failed to close DB: %v", err)
	}

	// InitDB second time (runs migration ALTER TABLE commands on already existing columns)
	err = InitDB(tempPath)
	if err != nil {
		t.Fatalf("failed to init DB second time: %v", err)
	}
	defer CloseDB()

	// Verify user still exists and can authenticate
	ok, err := AuthenticateUser("migrateduser", "password")
	if err != nil {
		t.Errorf("failed to authenticate: %v", err)
	}
	if !ok {
		t.Errorf("expected authenticated to succeed after migrating")
	}
}

