package cmd

import (
	"bytes"
	"embed"
	"os"
	"strings"
	"testing"
)

func TestSetFrontendFS(t *testing.T) {
	var dummyFS embed.FS
	SetFrontendFS(dummyFS)
	// Just verifies that setting works and matches
	if &frontendFS == nil {
		t.Errorf("expected frontendFS to be initialized")
	}
}

func TestCommandStructure(t *testing.T) {
	// Verify command use strings
	if rootCmd.Use != "nexus-research" {
		t.Errorf("expected rootCmd use to be 'nexus-research', got '%s'", rootCmd.Use)
	}

	if serveCmd.Use != "serve" {
		t.Errorf("expected serveCmd use to be 'serve', got '%s'", serveCmd.Use)
	}

	// Verify that serveCmd is a subcommand of rootCmd
	found := false
	foundConfig := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "serve" {
			found = true
		}
		if cmd.Name() == "config" {
			foundConfig = true
		}
	}
	if !found {
		t.Errorf("expected serve command to be registered under root command")
	}
	if !foundConfig {
		t.Errorf("expected config command to be registered under root command")
	}
}

func TestServeFlags(t *testing.T) {
	flag := serveCmd.Flags().Lookup("port")
	if flag == nil {
		t.Fatalf("expected 'port' flag to be registered on serve command")
	}
	if flag.Shorthand != "p" {
		t.Errorf("expected shorthand for 'port' flag to be 'p', got '%s'", flag.Shorthand)
	}
	if flag.DefValue != "8080" {
		t.Errorf("expected default value for 'port' flag to be '8080', got '%s'", flag.DefValue)
	}
}

func TestExecuteHelp(t *testing.T) {
	// Re-direct stderr/stdout to capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	Execute()

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("NEXUS Research Station")) {
		t.Errorf("expected help output to contain 'NEXUS Research Station', got:\n%s", output)
	}
}

func TestServeCommandRun(t *testing.T) {
	rootCmd.SetArgs([]string{"serve", "--port", "-1"})
	err := rootCmd.Execute()
	if err == nil {
		t.Errorf("expected error when running serve command on port -1, got nil")
	}
}

func TestUserCreateFlags(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"user", "create", "-u", "adminflags", "-p", "passwordflags", "--db", ":memory:"})

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "User 'adminflags' successfully created.") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestUserCreateInteractive(t *testing.T) {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() {
		os.Stdin = oldStdin
	}()
	os.Stdin = r

	go func() {
		defer w.Close()
		w.Write([]byte("admininteractive\nadminpassword\n"))
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"user", "create", "-u", "", "-p", "", "--db", ":memory:"})

	err = rootCmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "User 'admininteractive' successfully created.") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestUserCreateError(t *testing.T) {
	rootCmd.SetArgs([]string{"user", "create", "-u", "test", "-p", "test", "--db", "/nonexistentdir/nexus.db"})
	err := rootCmd.Execute()
	if err == nil {
		t.Errorf("expected error when running user create with invalid DB path, got nil")
	}
}

func TestUserCreateEmptyInteractiveError(t *testing.T) {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() {
		os.Stdin = oldStdin
	}()
	os.Stdin = r

	go func() {
		defer w.Close()
		w.Write([]byte("\n\n")) // empty username and password
	}()

	rootCmd.SetArgs([]string{"user", "create", "-u", "", "-p", "", "--db", ":memory:"})
	err = rootCmd.Execute()
	if err == nil {
		t.Errorf("expected error when username and password are empty, got nil")
	}
}

func TestConfigCommand(t *testing.T) {
	oldRunTUI := runTUI
	defer func() { runTUI = oldRunTUI }()

	runTUICalled := false
	runTUI = func(dbPath string) error {
		runTUICalled = true
		if dbPath != ":memory:" {
			t.Errorf("expected dbPath to be ':memory:', got '%s'", dbPath)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"config", "--db", ":memory:"})
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !runTUICalled {
		t.Errorf("expected runTUI to be called")
	}
}

func TestConfigCommandError(t *testing.T) {
	rootCmd.SetArgs([]string{"config", "--db", "/nonexistentdir/nexus.db"})
	err := rootCmd.Execute()
	if err == nil {
		t.Errorf("expected error when running config with invalid DB path, got nil")
	}
}
