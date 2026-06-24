package server

import (
	"log"
	"time"

	"github.com/techmuch/nexus-research/db"
)

// StartBackupScheduler runs a background goroutine to perform automated backups based on DB config.
func StartBackupScheduler(dbPath string, backupDir string) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			config, err := db.GetBackupConfig()
			if err != nil {
				log.Printf("Scheduler error fetching backup config: %v", err)
				continue
			}

			if !config.Enabled {
				continue
			}

			log.Println("Starting automated background backup...")
			_, err = db.CreateBackup(dbPath, backupDir)
			if err != nil {
				log.Printf("Automated backup failed: %v", err)
				continue
			}

			err = db.PruneBackups(backupDir, config)
			if err != nil {
				log.Printf("Automated backup pruning failed: %v", err)
			} else {
				log.Println("Automated backup & pruning completed successfully.")
			}
		}
	}()
}
