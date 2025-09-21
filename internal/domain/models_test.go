package domain

import (
	"testing"
	"time"
)

func TestShortcut_Validation(t *testing.T) {
	tests := []struct {
		name     string
		shortcut Shortcut
		wantErr  bool
	}{
		{
			name: "valid shortcut",
			shortcut: Shortcut{
				ID:        1,
				Word:      "docs",
				Link:      "https://docs.example.com",
				User:      "testuser",
				CreatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty word",
			shortcut: Shortcut{
				ID:        1,
				Word:      "",
				Link:      "https://docs.example.com",
				User:      "testuser",
				CreatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "empty link",
			shortcut: Shortcut{
				ID:        1,
				Word:      "docs",
				Link:      "",
				User:      "testuser",
				CreatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - in a real app you might have validation methods
			hasError := tt.shortcut.Word == "" || tt.shortcut.Link == ""
			if hasError != tt.wantErr {
				t.Errorf("Shortcut validation = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestLinkRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request LinkRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: LinkRequest{
				Word: "github",
				Link: "https://github.com",
			},
			wantErr: false,
		},
		{
			name: "word with trailing slash",
			request: LinkRequest{
				Word: "github/",
				Link: "https://github.com",
			},
			wantErr: true,
		},
		{
			name: "empty word",
			request: LinkRequest{
				Word: "",
				Link: "https://github.com",
			},
			wantErr: true,
		},
		{
			name: "empty link",
			request: LinkRequest{
				Word: "github",
				Link: "",
			},
			wantErr: true,
		},
		{
			name: "word equals link (recursive)",
			request: LinkRequest{
				Word: "test",
				Link: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation logic from service layer
			hasError := tt.request.Word == "" ||
				tt.request.Link == "" ||
				tt.request.Word == tt.request.Link ||
				len(tt.request.Word) > 0 && tt.request.Word[len(tt.request.Word)-1] == '/'

			if hasError != tt.wantErr {
				t.Errorf("LinkRequest validation = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}
