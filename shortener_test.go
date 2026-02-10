package main

import (
	"errors"
	"testing"
)

func TestURLShortener_Shorten_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"валидный HTTP URL", "http://example.com", false},
		{"валидный HTTPS URL", "https://google.com/search?q=test", false},
		{"невалидный URL", "not-a-url", true},
		{"пустая строка", "", true},
		{"нет схемы", "example.com/path", true},
		{"не http/https", "ftp://example.com/file", true},
		{"нет host", "http:///path", true},
	}

	shortener := NewURLShortener()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			shortID, err := shortener.Shorten(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ошибка = %v, ожидали ошибку = %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if len(shortID) < 6 || len(shortID) > 8 {
					t.Fatalf("короткий ID должен быть 6-8 символов, получили: %q (len=%d)", shortID, len(shortID))
				}
			}
		})
	}
}

func TestURLShortener_Shorten_UniqueIDs(t *testing.T) {
	us := NewURLShortener()

	id1, err := us.Shorten("http://example.com/a")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	id2, err := us.Shorten("http://example.com/b")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("ожидали разные short_id, получили одинаковые: %s", id1)
	}
}

func TestURLShortener_GetOriginal(t *testing.T) {
	us := NewURLShortener()

	t.Run("not found", func(t *testing.T) {
		_, err := us.GetOriginal("missing")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("ожидали ErrNotFound, получили: %v", err)
		}
	})

	t.Run("empty id", func(t *testing.T) {
		_, err := us.GetOriginal("")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("ожидали ErrNotFound, получили: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		orig := "https://example.com/long/path"
		id, err := us.Shorten(orig)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		got, err := us.GetOriginal(id)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != orig {
			t.Fatalf("ожидали %q, получили %q", orig, got)
		}
	})
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"ok http", "http://example.com", true},
		{"ok https", "https://example.com/a?b=c", true},
		{"bad empty", "", false},
		{"bad spaces", "   ", false},
		{"bad scheme", "ftp://example.com", false},
		{"bad host", "http:///a", false},
		{"bad no scheme", "example.com/a", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidURL(tt.in); got != tt.want {
				t.Fatalf("isValidURL(%q)=%v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
