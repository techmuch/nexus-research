package cmd

import (
	"github.com/spf13/cobra"
	"github.com/techmuch/nexus-research/db"
	"github.com/techmuch/nexus-research/server"
)

var (
	host string
	port string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the NEXUS Research Station web server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := db.InitDB(DBPath); err != nil {
			return err
		}
		defer db.CloseDB()

		server.StartBackupScheduler(DBPath, "backups")

		// Load config from database
		cfg, err := db.GetServerConfig()
		if err != nil {
			cfg = db.ServerConfig{
				Host: "0.0.0.0",
				Port: "8080",
			}
		}

		finalHost := cfg.Host
		finalPort := cfg.Port

		// Override with command-line flags if they were explicitly changed
		if cmd.Flags().Changed("host") {
			finalHost = host
		}
		if cmd.Flags().Changed("port") {
			finalPort = port
		}

		s := server.NewServer(frontendFS, finalHost, finalPort)
		return s.Start()
	},
}

func init() {
	serveCmd.Flags().StringVarP(&host, "host", "H", "0.0.0.0", "Host/interface to bind the server to")
	serveCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to serve the application on")
	rootCmd.AddCommand(serveCmd)
}

