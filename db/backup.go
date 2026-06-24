package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type BackupInfo struct {
	Path      string
	Size      int64
	Timestamp time.Time
}

type BackupConfig struct {
	Enabled          bool
	HourlyRetention  int
	DailyRetention   int
	MonthlyRetention int
}

func isPostgres(dbPath string) bool {
	return strings.HasPrefix(dbPath, "postgres://") || strings.HasPrefix(dbPath, "postgresql://") || strings.HasPrefix(dbPath, "host=")
}

// CreateBackup creates a snapshot of the database in the given directory.
func CreateBackup(dbPath string, backupDir string) (string, error) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102_150405")
	var backupFile string

	if isPostgres(dbPath) {
		backupFile = filepath.Join(backupDir, fmt.Sprintf("nexus_%s.sql", timestamp))
		cmd := exec.Command("pg_dump", dbPath, "-f", backupFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("pg_dump failed: %w", err)
		}
	} else {
		backupFile = filepath.Join(backupDir, fmt.Sprintf("nexus_%s.db", timestamp))
		if DB == nil {
			return "", errors.New("sqlite database connection is nil")
		}
		// Safely copy SQLite DB without locking out concurrent writers
		_, err := DB.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupFile))
		if err != nil {
			return "", fmt.Errorf("sqlite backup failed: %w", err)
		}
	}

	return backupFile, nil
}

// ListBackups returns an inventory of backups.
func ListBackups(backupDir string) ([]BackupInfo, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "nexus_") && (strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".sql")) {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Parse timestamp: nexus_20060102_150405.db
			timeStr := strings.TrimPrefix(name, "nexus_")
			timeStr = strings.TrimSuffix(timeStr, ".db")
			timeStr = strings.TrimSuffix(timeStr, ".sql")

			t, err := time.Parse("20060102_150405", timeStr)
			if err != nil {
				continue // Skip files with invalid timestamp formats
			}

			backups = append(backups, BackupInfo{
				Path:      filepath.Join(backupDir, name),
				Size:      info.Size(),
				Timestamp: t,
			})
		}
	}

	// Sort newest to oldest
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// RestoreBackup restores a database from a backup file.
func RestoreBackup(dbPath string, backupFile string) error {
	if isPostgres(dbPath) {
		cmd := exec.Command("pg_restore", "-d", dbPath, "-c", backupFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// If it's a plain SQL file (not custom format), psql is needed instead
			if strings.HasSuffix(backupFile, ".sql") {
				cmd = exec.Command("psql", dbPath, "-f", backupFile)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("psql restore failed: %w", err)
				}
				return nil
			}
			return fmt.Errorf("pg_restore failed: %w", err)
		}
		return nil
	}

	// For SQLite
	// Close current DB to unlock file
	if DB != nil {
		DB.Close()
		DB = nil
	}

	// Overwrite file
	input, err := os.ReadFile(backupFile)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dbPath, input, 0644); err != nil {
		return err
	}

	// Re-initialize DB
	return InitDB(dbPath)
}

// PruneBackups enforces the rolling retention policy.
func PruneBackups(backupDir string, config BackupConfig) error {
	if !config.Enabled {
		return nil
	}

	backups, err := ListBackups(backupDir)
	if err != nil {
		return err
	}

	// Simple bucketing strategy:
	// We iterate from oldest to newest to decide what to keep.
	// But it's easier to bucket them and keep the newest N in each bucket.
	// For simplicity, we keep the last X hourly, last Y daily (one per day), last Z monthly.
	
	// Sort newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	var hourlyCount int
	dailyMap := make(map[string]bool)
	monthlyMap := make(map[string]bool)

	for _, b := range backups {
		keep := false
		dayStr := b.Timestamp.Format("2006-01-02")
		monthStr := b.Timestamp.Format("2006-01")

		// Hourly bucket
		if hourlyCount < config.HourlyRetention {
			hourlyCount++
			keep = true
		}

		// Daily bucket
		if !keep && len(dailyMap) < config.DailyRetention {
			if !dailyMap[dayStr] {
				dailyMap[dayStr] = true
				keep = true
			}
		} else if !keep {
			// even if we don't keep it for daily count, we might have seen this day via an hourly backup
			dailyMap[dayStr] = true
		}

		// Monthly bucket
		if !keep && len(monthlyMap) < config.MonthlyRetention {
			if !monthlyMap[monthStr] {
				monthlyMap[monthStr] = true
				keep = true
			}
		} else if !keep {
			monthlyMap[monthStr] = true
		}

		if !keep {
			os.Remove(b.Path)
		}
	}

	return nil
}

// GetBackupConfig fetches the configuration from SQLite (stubbed, ideally fetched from DB)
func GetBackupConfig() (BackupConfig, error) {
	// For now, default to disabled until TUI enables it.
	// In a real multi-tenant scenario, this might be stored in a system settings table.
	// We'll create a simple table if it doesn't exist to store this.
	config := BackupConfig{
		Enabled:          false,
		HourlyRetention:  24,
		DailyRetention:   7,
		MonthlyRetention: 12,
	}

	if DB == nil {
		return config, nil
	}

	// Create table if not exists
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS backup_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			enabled BOOLEAN NOT NULL DEFAULT 0,
			hourly INTEGER NOT NULL DEFAULT 24,
			daily INTEGER NOT NULL DEFAULT 7,
			monthly INTEGER NOT NULL DEFAULT 12
		)
	`)
	if err != nil {
		return config, err
	}

	// Insert default row if empty
	_, _ = DB.Exec(`INSERT OR IGNORE INTO backup_config (id, enabled, hourly, daily, monthly) VALUES (1, 0, 24, 7, 12)`)

	err = DB.QueryRow("SELECT enabled, hourly, daily, monthly FROM backup_config WHERE id = 1").
		Scan(&config.Enabled, &config.HourlyRetention, &config.DailyRetention, &config.MonthlyRetention)

	if err != nil && err != sql.ErrNoRows {
		return config, err
	}

	return config, nil
}

// SaveBackupConfig updates the backup settings.
func SaveBackupConfig(config BackupConfig) error {
	if DB == nil {
		return errors.New("database not initialized")
	}

	_, err := DB.Exec(`
		UPDATE backup_config 
		SET enabled = ?, hourly = ?, daily = ?, monthly = ? 
		WHERE id = 1
	`, config.Enabled, config.HourlyRetention, config.DailyRetention, config.MonthlyRetention)

	return err
}
