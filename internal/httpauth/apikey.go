package httpauth

import (
	"net/http"
	"os"
	"strings"
)

// APIKeyMiddleware requires Authorization: Bearer <key> or X-API-Key when
// keys is non-empty. Health, metrics, and OpenAPI paths are always public.
func APIKeyMiddleware(keys []string, next http.Handler) http.Handler {
	if len(keys) == 0 {
		return next
	}
	allowed := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			allowed[k] = struct{}{}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		key := ExtractKey(r)
		if _, ok := allowed[key]; !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isPublicPath(path string) bool {
	switch path {
	case "/api/v1/health", "/metrics", "/api/openapi.yaml", "/api/v1/stream":
		return true
	default:
		return false
	}
}

// ExtractKey returns the API key from the request headers.
func ExtractKey(r *http.Request) string {
	if h := r.Header.Get("X-API-Key"); h != "" {
		return strings.TrimSpace(h)
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

// KeysFromEnv splits FLUXLENS_API_KEYS on commas.
func KeysFromEnv() []string {
	return SplitKeys(os.Getenv("FLUXLENS_API_KEYS"))
}

// KeysToSet builds a lookup set for WebSocket auth.
func KeysToSet(keys []string) map[string]struct{} {
	if len(keys) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			m[k] = struct{}{}
		}
	}
	return m
}

// SplitKeys parses a comma-separated key list.
func SplitKeys(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
