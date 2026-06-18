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
	err = CreateUser("admin", "adminpassword")
	if err != nil {
		t.Errorf("failed to create user: %v", err)
	}

	// Test user duplicate creation returns error
	err = CreateUser("admin", "anotherpassword")
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}

	// Test empty credentials return error
	err = CreateUser("", "password")
	if err == nil {
		t.Errorf("expected error when username is empty, got nil")
	}

	err = CreateUser("user", "")
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
