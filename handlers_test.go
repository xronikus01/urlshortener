package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPOST_Shorten_Handler(t *testing.T) {
	us := NewURLShortener()
	srv := httptest.NewServer(NewMux(us))
	defer srv.Close()

	t.Run("success", func(t *testing.T) {
		body := `{"url":"http://example.com/long/path"}`
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/shorten", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request err: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusOK)
		}

		var got shortenResponse
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode err: %v", err)
		}
		if got.OriginalURL != "http://example.com/long/path" {
			t.Fatalf("original=%q, want %q", got.OriginalURL, "http://example.com/long/path")
		}
		if len(got.ShortURL) < 6 || len(got.ShortURL) > 8 {
			t.Fatalf("short_url len=%d, want 6..8", len(got.ShortURL))
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		body := `{"url":` // битый JSON
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/shorten", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request err: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("invalid url", func(t *testing.T) {
		body := `{"url":"not-a-url"}`
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/shorten", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request err: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("unsupported content-type", func(t *testing.T) {
		body := `{"url":"http://example.com"}`
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/shorten", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "text/plain")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request err: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnsupportedMediaType {
			t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusUnsupportedMediaType)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/shorten")
		if err != nil {
			t.Fatalf("request err: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
		}
	})
	t.Run("extra data after json -> 400", func(t *testing.T) {
		// два JSON подряд -> после первого объекта есть лишнее
		body := `{"url":"http://example.com"}{"x":1}`
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/shorten", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request err: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

func TestGET_Redirect_Handler(t *testing.T) {
	us := NewURLShortener()
	handler := NewMux(us)

	// Сначала создаём ID через бизнес-логику
	orig := "https://example.com/abc"
	id, err := us.Shorten(orig)
	if err != nil {
		t.Fatalf("shorten err: %v", err)
	}

	t.Run("success redirect 302", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/"+id, nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusFound)
		}
		loc := rr.Header().Get("Location")
		if loc != orig {
			t.Fatalf("Location=%q, want %q", loc, orig)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/missingid", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("bad path with extra segment -> 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/a/b", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
		}
	})
	t.Run("root path -> 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("method not allowed on /{id} -> 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/"+id, nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusMethodNotAllowed)
		}
	})
}
