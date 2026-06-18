package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	frontendFS  embed.FS
	port        string
	startTime   time.Time
	frontendDir string
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
	}
}

func (s *Server) setupRouter() (*http.ServeMux, error) {
	mux := http.NewServeMux()

	// 1. API routes
	mux.HandleFunc("/api/status", s.handleStatus)

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

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	uptime := time.Since(s.startTime).Round(time.Second).String()
	
	resp := StatusResponse{
		Status:      "ok",
		Uptime:      uptime,
		Version:     "0.1.0",
		DBConnected: true, // Mocked for stand-alone mode
	}

	json.NewEncoder(w).Encode(resp)
}
