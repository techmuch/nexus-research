package db

import (
	"database/sql"
	"errors"
	"time"

	_ "github.com/glebarez/go-sqlite" // pure Go SQLite driver
	"golang.org/x/crypto/bcrypt"
)

var (
	DB                  *sql.DB
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

type User struct {
	ID           int
	Username     string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
}

func InitDB(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	// Migration: Add is_admin column if it doesn't exist. Ignore error if column already exists.
	_, _ = DB.Exec("ALTER TABLE users ADD COLUMN is_admin BOOLEAN DEFAULT 0")

	// Create tables
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT 0,
		created_at DATETIME NOT NULL
	);`
	
	_, err = DB.Exec(query)
	return err
}

func CloseDB() error {
	if DB != nil {
		err := DB.Close()
		DB = nil
		return err
	}
	return nil
}

func CreateUser(username, password string, isAdmin bool) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	if username == "" || password == "" {
		return errors.New("username and password cannot be empty")
	}

	// Check if user already exists
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrUserAlreadyExists
	}

	// Hash password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = DB.Exec(
		"INSERT INTO users (username, password_hash, is_admin, created_at) VALUES (?, ?, ?, ?)",
		username,
		string(hashedBytes),
		isAdmin,
		time.Now(),
	)
	return err
}

func AuthenticateUser(username, password string) (bool, error) {
	if DB == nil {
		return false, errors.New("database not initialized")
	}
	if username == "" || password == "" {
		return false, nil
	}

	var hash string
	err := DB.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // user not found, return false without database error
		}
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func ListUsers() ([]User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	rows, err := DB.Query("SELECT id, username, password_hash, is_admin, created_at FROM users ORDER BY username ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

func DeleteUser(username string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	if username == "" {
		return errors.New("username cannot be empty")
	}

	result, err := DB.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}
