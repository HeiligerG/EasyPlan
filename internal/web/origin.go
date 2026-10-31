package web

import (
	"net/http"
	"net/url"
	"strings"
)

// originCheck rejects the writing requests that come from a foreign origin.
//
// Konzept.md §9 protects the forms against cross-site requests without a
// token: the browser sends the origin of the page with every writing
// request, and the server compares it with the host it was addressed to.
//
// A request without an origin is rejected as well. A request of the
// application always has one, so a missing origin means that the request
// does not come from the interface.
//
// The Host header is used and not X-Forwarded-Host: behind a reverse proxy
// the host stays the host of the application, which is what the browser sent
// as the origin.
func (s *server) originCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isWritingMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		if !originMatches(r) {
			s.logger.Warn("request rejected",
				"reason", "origin does not match the host",
				"method", r.Method,
				"path", r.URL.Path,
				"origin", r.Header.Get("Origin"),
				"host", r.Host,
			)
			s.render(w, http.StatusForbidden, errorPage, PageData{
				Title: "Zugriff verweigert",
				Error: "Die Anfrage kam von einer fremden Herkunft.",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isWritingMethod reports whether the method changes data.
func isWritingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// originMatches reports whether the origin of the request belongs to the
// host of the request.
//
// The scheme is ignored, so the check also works behind a proxy that
// terminates TLS. The host with its port is compared in lower case.
func originMatches(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if parsed.Host == "" {
		// The origin "null" of a sandboxed frame has no host and belongs to
		// nobody.
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}
