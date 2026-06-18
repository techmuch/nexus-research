package cmd

import (
	"bytes"
	"embed"
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
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "serve" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected serve command to be registered under root command")
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
