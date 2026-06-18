package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techmuch/nexus-research/db"
)

var (
	usernameFlag string
	passwordFlag string
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage NEXUS Research Station users",
}

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize DB
		err := db.InitDB(DBPath)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer db.CloseDB()

		username := usernameFlag
		password := passwordFlag

		reader := bufio.NewReader(os.Stdin)

		if username == "" {
			cmd.Print("Enter username: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			username = strings.TrimSpace(input)
		}

		if password == "" {
			cmd.Print("Enter password: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			password = strings.TrimSpace(input)
		}

		if username == "" || password == "" {
			return fmt.Errorf("username and password cannot be empty")
		}

		err = db.CreateUser(username, password)
		if err != nil {
			return err
		}

		cmd.Printf("User '%s' successfully created.\n", username)
		return nil
	},
}

func init() {
	userCreateCmd.Flags().StringVarP(&usernameFlag, "username", "u", "", "Username for the new user")
	userCreateCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "Password for the new user")
	
	userCmd.AddCommand(userCreateCmd)
	rootCmd.AddCommand(userCmd)
}
