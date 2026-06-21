package db

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite" // pure Go SQLite driver used by golang-migrate
	"github.com/techmuch/nexus-research/db/migrations"
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
	IsDisabled   bool
	CreatedAt    time.Time
}

func InitDB(dbPath string) error {
	if dbPath != ":memory:" && dbPath != "" {
		dir := filepath.Dir(dbPath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
	}

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	// Run migrations
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	dbDriver, err := sqlite.WithInstance(DB, &sqlite.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", dbDriver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
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
	username = strings.TrimSpace(username)
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
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return false, nil
	}

	var hash string
	var isDisabled bool
	err := DB.QueryRow("SELECT password_hash, is_disabled FROM users WHERE username = ?", username).Scan(&hash, &isDisabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // user not found, return false without database error
		}
		return false, err
	}

	if isDisabled {
		return false, nil // user is disabled, reject login
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
	rows, err := DB.Query("SELECT id, username, password_hash, is_admin, is_disabled, created_at FROM users ORDER BY username ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.IsDisabled, &u.CreatedAt)
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
	username = strings.TrimSpace(username)
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

func ChangePassword(username, newPassword string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	username = strings.TrimSpace(username)
	if username == "" || newPassword == "" {
		return errors.New("username and password cannot be empty")
	}
	if len(newPassword) < 4 {
		return errors.New("password must be at least 4 characters")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = DB.Exec("UPDATE users SET password_hash = ? WHERE username = ?", string(hashedBytes), username)
	return err
}

func RenameUser(oldUsername, newUsername string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	oldUsername = strings.TrimSpace(oldUsername)
	newUsername = strings.TrimSpace(newUsername)
	if oldUsername == "" || newUsername == "" {
		return errors.New("username cannot be empty")
	}

	// Check if newUsername already exists
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", newUsername).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrUserAlreadyExists
	}

	_, err = DB.Exec("UPDATE users SET username = ? WHERE username = ?", newUsername, oldUsername)
	return err
}

func SetDisabled(username string, disabled bool) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username cannot be empty")
	}

	_, err := DB.Exec("UPDATE users SET is_disabled = ? WHERE username = ?", disabled, username)
	return err
}

// LogAuditAction records a system action to the audit_logs table for compliance
func LogAuditAction(username string, action, resourceType, resourceID, details string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}

	var userID *int
	if username != "" {
		var id int
		err := DB.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&id)
		if err == nil {
			userID = &id
		}
	}

	_, err := DB.Exec(`INSERT INTO audit_logs (user_id, action, resource_type, resource_id, details) 
		VALUES (?, ?, ?, ?, ?)`, userID, action, resourceType, resourceID, details)
	return err
}

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

func CreateProject(username, projectID, name string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}

	var userID int
	err := DB.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		return err
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("INSERT INTO projects (id, name) VALUES (?, ?)", projectID, name); err != nil {
		return err
	}

	if _, err := tx.Exec("INSERT INTO project_users (project_id, user_id, role) VALUES (?, ?, ?)", projectID, userID, "owner"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	_ = LogAuditAction(username, "CREATE", "project", projectID, "Created new project: "+name)
	return nil
}

func GetProjects(username string) ([]Project, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}

	var userID int
	if err := DB.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID); err != nil {
		return nil, err
	}

	rows, err := DB.Query(`
		SELECT p.id, p.name, pu.role, p.created_at
		FROM projects p
		JOIN project_users pu ON p.id = pu.project_id
		WHERE pu.user_id = ?
		ORDER BY p.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Role, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func ShareProject(ownerUsername, projectID, targetUsername, role string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}

	var ownerID int
	if err := DB.QueryRow("SELECT id FROM users WHERE username = ?", ownerUsername).Scan(&ownerID); err != nil {
		return err
	}

	var hasAccess bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM project_users WHERE project_id = ? AND user_id = ? AND role = 'owner')", projectID, ownerID).Scan(&hasAccess)
	if err != nil || !hasAccess {
		return errors.New("unauthorized or project does not exist")
	}

	var targetUserID int
	err = DB.QueryRow("SELECT id FROM users WHERE username = ?", targetUsername).Scan(&targetUserID)
	if err != nil {
		return errors.New("target user not found")
	}

	_, err = DB.Exec("INSERT INTO project_users (project_id, user_id, role) VALUES (?, ?, ?) ON CONFLICT(project_id, user_id) DO UPDATE SET role = ?", projectID, targetUserID, role, role)
	if err == nil {
		_ = LogAuditAction(ownerUsername, "SHARE", "project", projectID, "Shared with "+targetUsername+" as "+role)
	}
	return err
}

// File Node struct matching the frontend ITreeNode interface + content
type FileNode struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"project_id"`
	ParentID  *string `json:"parent_id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Content   string  `json:"content"`
}

func CreateFile(username, id, projectID string, parentID *string, name, fileType string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}

	// Basic authorization check
	var hasAccess bool
	err := DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM project_users pu 
			JOIN users u ON u.id = pu.user_id 
			WHERE pu.project_id = ? AND u.username = ?
		)`, projectID, username).Scan(&hasAccess)

	if err != nil || !hasAccess {
		return errors.New("unauthorized to create file in this project")
	}

	_, err = DB.Exec(`INSERT INTO files (id, project_id, parent_id, name, type) VALUES (?, ?, ?, ?, ?)`,
		id, projectID, parentID, name, fileType)
	
	if err == nil {
		_ = LogAuditAction(username, "CREATE", "file", id, "Created "+fileType+": "+name)
	}
	return err
}

func UpdateFileContent(username, id, content string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	_, err := DB.Exec("UPDATE files SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", content, id)
	return err
}

func RenameFile(username, id, newName string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	_, err := DB.Exec("UPDATE files SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", newName, id)
	return err
}

func DeleteFile(username, id string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	_, err := DB.Exec("DELETE FROM files WHERE id = ?", id)
	return err
}

func GetFilesTree(username string) ([]FileNode, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := DB.Query(`
		SELECT f.id, f.project_id, f.parent_id, f.name, f.type, IFNULL(f.content, '')
		FROM files f
		JOIN project_users pu ON f.project_id = pu.project_id
		JOIN users u ON u.id = pu.user_id
		WHERE u.username = ?`, username)
		
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileNode
	for rows.Next() {
		var f FileNode
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.ParentID, &f.Name, &f.Type, &f.Content); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}
