package main

import (
	"net/http"
	"os"
)

// spaFS wraps an http.FileSystem so that requests for missing files fall back
// to index.html, which is required for client-side routing in a React SPA.
// Directories are left alone so http.FileServer can look up index.html inside
// them naturally (converting them here caused an infinite redirect on "/").
type spaFS struct{ http.FileSystem }

func (s spaFS) Open(name string) (http.File, error) {
	f, err := s.FileSystem.Open(name)
	if os.IsNotExist(err) {
		return s.FileSystem.Open("/index.html")
	}
	return f, err
}

func corsMiddleware(frontendURL, environment string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if environment == "production" {
			w.Header().Set("Access-Control-Allow-Origin", frontendURL)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
