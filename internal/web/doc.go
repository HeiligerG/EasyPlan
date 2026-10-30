// Package web serves the HTML interface of EasyPlan.
//
// It provides the HTTP handlers, the embedded templates and the embedded
// static files (CSS and HTMX). Handlers read and write through package
// store and delegate every calculation to package plan.
package web
