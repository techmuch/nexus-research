package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/techmuch/nexus-research/db"
)

var backupDir = "backups"

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage Nexus Research Station database backups",
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new database backup snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := db.InitDB(DBPath); err != nil {
			return err
		}
		defer db.CloseDB()

		fmt.Printf("Creating backup of %s...\n", DBPath)
		file, err := db.CreateBackup(DBPath, backupDir)
		if err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
		fmt.Printf("Backup successfully created at: %s\n", file)
		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all existing database backups",
	RunE: func(cmd *cobra.Command, args []string) error {
		backups, err := db.ListBackups(backupDir)
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}
		if len(backups) == 0 {
			fmt.Println("No backups found.")
			return nil
		}
		fmt.Printf("%-30s | %-15s | %s\n", "Timestamp", "Size (Bytes)", "File Path")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, b := range backups {
			fmt.Printf("%-30s | %-15d | %s\n", b.Timestamp.Format("2006-01-02 15:04:05"), b.Size, b.Path)
		}
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [backup_file]",
	Short: "Restore a database backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backupFile := args[0]
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			return fmt.Errorf("backup file %s does not exist", backupFile)
		}
		
		fmt.Printf("Restoring backup from %s to %s...\n", backupFile, DBPath)
		// Init DB might not be strictly needed for Postgres pg_restore, but we do it to match structure
		if err := db.InitDB(DBPath); err != nil {
			return err
		}
		defer db.CloseDB()

		if err := db.RestoreBackup(DBPath, backupFile); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}
		fmt.Println("Restore completed successfully.")
		return nil
	},
}

var backupPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune old backups according to the retention policy",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := db.InitDB(DBPath); err != nil {
			return err
		}
		defer db.CloseDB()

		config, err := db.GetBackupConfig()
		if err != nil {
			return fmt.Errorf("failed to get backup config: %w", err)
		}

		fmt.Println("Pruning backups based on retention policy...")
		if err := db.PruneBackups(backupDir, config); err != nil {
			return fmt.Errorf("prune failed: %w", err)
		}
		fmt.Println("Prune completed successfully.")
		return nil
	},
}

func init() {
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupPruneCmd)
	rootCmd.AddCommand(backupCmd)
}
