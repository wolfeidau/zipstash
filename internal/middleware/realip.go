package middleware

import (
	"net/http"
)

// RealIP returns a middleware that sets a http.Request's RemoteAddr to the results of parsing either the X-Forwarded-For or X-Real-IP header fields.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try X-Forwarded-For first (AWS Lambda/ALB typically sets this)
		cip := r.Header.Get("X-Forwarded-For")
		if cip == "" {
			// Fallback to X-Real-IP
			cip = r.Header.Get("X-Real-IP")
		}

		if cip != "" {
			r.RemoteAddr = cip
		}

		next.ServeHTTP(w, r)
	})
}
