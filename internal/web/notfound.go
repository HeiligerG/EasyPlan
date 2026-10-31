package web

import "net/http"

// notFoundPage serves the page for an unknown address.
//
// Every address that no route matches ends here, so the visitor gets the
// same page instead of the plain text of the standard library.
func (s *server) notFoundPage(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusNotFound, notFoundPage, PageData{
		Title: "Seite nicht gefunden",
	})
}
