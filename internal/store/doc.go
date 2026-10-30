// Package store owns the SQLite persistence of EasyPlan.
//
// It holds the embedded schema migrations and all SQL queries of the app. The
// store exposes the data model of Konzept.md §3 (accounts and entries), the
// calculations live in package plan.
//
// # Schema
//
// The schema is created by the migration scripts in migrations, which are
// embedded into the binary. The version of the schema is kept in the header
// of the database file (PRAGMA user_version), so no extra table is needed.
// The migrations of a start run in a single transaction: the change of the
// schema and the new version either both happen or neither does. The version
// is read inside that transaction, so two processes that start at the same
// time cannot apply the same migration twice. Opening a database that is
// already up to date changes nothing, so the app can be started as often as
// it likes. A database whose schema is newer than the migrations of the
// binary is refused.
//
// The default accounts of Konzept.md §3 are part of the first migration, so
// they exist in every new database and never come back after they have been
// deleted.
//
// # Connections
//
// The pragmas are part of the data source name, because the SQLite driver
// applies them to every connection it opens. A pragma that is only set on a
// single connection would be lost as soon as the pool hands out another one.
// foreign_keys=ON therefore belongs into the data source name, otherwise the
// foreign keys of the schema are not enforced.
//
// The file database runs in WAL mode, the in-memory database of the tests
// cannot use it and does not ask for it.
//
// # Concurrency
//
// A Store is safe for concurrent use, database/sql takes care of the
// connections. Every method reads its rows completely before it returns, so a
// method never holds a connection while it runs another query.
package store
