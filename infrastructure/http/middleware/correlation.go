package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const CorrelationIDHeader = "X-Correlation-ID"

// CorrelationIDMiddleware ensures every request/response carries a correlation ID
func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid := r.Header.Get(CorrelationIDHeader)
		if cid == "" {
			cid = generateCorrelationID()
		}
		// propagate header to response
		w.Header().Set(CorrelationIDHeader, cid)
		// continue the chain
		next.ServeHTTP(w, r)
	})
}

func generateCorrelationID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// fallback to a static value in worst case (should not happen)
		return "unknown"
	}
	return hex.EncodeToString(b)
}