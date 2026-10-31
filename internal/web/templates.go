package web

import (
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

// layoutTemplate is the name of the template of the layout. The templates
// are parsed from the file system of the binary, so the name of the file is
// the name of the template.
const layoutTemplate = "layout.html"

// pages are the templates of the application.
//
// Every page is parsed together with the layout, so the layout can include
// the content of the page.
var pages = []string{
	"dashboard",
	"entries",
	"entryform",
	"entrydelete",
	"timeline",
	"accounts",
	"accountedit",
	notFoundPage,
	errorPage,
}

// The names of the pages that are not part of the navigation.
const (
	notFoundPage = "notfound"
	errorPage    = "error"
)

// templateFuncs are the functions the templates may call.
var templateFuncs = template.FuncMap{
	// money formats an amount in Rappen as whole francs.
	//
	// The amount is a plain text, so the escaper of html/template turns the
	// apostrophe of the thousands separator into an entity. The browser shows
	// the apostrophe again, the amount stays readable.
	"money": FormatMoney,
	// amount formats an unrounded amount in Rappen as whole francs.
	//
	// The amount is a plain text, like the money function.
	"amount": FormatAmount,
	// date formats a calendar day for the display.
	"date": FormatDate,
	// isodate formats a calendar day for a form field.
	"isodate": FormatISODate,
}

// loadTemplates parses the templates of all pages.
func (s *server) loadTemplates() error {
	s.templates = make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		tmpl, err := template.New(layoutTemplate).
			Funcs(templateFuncs).
			ParseFS(templateFS, "templates/"+layoutTemplate, "templates/"+page+".html")
		if err != nil {
			return fmt.Errorf("web: cannot parse the templates of page %q: %w", page, err)
		}
		s.templates[page] = tmpl
	}
	return nil
}
