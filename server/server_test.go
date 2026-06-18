package server

import (
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//go:embed all:frontend/dist
var testFrontendFS embed.FS

//go:embed all:frontend_no_index/dist
var testFrontendNoIndexFS embed.FS

func TestNewServer(t *testing.T) {
	s := NewServer(testFrontendFS, "9090")
	if s.port != "9090" {
		t.Errorf("Expected port to be 9090, got %s", s.port)
	}
}

func TestHandleStatus(t *testing.T) {
	s := NewServer(testFrontendFS, "9090")
	req := httptest.NewRequest("GET", "/api/status", nil)
	rr := httptest.NewRecorder()

	s.handleStatus(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
	}

	var resp StatusResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}
	if resp.Version != "0.1.0" {
		t.Errorf("expected version '0.1.0', got '%s'", resp.Version)
	}
	if !resp.DBConnected {
		t.Errorf("expected db_connected to be true")
	}
	if resp.Uptime == "" {
		t.Errorf("expected non-empty uptime")
	}
}

func TestSetupRouter(t *testing.T) {
	s := NewServer(testFrontendFS, "9090")
	router, err := s.setupRouter()
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
		expectedBody   string
		checkBody      bool
	}{
		{
			name:           "API status endpoint",
			method:         "GET",
			url:            "/api/status",
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},
		{
			name:           "API invalid path returns 404",
			method:         "GET",
			url:            "/api/non-existent",
			expectedStatus: http.StatusNotFound,
			checkBody:      false,
		},
		{
			name:           "Root path serves index.html",
			method:         "GET",
			url:            "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "NEXUS Research Station Mock UI",
			checkBody:      true,
		},
		{
			name:           "Vite asset path serves asset",
			method:         "GET",
			url:            "/assets/test.css",
			expectedStatus: http.StatusOK,
			expectedBody:   "background-color: #000;",
			checkBody:      true,
		},
		{
			name:           "SPA fallback serves index.html",
			method:         "GET",
			url:            "/dashboard/reports",
			expectedStatus: http.StatusOK,
			expectedBody:   "NEXUS Research Station Mock UI",
			checkBody:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkBody {
				bodyStr := rr.Body.String()
				if !strings.Contains(bodyStr, tt.expectedBody) {
					t.Errorf("expected body to contain '%s', got '%s'", tt.expectedBody, bodyStr)
				}
			}
		})
	}
}

func TestSetupRouterMissingIndex(t *testing.T) {
	s := NewServer(testFrontendNoIndexFS, "9090")
	router, err := s.setupRouter()
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest("GET", "/non-existent", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 Internal Server Error when index.html is missing, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "index.html not found") {
		t.Errorf("expected error message to complain about index.html, got: %s", rr.Body.String())
	}
}

func TestServerStartError(t *testing.T) {
	s := NewServer(testFrontendFS, "-1")
	err := s.Start()
	if err == nil {
		t.Errorf("expected error when starting server on invalid port, got nil")
	}
}

func TestSetupRouterError(t *testing.T) {
	s := NewServer(testFrontendFS, "9090")
	s.frontendDir = "invalid/../path"
	_, err := s.setupRouter()
	if err == nil {
		t.Errorf("expected error when setting up router with invalid frontendDir, got nil")
	}
}

func TestServerStartErrorInvalidDir(t *testing.T) {
	s := NewServer(testFrontendFS, "9090")
	s.frontendDir = "invalid/../path"
	err := s.Start()
	if err == nil {
		t.Errorf("expected error when starting server with invalid frontendDir, got nil")
	}
}
