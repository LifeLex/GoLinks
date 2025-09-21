package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golinks/internal/config"
	"golinks/internal/domain"
	"golinks/internal/service"

	"github.com/gorilla/mux"
)

// Mock LinkService for testing
type mockLinkService struct {
	links         map[string]string
	recentQueries []domain.PopularQuery
	allKeywords   []domain.KeywordInfo
	updateError   error
	getError      error
}

func (m *mockLinkService) GetLink(ctx context.Context, word string, searchTerm string) (string, error) {
	if m.getError != nil {
		return "", m.getError
	}
	if link, exists := m.links[word]; exists {
		return link, nil
	}
	return "", service.InvalidQueryError{Message: "not found"}
}

func (m *mockLinkService) UpdateLink(ctx context.Context, req domain.LinkRequest, userID string) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.links[req.Word] = req.Link
	return nil
}

func (m *mockLinkService) GetRecentQueries(ctx context.Context) ([]domain.PopularQuery, error) {
	return m.recentQueries, nil
}

func (m *mockLinkService) GetAllKeywords(ctx context.Context) ([]domain.KeywordInfo, error) {
	return m.allKeywords, nil
}

func setupTestHandler() *Handler {
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}

	// Create simple templates for testing
	templates := template.Must(template.New("").Funcs(template.FuncMap{
		"urlify": func(url string) template.HTML {
			if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
				return template.HTML(`<a href="` + url + `">` + url + `</a>`)
			}
			return template.HTML(url)
		},
	}).Parse(`
		{{define "homepage.html"}}
		<html>
		<body>
			<h1>GoLinks</h1>
			{{if .Missing}}<div>Missing: {{.Missing}}</div>{{end}}
			{{if .Success}}<div>Success: {{.Success}}</div>{{end}}
			{{if .Failure}}<div>Failure: {{.Failure}} - {{.Reason}}</div>{{end}}
			<div>Recent Queries: {{len .RecentQueries}}</div>
			<div>All Keywords: {{len .AllKeywords}}</div>
		</body>
		</html>
		{{end}}
		{{define "setup.html"}}
		<html>
		<body>
			<h1>Setup</h1>
			<p>Base URL: {{.BaseURL}}</p>
		</body>
		</html>
		{{end}}
	`))

	mockService := &mockLinkService{
		links: map[string]string{
			"docs":   "https://docs.example.com",
			"github": "https://github.com",
		},
		recentQueries: []domain.PopularQuery{
			{Count: 5, Word: "docs", Link: "https://docs.example.com"},
		},
		allKeywords: []domain.KeywordInfo{
			{Word: "docs", Link: "https://docs.example.com"},
		},
	}

	handler := &Handler{
		linkService: mockService,
		config:      cfg,
		templates:   templates,
	}

	return handler
}

func TestHandler_RedirectHandler(t *testing.T) {
	handler := setupTestHandler()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedHeader string
		setupError     error
	}{
		{
			name:           "successful redirect",
			path:           "/query/docs",
			expectedStatus: http.StatusFound,
			expectedHeader: "https://docs.example.com",
		},
		{
			name:           "missing query redirect to homepage",
			path:           "/query/nonexistent",
			expectedStatus: http.StatusFound,
			expectedHeader: "http://localhost:8080/homepage/?missing=nonexistent",
		},
		{
			name:           "empty path",
			path:           "/query/",
			expectedStatus: http.StatusFound,
			expectedHeader: "http://localhost:8080/homepage/?missing=",
		},
		{
			name:           "path with trailing slash",
			path:           "/query/docs/",
			expectedStatus: http.StatusFound,
			expectedHeader: "https://docs.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			// Setup router to extract path variable
			router := mux.NewRouter()
			router.HandleFunc("/query/{path:.*}", handler.RedirectHandler).Methods("GET")
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("RedirectHandler() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusFound {
				location := w.Header().Get("Location")
				if location != tt.expectedHeader {
					t.Errorf("RedirectHandler() Location = %v, want %v", location, tt.expectedHeader)
				}
			}
		})
	}
}

func TestHandler_UpdateLinkHandler(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		setupError     error
	}{
		{
			name: "successful update",
			requestBody: domain.LinkRequest{
				Word: "test",
				Link: "https://test.com",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: domain.LinkRequest{
				Word: "error",
				Link: "https://error.com",
			},
			expectedStatus: http.StatusBadRequest,
			setupError:     service.InvalidQueryError{Message: "test error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTestHandler()

			// Setup error if needed
			if tt.setupError != nil {
				mockService := handler.linkService.(*mockLinkService)
				mockService.updateError = tt.setupError
			}

			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/update/", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.UpdateLinkHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateLinkHandler() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if response["status"] != "success" {
					t.Errorf("Expected success response, got %v", response)
				}
			}
		})
	}
}

func TestHandler_HomepageHandler(t *testing.T) {
	handler := setupTestHandler()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedBody   []string
	}{
		{
			name:           "basic homepage",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"<h1>GoLinks</h1>", "Recent Queries: 1", "All Keywords: 1"},
		},
		{
			name:           "homepage with success message",
			queryParams:    "?success=docs",
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"Success: docs"},
		},
		{
			name:           "homepage with failure message",
			queryParams:    "?failure=test&reason=invalid",
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"Failure: test - invalid"},
		},
		{
			name:           "homepage with missing query",
			queryParams:    "?missing=nonexistent",
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"Missing: nonexistent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/homepage/"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.HomepageHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("HomepageHandler() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			body := w.Body.String()
			for _, expected := range tt.expectedBody {
				if !strings.Contains(body, expected) {
					t.Errorf("HomepageHandler() body should contain %q, got %q", expected, body)
				}
			}
		})
	}
}

func TestHandler_SetupHandler(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("GET", "/setup/", nil)
	w := httptest.NewRecorder()

	handler.SetupHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("SetupHandler() status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	expectedContent := []string{
		"<h1>Setup</h1>",
		"Base URL: http://localhost:8080",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(body, expected) {
			t.Errorf("SetupHandler() body should contain %q, got %q", expected, body)
		}
	}
}

func TestHandler_RegisterRoutes(t *testing.T) {
	handler := setupTestHandler()
	router := mux.NewRouter()

	// This should not panic
	handler.RegisterRoutes(router)

	// Test that routes are registered by making requests
	tests := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/", http.StatusFound},              // Root redirect
		{"GET", "/homepage/", http.StatusOK},        // Homepage
		{"GET", "/setup/", http.StatusOK},           // Setup
		{"GET", "/query/docs", http.StatusFound},    // Query redirect
		{"POST", "/update/", http.StatusBadRequest}, // Update (bad request due to no body)
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			var req *http.Request
			if tt.method == "POST" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(""))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("Route %s %s status = %v, want %v", tt.method, tt.path, w.Code, tt.status)
			}
		})
	}
}

func TestHandler_getUserID(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("GET", "/", nil)
	userID := handler.getUserID(req)

	// Should return default user since we don't have OAuth2 implemented
	if userID != "DefaultUser" {
		t.Errorf("getUserID() = %v, want DefaultUser", userID)
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	handler := setupTestHandler()
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Test wrong method on homepage
	req := httptest.NewRequest("POST", "/homepage/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Wrong method should return %v, got %v", http.StatusMethodNotAllowed, w.Code)
	}
}
