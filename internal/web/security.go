package web

import "net/http"

// contentSecurityPolicy allows only the resources of the application itself.
//
// There is no inline script and no inline style, so the templates have to
// put their styles in the stylesheet and their scripts in a file.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self'; " +
	"img-src 'self' data:; " +
	"form-action 'self'; " +
	"base-uri 'self'; " +
	"frame-ancestors 'none'; " +
	"object-src 'none'"

// securityHeaders adds the security headers to every response.
func (s *server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Security-Policy", contentSecurityPolicy)
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
