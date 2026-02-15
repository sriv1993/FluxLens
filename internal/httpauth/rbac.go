package httpauth

import (
	"context"
	"net/http"
	"os"
	"strings"
)

// Role names for FluxLens operator APIs.
const (
	RoleOperator = "operator"
	RoleReviewer = "reviewer"
	RoleAdmin    = "admin"
	RoleAuditor  = "auditor"
)

type ctxKey int

const rolesCtxKey ctxKey = 1

// Principal holds the authenticated API key identity and roles.
type Principal struct {
	Key   string
	Roles []string
}

// ParseKeyRoles parses "key:role1+role2,key2:admin" entries.
// Keys without a colon default to operator role.
func ParseKeyRoles(entries []string) map[string][]string {
	out := make(map[string][]string)
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		key, rolePart, ok := strings.Cut(entry, ":")
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if !ok || strings.TrimSpace(rolePart) == "" {
			out[key] = []string{RoleOperator}
			continue
		}
		var roles []string
		for _, r := range strings.Split(rolePart, "+") {
			r = strings.TrimSpace(r)
			if r != "" {
				roles = append(roles, r)
			}
		}
		if len(roles) == 0 {
			roles = []string{RoleOperator}
		}
		out[key] = roles
	}
	return out
}

// RBACMiddleware attaches Principal to the request context when API keys are configured.
// When keys is empty, all requests receive admin+operator roles (dev mode).
func RBACMiddleware(keyRoles map[string][]string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) || r.URL.Path == "/api/v1/stream" {
			next.ServeHTTP(w, r)
			return
		}
		if len(keyRoles) == 0 {
			ctx := context.WithValue(r.Context(), rolesCtxKey, Principal{Roles: []string{RoleAdmin, RoleOperator, RoleReviewer, RoleAuditor}})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		key := ExtractKey(r)
		roles, ok := keyRoles[key]
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), rolesCtxKey, Principal{Key: key, Roles: roles})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PrincipalFromContext returns the authenticated principal, if any.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(rolesCtxKey).(Principal)
	return p, ok
}

// RequireRoles returns middleware that checks the principal has at least one role.
func RequireRoles(allowed ...string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFromContext(r.Context())
			if !ok {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			if hasRole(p.Roles, RoleAdmin) {
				next.ServeHTTP(w, r)
				return
			}
			for _, have := range p.Roles {
				if _, ok := allowedSet[have]; ok {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden: insufficient role", http.StatusForbidden)
		})
	}
}

func hasRole(roles []string, want string) bool {
	for _, r := range roles {
		if r == want {
			return true
		}
	}
	return false
}

// RolesFromEnv reads FLUXLENS_API_KEY_ROLES (key:role+role,...).
func RolesFromEnv() []string {
	return SplitKeys(os.Getenv("FLUXLENS_API_KEY_ROLES"))
}
