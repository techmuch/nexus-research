package db

import (
	"database/sql"
	"errors"
)

type ServerConfig struct {
	Host string
	Port string
}

// GetServerConfig retrieves the server config from the database.
// If the table or row does not exist, it creates it and inserts default values.
func GetServerConfig() (ServerConfig, error) {
	cfg := ServerConfig{
		Host: "0.0.0.0",
		Port: "8080",
	}

	if DB == nil {
		return cfg, nil
	}

	// Create table if not exists
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS server_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			host TEXT NOT NULL DEFAULT '0.0.0.0',
			port TEXT NOT NULL DEFAULT '8080'
		)
	`)
	if err != nil {
		return cfg, err
	}

	// Insert default row if empty
	_, _ = DB.Exec(`INSERT OR IGNORE INTO server_config (id, host, port) VALUES (1, '0.0.0.0', '8080')`)

	err = DB.QueryRow("SELECT host, port FROM server_config WHERE id = 1").Scan(&cfg.Host, &cfg.Port)
	if err != nil && err != sql.ErrNoRows {
		return cfg, err
	}

	return cfg, nil
}

// SaveServerConfig saves the server config to the database.
func SaveServerConfig(cfg ServerConfig) error {
	if DB == nil {
		return errors.New("database not initialized")
	}

	// Ensure table exists
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS server_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			host TEXT NOT NULL DEFAULT '0.0.0.0',
			port TEXT NOT NULL DEFAULT '8080'
		)
	`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		INSERT INTO server_config (id, host, port) 
		VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET host = excluded.host, port = excluded.port
	`, cfg.Host, cfg.Port)

	return err
}
