package middleware

import (
	"net/http"
	"strings"
)

// TrimPrefix is a http middleware trimming cluster names from localhost url paths
func TrimPrefix(prefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			next.ServeHTTP(w, r)
		})
	}
}
