package cmd

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	frontendFS embed.FS
	DBPath     string
	rootCmd    = &cobra.Command{
		Use:   "nexus-research",
		Short: "NEXUS Research Station CLI",
		Long:  `NEXUS Research Station is an autonomous multi-agent orchestration and analysis workbench.`,
	}
)

func SetFrontendFS(fs embed.FS) {
	frontendFS = fs
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&DBPath, "db", "nexus.db", "Path to SQLite database file")
}
