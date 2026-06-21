package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// RULE: Explicitly connect to the correct local repository due to Kopia 0.22.3 bug
	// where only one repository can be active at a time between CLI and KopiaUI.
	
	repoPath := os.Getenv("KOPIA_REPO_PATH")
	if repoPath == "" {
		// Fallback for local machine's backup based on user rule (dfhn-david-mac)
		repoPath = "/Users/david/backups/nexus-kopia-repo" 
	}

	log.Printf("Connecting to Kopia repository at %s...", repoPath)
	
	// Dev UX: Auto-create repo if it doesn't exist to prevent local script failure
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		log.Printf("Repository directory does not exist, initializing new Kopia repository at %s", repoPath)
		os.MkdirAll(repoPath, 0755)
		initCmd := exec.Command("kopia", "repository", "create", "filesystem", "--path", repoPath, "--password", "nexus-dev")
		if err := initCmd.Run(); err != nil {
			log.Printf("Failed to init repository: %v", err)
		}
	}

	cmdConnect := exec.Command("kopia", "repository", "connect", "filesystem", "--path", repoPath, "--password", "nexus-dev")
	cmdConnect.Stdout = os.Stdout
	cmdConnect.Stderr = os.Stderr
	if err := cmdConnect.Run(); err != nil {
		log.Fatalf("Failed to connect to kopia repository: %v", err)
	}

	// Target the enterprise SQLite database for Disaster Recovery
	dbPath := "nexus.db"
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		log.Fatalf("Failed to resolve db path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Printf("Skipping backup: Database file %s does not exist yet.", absPath)
		return
	}

	log.Printf("Creating enterprise snapshot for %s...", absPath)
	cmdSnapshot := exec.Command("kopia", "snapshot", "create", absPath)
	cmdSnapshot.Stdout = os.Stdout
	cmdSnapshot.Stderr = os.Stderr
	if err := cmdSnapshot.Run(); err != nil {
		log.Fatalf("Failed to create snapshot: %v", err)
	}

	log.Println("Database disaster recovery backup completed successfully.")
}
