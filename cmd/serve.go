package cmd

import (
	"github.com/spf13/cobra"
	"github.com/techmuch/nexus-research/db"
	"github.com/techmuch/nexus-research/server"
)

var port string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the NEXUS Research Station web server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := db.InitDB(DBPath); err != nil {
			return err
		}
		defer db.CloseDB()

		s := server.NewServer(frontendFS, port)
		return s.Start()
	},
}

func init() {
	serveCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to serve the application on")
	rootCmd.AddCommand(serveCmd)
}
