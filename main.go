package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// NewMux удобно использовать в тестах обработчиков.
func NewMux(us *URLShortener) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/shorten", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// (рекомендуется) Валидация Content-Type
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.HasPrefix(ct, "application/json") {
			http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
			return
		}

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req shortenRequest
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		// проверка "мусора" после JSON-объекта
		if dec.More() {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		var extra any
		if err := dec.Decode(&extra); err == nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		id, err := us.Shorten(req.URL)
		if err != nil {
			if errors.Is(err, ErrInvalidURL) {
				http.Error(w, "invalid url", http.StatusBadRequest)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := shortenResponse{
			ShortURL:    id,
			OriginalURL: req.URL,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// GET /{short_url}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// кроме /shorten сюда попадут все
		if r.URL.Path == "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// ожидаем ровно один сегмент: "/abc123"
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || strings.Contains(path, "/") {
			http.NotFound(w, r)
			return
		}

		orig, err := us.GetOriginal(path)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, orig, http.StatusFound)
	})

	return mux
}

func main() {
	us := NewURLShortener()
	handler := NewMux(us)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
