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
	return http.ListenAndServe(":"+s.port, mux)
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
