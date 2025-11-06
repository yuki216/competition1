package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/fixora/fixora/infrastructure/http/middleware"
)

func main() {
	allowed := []string{"http://localhost:3000", "https://competition-v1.netlify.app"}
	h := middleware.CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), allowed, true)

	req := httptest.NewRequest(http.MethodOptions, "http://example.com/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	fmt.Println("Status:", rr.Code)
	for k, v := range rr.Header() {
		fmt.Println(k+":", v)
	}
}
