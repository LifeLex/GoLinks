package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golinks/internal/core/links"
	"golinks/internal/platform/config"
	"golinks/internal/platform/logger"

	"github.com/gorilla/mux"
)

type mockLinkService struct {
	links         map[string]string
	recentQueries []links.PopularQuery
	allKeywords   []links.KeywordInfo
	updateError   error
	getError      error
}

func (m *mockLinkService) GetLink(_ context.Context, word, _ string) (string, error) {
	if m.getError != nil {
		return "", m.getError
	}
	if link, exists := m.links[word]; exists {
		return link, nil
	}
	return "", links.InvalidQueryError{Message: "not found"}
}

func (m *mockLinkService) UpdateLink(_ context.Context, req links.LinkRequest, _ string) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.links[req.Word] = req.Link
	return nil
}

func (m *mockLinkService) GetRecentQueries(_ context.Context) ([]links.PopularQuery, error) {
	return m.recentQueries, nil
}

func (m *mockLinkService) GetAllKeywords(_ context.Context) ([]links.KeywordInfo, error) {
	return m.allKeywords, nil
}

func setupTestHandler() *LinksHandler {
	return NewLinksHandler(
		&mockLinkService{
			links: map[string]string{
				"docs":   "https://docs.example.com",
				"github": "https://github.com",
			},
			recentQueries: []links.PopularQuery{
				{Count: 5, Word: "docs", Link: "https://docs.example.com"},
			},
			allKeywords: []links.KeywordInfo{
				{Word: "docs", Link: "https://docs.example.com"},
			},
		},
		&config.Config{BaseURL: "http://localhost:8080"},
		logger.Default(),
	)
}

func TestRedirect(t *testing.T) {
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
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc("/query/{path:.*}", handler.Redirect).Methods(http.MethodGet)
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

func TestCreateJSON(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
		setupError     error
	}{
		{"valid JSON body", `{"word":"test","link":"https://test.com"}`, http.StatusOK, nil},
		{"malformed JSON", "not json", http.StatusBadRequest, nil},
		{"service rejects input", `{"word":"bad","link":"https://x.com"}`, http.StatusBadRequest, links.InvalidQueryError{Message: "bad input"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTestHandler()
			if tt.setupError != nil {
				handler.service.(*mockLinkService).updateError = tt.setupError
			}

			req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Create(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %v, want %v, body=%q", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

func TestList(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %v, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	var resp struct {
		Keywords      []links.KeywordInfo  `json:"keywords"`
		RecentQueries []links.PopularQuery `json:"recent_queries"`
		BaseURL       string               `json:"base_url"`
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

func TestUpdateLegacyForm(t *testing.T) {
	handler := setupTestHandler()

	form := strings.NewReader("word=leg&link=https://legacy.example.com")
	req := httptest.NewRequest(http.MethodPost, "/update/", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.UpdateLegacy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %v, want 200, body=%q", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Link added successfully") {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

func TestRegister(t *testing.T) {
	handler := setupTestHandler()
	router := mux.NewRouter()
	handler.Register(router)

	tests := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{http.MethodGet, "/api/links", "", http.StatusOK},
		{http.MethodPost, "/api/links", `{"word":"x","link":"https://x.com"}`, http.StatusOK},
		{http.MethodGet, "/query/docs", "", http.StatusFound},
		{http.MethodPost, "/update/", "word=x&link=https://x.com", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.method == http.MethodPost && tt.path == "/api/links" {
				req.Header.Set("Content-Type", "application/json")
			} else if tt.method == http.MethodPost {
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

func TestUserIDFromRequest(t *testing.T) {
	if uid := userIDFromRequest(httptest.NewRequest(http.MethodGet, "/", nil)); uid != "DefaultUser" {
		t.Errorf("userIDFromRequest() = %v, want DefaultUser", uid)
	}
}
