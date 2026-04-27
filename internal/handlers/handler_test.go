package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golinks/internal/config"
	"golinks/internal/domain"
	"golinks/internal/logger"
	"golinks/internal/service"

	"github.com/gorilla/mux"
)

type mockLinkService struct {
	links         map[string]string
	recentQueries []domain.PopularQuery
	allKeywords   []domain.KeywordInfo
	updateError   error
	getError      error
}

func (m *mockLinkService) GetLink(_ context.Context, word string, _ string) (string, error) {
	if m.getError != nil {
		return "", m.getError
	}
	if link, exists := m.links[word]; exists {
		return link, nil
	}
	return "", service.InvalidQueryError{Message: "not found"}
}

func (m *mockLinkService) UpdateLink(_ context.Context, req domain.LinkRequest, _ string) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.links[req.Word] = req.Link
	return nil
}

func (m *mockLinkService) GetRecentQueries(_ context.Context) ([]domain.PopularQuery, error) {
	return m.recentQueries, nil
}

func (m *mockLinkService) GetAllKeywords(_ context.Context) ([]domain.KeywordInfo, error) {
	return m.allKeywords, nil
}

func setupTestHandler() *Handler {
	return &Handler{
		linkService: &mockLinkService{
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
		},
		config: &config.Config{BaseURL: "http://localhost:8080"},
		logger: logger.Default(),
	}
}

func TestRedirectHandler(t *testing.T) {
	handler := setupTestHandler()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedHeader string
	}{
		{"hit", "/query/docs", http.StatusFound, "https://docs.example.com"},
		{"miss", "/query/nonexistent", http.StatusFound, "http://localhost:8080/?missing=nonexistent"},
		{"empty", "/query/", http.StatusFound, "http://localhost:8080/?missing="},
		{"trailing slash", "/query/docs/", http.StatusFound, "https://docs.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc("/query/{path:.*}", handler.RedirectHandler).Methods("GET")
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %v, want %v", w.Code, tt.expectedStatus)
			}
			if w.Header().Get("Location") != tt.expectedHeader {
				t.Errorf("Location = %q, want %q", w.Header().Get("Location"), tt.expectedHeader)
			}
		})
	}
}

func TestCreateLinkJSON(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
		setupError     error
	}{
		{
			name:           "valid JSON body",
			body:           `{"word":"test","link":"https://test.com"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "malformed JSON",
			body:           "not json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service rejects input",
			body:           `{"word":"bad","link":"https://x.com"}`,
			expectedStatus: http.StatusBadRequest,
			setupError:     service.InvalidQueryError{Message: "bad input"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTestHandler()
			if tt.setupError != nil {
				handler.linkService.(*mockLinkService).updateError = tt.setupError
			}

			req := httptest.NewRequest("POST", "/api/links", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CreateLink(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %v, want %v, body=%q", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

func TestListLinks(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/links", nil)
	w := httptest.NewRecorder()
	handler.ListLinks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %v, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	var resp struct {
		Keywords      []domain.KeywordInfo  `json:"keywords"`
		RecentQueries []domain.PopularQuery `json:"recent_queries"`
		BaseURL       string                `json:"base_url"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Keywords) != 1 || resp.Keywords[0].Word != "docs" {
		t.Errorf("unexpected keywords: %#v", resp.Keywords)
	}
	if len(resp.RecentQueries) != 1 {
		t.Errorf("unexpected recent queries: %#v", resp.RecentQueries)
	}
	if resp.BaseURL != "http://localhost:8080" {
		t.Errorf("base_url = %q", resp.BaseURL)
	}
}

func TestUpdateLinkLegacyForm(t *testing.T) {
	handler := setupTestHandler()

	form := strings.NewReader("word=leg&link=https://legacy.example.com")
	req := httptest.NewRequest("POST", "/update/", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.UpdateLinkLegacy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %v, want 200, body=%q", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Link added successfully") {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

func TestRegisterRoutes(t *testing.T) {
	handler := setupTestHandler()
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{"GET", "/api/links", "", http.StatusOK},
		{"POST", "/api/links", `{"word":"x","link":"https://x.com"}`, http.StatusOK},
		{"GET", "/query/docs", "", http.StatusFound},
		{"POST", "/update/", "word=x&link=https://x.com", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.method == "POST" && tt.path == "/api/links" {
				req.Header.Set("Content-Type", "application/json")
			} else if tt.method == "POST" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tt.status {
				t.Errorf("status = %v, want %v, body=%q", w.Code, tt.status, w.Body.String())
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	handler := setupTestHandler()
	if uid := handler.getUserID(httptest.NewRequest("GET", "/", nil)); uid != "DefaultUser" {
		t.Errorf("getUserID() = %v, want DefaultUser", uid)
	}
}
