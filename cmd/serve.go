package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/techmuch/nexus-research/server"
)

var port string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the NEXUS Research Station web server",
	Run: func(cmd *cobra.Command, args []string) {
		s := server.NewServer(frontendFS, port)
		if err := s.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	},
}

func init() {
	serveCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to serve the application on")
	rootCmd.AddCommand(serveCmd)
}
