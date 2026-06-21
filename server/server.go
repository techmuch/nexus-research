package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/techmuch/nexus-research/db"
)

type Server struct {
	frontendFS  embed.FS
	port        string
	startTime   time.Time
	frontendDir string
	sessions    map[string]string
	sessionsMu  sync.RWMutex
}

type StatusResponse struct {
	Status      string `json:"status"`
	Uptime      string `json:"uptime"`
	Version     string `json:"version"`
	DBConnected bool   `json:"db_connected"`
}

func NewServer(frontendFS embed.FS, port string) *Server {
	return &Server{
		frontendFS:  frontendFS,
		port:        port,
		startTime:   time.Now(),
		frontendDir: "frontend/dist",
		sessions:    make(map[string]string),
	}
}

func (s *Server) setupRouter() (*http.ServeMux, error) {
	mux := http.NewServeMux()

	// 1. API routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/auth/check", s.handleAuthCheck)
	mux.HandleFunc("/api/projects", s.handleProjects)
	mux.HandleFunc("/api/projects/share", s.handleShareProject)
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/maps/content", s.handleMapContent)

	// 2. Static and SPA routing
	// Extract the subdirectory from the embedded FS
	publicFS, err := fs.Sub(s.frontendFS, s.frontendDir)
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(publicFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If it's an API route that didn't match, return 404
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try to open the file in the embedded FS
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := publicFS.Open(path)
		if err != nil {
			// File does not exist, serve index.html (fallback for React Router SPA)
			indexBytes, err := fs.ReadFile(publicFS, "index.html")
			if err != nil {
				http.Error(w, "index.html not found in embedded frontend FS", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexBytes)
			return
		}
		f.Close()

		// Otherwise serve the file using the standard file server
		fileServer.ServeHTTP(w, r)
	})

	return mux, nil
}

func (s *Server) Start() error {
	mux, err := s.setupRouter()
	if err != nil {
		return err
	}
	log.Printf("Starting NEXUS research server on http://localhost:%s\n", s.port)
	return http.ListenAndServe(":"+s.port, s.auditMiddleware(mux))
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (s *Server) auditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggingResponseWriter{w, http.StatusOK}
		next.ServeHTTP(lrw, r)
		
		// Log to database if it's a mutating action (POST, PUT, DELETE)
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" {
			username, _ := s.isAuthorized(r)
			if username != "" && lrw.statusCode < 400 {
				action := r.Method
				resourceType := "api"
				if strings.Contains(r.URL.Path, "projects") {
					resourceType = "project"
				} else if strings.Contains(r.URL.Path, "login") {
					action = "LOGIN"
				}
				db.LogAuditAction(username, action, resourceType, r.URL.Path, "API Request")
			}
		}
	})
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthCheckResponse struct {
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
}

func (s *Server) isAuthorized(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return "", false
	}

	s.sessionsMu.RLock()
	username, ok := s.sessions[cookie.Value]
	s.sessionsMu.RUnlock()

	if !ok {
		return "", false
	}

	if db.DB != nil {
		var isDisabled bool
		err := db.DB.QueryRow("SELECT is_disabled FROM users WHERE username = ?", username).Scan(&isDisabled)
		if err != nil || isDisabled {
			return "", false
		}
	}

	return username, true
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	valid, err := db.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid username or password"})
		return
	}

	// Session token generation
	token := uuid.New().String()
	s.sessionsMu.Lock()
	s.sessions[token] = req.Username
	s.sessionsMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set true if running HTTPS
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"username": req.Username,
	})
}

func (s *Server) handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	username, ok := s.isAuthorized(r)
	w.Header().Set("Content-Type", "application/json")
	
	resp := AuthCheckResponse{
		Authenticated: ok,
		Username:      username,
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.isAuthorized(r); !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	uptime := time.Since(s.startTime).Round(time.Second).String()
	
	resp := StatusResponse{
		Status:      "ok",
		Uptime:      uptime,
		Version:     "0.1.0",
		DBConnected: true,
	}

	json.NewEncoder(w).Encode(resp)
}

type ProjectRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	username, auth := s.isAuthorized(r)
	if !auth {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == "GET" {
		projects, err := db.GetProjects(username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(projects)
		return
	}

	if r.Method == "POST" {
		var req ProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			req.ID = "proj-" + uuid.New().String()
		}
		if err := db.CreateProject(username, req.ID, req.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "id": req.ID})
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

type ShareRequest struct {
	ProjectID      string `json:"project_id"`
	TargetUsername string `json:"target_username"`
	Role           string `json:"role"`
}

func (s *Server) handleShareProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, auth := s.isAuthorized(r)
	if !auth {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req ShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Role != "owner" && req.Role != "editor" && req.Role != "viewer" {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	if err := db.ShareProject(username, req.ProjectID, req.TargetUsername, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	username, auth := s.isAuthorized(r)
	if !auth {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "GET":
		files, err := db.GetFilesTree(username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(files)

	case "POST":
		var req struct {
			ID        string  `json:"id"`
			ProjectID string  `json:"project_id"`
			ParentID  *string `json:"parent_id"`
			Name      string  `json:"name"`
			Type      string  `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			req.ID = "file-" + uuid.New().String()
		}
		if err := db.CreateFile(username, req.ID, req.ProjectID, req.ParentID, req.Name, req.Type); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "id": req.ID})

	case "PUT":
		var req struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if err := db.RenameFile(username, req.ID, req.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		if err := db.DeleteFile(username, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMapContent(w http.ResponseWriter, r *http.Request) {
	username, auth := s.isAuthorized(r)
	if !auth {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != "PUT" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if err := db.UpdateFileContent(username, req.ID, req.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
