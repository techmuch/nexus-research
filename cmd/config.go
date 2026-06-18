package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/techmuch/nexus-research/db"
	"github.com/techmuch/nexus-research/tui"
)

var runTUI = func(dbPath string) error {
	m := tui.NewModel(dbPath)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open the interactive TUI configuration and user manager",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize the database connection using DBPath
		if err := db.InitDB(DBPath); err != nil {
			return fmt.Errorf("failed to initialize SQLite database: %w", err)
		}
		defer db.CloseDB()

		return runTUI(DBPath)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
