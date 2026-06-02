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
	links          map[string]string
	recentQueries  []domain.PopularQuery
	allKeywords    []domain.KeywordInfo
	searchResults  []domain.KeywordInfo
	searchError    error
	updateError    error
	getError       error
	lastSearchQ    string
	lastSearchLim  int
	lastUpdateUser string
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

func (m *mockLinkService) UpdateLink(_ context.Context, req domain.LinkRequest, userID string) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.lastUpdateUser = userID
	m.links[req.Word] = req.Link
	return nil
}

func (m *mockLinkService) GetRecentQueries(_ context.Context) ([]domain.PopularQuery, error) {
	return m.recentQueries, nil
}

func (m *mockLinkService) GetAllKeywords(_ context.Context) ([]domain.KeywordInfo, error) {
	return m.allKeywords, nil
}

func (m *mockLinkService) Search(_ context.Context, query string, limit int) ([]domain.KeywordInfo, error) {
	m.lastSearchQ = query
	m.lastSearchLim = limit
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResults, nil
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

func TestHealthCheck(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	handler.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %v, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	var resp struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
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

func TestSearchLinks(t *testing.T) {
	handler := setupTestHandler()
	mock := handler.linkService.(*mockLinkService)
	mock.searchResults = []domain.KeywordInfo{
		{Word: "docs", Link: "https://docs.example.com", Tags: []string{"infra"}},
	}

	req := httptest.NewRequest("GET", "/api/search?q=doc&limit=5", nil)
	w := httptest.NewRecorder()
	handler.SearchLinks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %v, want 200, body=%q", w.Code, w.Body.String())
	}

	var resp struct {
		Query   string               `json:"query"`
		Results []domain.KeywordInfo `json:"results"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Query != "doc" {
		t.Errorf("query echoed = %q, want %q", resp.Query, "doc")
	}
	if len(resp.Results) != 1 || resp.Results[0].Word != "docs" {
		t.Errorf("unexpected results: %#v", resp.Results)
	}
	if len(resp.Results[0].Tags) != 1 || resp.Results[0].Tags[0] != "infra" {
		t.Errorf("tags not surfaced: %#v", resp.Results[0].Tags)
	}
	if mock.lastSearchQ != "doc" || mock.lastSearchLim != 5 {
		t.Errorf("service called with q=%q limit=%d, want q=doc limit=5", mock.lastSearchQ, mock.lastSearchLim)
	}
}

func TestSearchLinks_BadLimit(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/search?q=doc&limit=abc", nil)
	w := httptest.NewRecorder()
	handler.SearchLinks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %v, want 400", w.Code)
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
	// Gating is exercised in middleware_test.go; here `authed` is an unguarded
	// subrouter so we verify only that routes register and match by method.
	authed := router.NewRoute().Subrouter()
	handler.RegisterRoutes(router, authed)

	tests := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{"GET", "/healthz", "", http.StatusOK},
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

	// Anonymous request → empty author.
	if uid := handler.getUserID(httptest.NewRequest("GET", "/", nil)); uid != "" {
		t.Errorf("getUserID(anonymous) = %q, want empty", uid)
	}

	// Authenticated request → the user's email.
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(WithUser(req.Context(), &domain.User{Email: "user@example.com"}))
	if uid := handler.getUserID(req); uid != "user@example.com" {
		t.Errorf("getUserID(authed) = %q, want user@example.com", uid)
	}
}
