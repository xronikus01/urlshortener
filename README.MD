# URL Shortener (Go)

HTTP-сервис для сокращения URL.

## Endpoints

- `POST /shorten`  
  Request: `{"url":"https://example.com/long/path"}`  
  Response: `{"short_url":"abc123","original_url":"https://example.com/long/path"}`

- `GET /{short_url}`  
  302 redirect на оригинальный URL, либо 404 если не найдено.

## Run

```powershell
go run .
```

Tests
```powershell
go test -v ./...
go test -cover ./...
```
Coverage: 84.8% of statements