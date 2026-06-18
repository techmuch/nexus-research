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

	// Create tables
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
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

func CreateUser(username, password string) error {
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
		"INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, ?)",
		username,
		string(hashedBytes),
		time.Now(),
	)
	return err
}

func AuthenticateUser(username, password string) (bool, error) {
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
