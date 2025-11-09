package middleware

import (
	"net/http"
	"strings"
)

// CORSMiddleware adds CORS headers for allowed origins and handles preflight requests.
// allowedOrigins is a list of exact origins to allow (scheme + host + optional port).
// If allowCredentials is true, Access-Control-Allow-Credentials will be set to true.
func CORSMiddleware(next http.Handler, allowedOrigins []string, allowCredentials bool) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "" {
			continue
		}
		allowed[strings.TrimSpace(o)] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Always vary on Origin so caches don't mix responses
		w.Header().Add("Vary", "Origin")

		if origin != "" {
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if allowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				w.Header().Set("Access-Control-Expose-Headers", "X-Correlation-ID")
			}
		}

		// Handle preflight
		if r.Method == http.MethodOptions {
			// Set allowed methods and headers for preflight requests
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			reqHeaders := r.Header.Get("Access-Control-Request-Headers")
			if reqHeaders == "" {
				// Default allowed headers commonly used in this service
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Correlation-ID, Refresh-Token")
			} else {
				// Echo back requested headers
				w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
			}
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}