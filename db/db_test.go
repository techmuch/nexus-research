package db

import (
	"errors"
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

	// Reinitialize
	_ = InitDB(":memory:")
}
