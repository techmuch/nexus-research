package server

import (
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/techmuch/nexus-research/db"
)

func TestMain(m *testing.M) {
	_ = db.InitDB(":memory:")
	_ = db.CreateUser("admin", "adminpassword")
	code := m.Run()
	_ = db.CloseDB()
	os.Exit(code)
}

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

	// 1. Test unauthorized status call
	reqUnauth := httptest.NewRequest("GET", "/api/status", nil)
	rrUnauth := httptest.NewRecorder()
	s.handleStatus(rrUnauth, reqUnauth)
	if rrUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 status for unauthenticated request, got %d", rrUnauth.Code)
	}

	// 2. Test authorized status call
	s.sessions["test-token"] = "admin"
	reqAuth := httptest.NewRequest("GET", "/api/status", nil)
	reqAuth.AddCookie(&http.Cookie{Name: "session_token", Value: "test-token"})
	rrAuth := httptest.NewRecorder()

	s.handleStatus(rrAuth, reqAuth)

	if status := rrAuth.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	contentType := rrAuth.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
	}

	var resp StatusResponse
	err := json.NewDecoder(rrAuth.Body).Decode(&resp)
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
			name:           "API status endpoint (unauthorized)",
			method:         "GET",
			url:            "/api/status",
			expectedStatus: http.StatusUnauthorized,
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

func TestAuthEndpoints(t *testing.T) {
	s := NewServer(testFrontendFS, "9090")
	router, err := s.setupRouter()
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	// 1. Test GET /api/auth/check (unauthenticated)
	reqCheckUnauth := httptest.NewRequest("GET", "/api/auth/check", nil)
	rrCheckUnauth := httptest.NewRecorder()
	router.ServeHTTP(rrCheckUnauth, reqCheckUnauth)
	if rrCheckUnauth.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rrCheckUnauth.Code)
	}
	var respCheckUnauth AuthCheckResponse
	json.NewDecoder(rrCheckUnauth.Body).Decode(&respCheckUnauth)
	if respCheckUnauth.Authenticated {
		t.Errorf("expected authenticated to be false")
	}

	// 2. Test POST /api/login with invalid credentials
	badLoginPayload := `{"username":"admin","password":"wrongpassword"}`
	reqBadLogin := httptest.NewRequest("POST", "/api/login", strings.NewReader(badLoginPayload))
	rrBadLogin := httptest.NewRecorder()
	router.ServeHTTP(rrBadLogin, reqBadLogin)
	if rrBadLogin.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rrBadLogin.Code)
	}

	// 3. Test POST /api/login with valid credentials
	goodLoginPayload := `{"username":"admin","password":"adminpassword"}`
	reqGoodLogin := httptest.NewRequest("POST", "/api/login", strings.NewReader(goodLoginPayload))
	rrGoodLogin := httptest.NewRecorder()
	router.ServeHTTP(rrGoodLogin, reqGoodLogin)
	if rrGoodLogin.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rrGoodLogin.Code)
	}
	
	// Check cookie was set
	cookies := rrGoodLogin.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected session_token cookie to be set")
	}

	// 4. Test GET /api/auth/check (authenticated)
	reqCheckAuth := httptest.NewRequest("GET", "/api/auth/check", nil)
	reqCheckAuth.AddCookie(sessionCookie)
	rrCheckAuth := httptest.NewRecorder()
	router.ServeHTTP(rrCheckAuth, reqCheckAuth)
	if rrCheckAuth.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rrCheckAuth.Code)
	}
	var respCheckAuth AuthCheckResponse
	json.NewDecoder(rrCheckAuth.Body).Decode(&respCheckAuth)
	if !respCheckAuth.Authenticated {
		t.Errorf("expected authenticated to be true")
	}
	if respCheckAuth.Username != "admin" {
		t.Errorf("expected username to be 'admin', got '%s'", respCheckAuth.Username)
	}

	// 5. Test POST /api/login with invalid HTTP method
	reqInvalidMethod := httptest.NewRequest("GET", "/api/login", nil)
	rrInvalidMethod := httptest.NewRecorder()
	router.ServeHTTP(rrInvalidMethod, reqInvalidMethod)
	if rrInvalidMethod.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", rrInvalidMethod.Code)
	}

	// 6. Test POST /api/login with bad JSON
	reqBadJSON := httptest.NewRequest("POST", "/api/login", strings.NewReader("bad-json"))
	rrBadJSON := httptest.NewRecorder()
	router.ServeHTTP(rrBadJSON, reqBadJSON)
	if rrBadJSON.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rrBadJSON.Code)
	}
}
