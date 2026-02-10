package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"sync"
)

var (
	ErrInvalidURL = errors.New("invalid url")
	ErrNotFound   = errors.New("short id not found")
)

type URLShortener struct {
	urls map[string]string
	mu   sync.RWMutex
}

func NewURLShortener() *URLShortener {
	return &URLShortener{
		urls: make(map[string]string),
	}
}

// Shorten создает короткий идентификатор для URL
func (us *URLShortener) Shorten(originalURL string) (string, error) {
	if !isValidURL(originalURL) {
		return "", ErrInvalidURL
	}

	// Генерируем уникальный ID (фиксированно 8 символов, что входит в 6-8)
	// На случай коллизий пробуем несколько раз.
	const maxAttempts = 10

	us.mu.Lock()
	defer us.mu.Unlock()

	for i := 0; i < maxAttempts; i++ {
		id := generateShortID() // 8 символов
		if _, exists := us.urls[id]; exists {
			continue
		}
		us.urls[id] = originalURL
		return id, nil
	}

	// Очень маловероятно, но на всякий случай
	return "", errors.New("failed to generate unique short id")
}

// GetOriginal возвращает оригинальный URL по короткому ID
func (us *URLShortener) GetOriginal(shortID string) (string, error) {
	if strings.TrimSpace(shortID) == "" {
		return "", ErrNotFound
	}

	us.mu.RLock()
	defer us.mu.RUnlock()

	orig, ok := us.urls[shortID]
	if !ok {
		return "", ErrNotFound
	}
	return orig, nil
}

// generateShortID генерирует случайный короткий идентификатор (8 символов)
func generateShortID() string {
	// 6 байт -> base64url без паддинга => ровно 8 символов
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// isValidURL проверяет корректность URL (только http/https, с host)
func isValidURL(str string) bool {
	s := strings.TrimSpace(str)
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	return true
}
