// Package web serves the HTML interface of EasyPlan.
//
// It provides the HTTP handlers, the embedded templates and the embedded
// static files (CSS and HTMX). Handlers read and write through package
// store and delegate every calculation to package plan.
//
// # Layout of the package
//
//   - web.go holds the constructor, the routes and the rendering of the pages.
//   - templates.go loads the embedded templates and the template functions.
//   - static.go serves the embedded CSS and HTMX file.
//   - origin.go and security.go hold the middlewares.
//   - health.go answers the check of the health of the application.
//   - backup.go writes the copy of the database, sends it as a download and
//     removes the temporary directories that a killed backup left behind.
//   - money.go formats and parses the amounts.
//   - entries.go serves the list of the entries and the deletion.
//   - entryform.go holds the form of an entry and its validation.
//   - accounts.go serves the list of the accounts and the actions on them.
//   - active.go filters the entries that count in the summaries.
//   - summary.go builds the values of the dashboard.
//   - timeline.go serves the timeline and builds its months.
//
// # Dependencies
//
// The constructor takes the store, the clock and the time zone, so the
// handlers never read the current time themselves and the tests can run with
// a fixed date. The clock is a function, the store is an interface, so a
// test can use the SQLite database in memory.
package web
