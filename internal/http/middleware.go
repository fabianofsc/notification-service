package http

import (
	"encoding/base64"
	"net/http"
	"strings"
)

func BasicAuth(user, pass string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				writeUnauthorized(w)
				return
			}

			const prefix = "Basic "
			if !strings.HasPrefix(auth, prefix) {
				writeUnauthorized(w)
				return
			}

			decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
			if err != nil {
				writeUnauthorized(w)
				return
			}

			creds := strings.SplitN(string(decoded), ":", 2)
			if len(creds) != 2 || creds[0] != user || creds[1] != pass {
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Basic realm="notification-service"`)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"unauthorized"}`))
}
