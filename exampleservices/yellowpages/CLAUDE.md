# yellowpages.example

## Purpose

The `yellowpages.example` microservice is the example of a SQL-backed microservice. It persists `Person` records
(first name, last name, email, birthday) in a SQL database through the sequel package, following
`.claude/rules/sequel.txt`. It exposes the same five operations twice: as JSON endpoints (`Create`, `Load`,
`Store`, `Delete`, `List`) for callers inside the application, and as REST endpoints on `/persons` and
`/persons/{key}` for callers outside it. The `Demo` web page drives the REST endpoints, and the `take-tour` skill
links to it.

## Design notes

**Deliberately minimal.** It shows the seam between sequel and a microservice (opening, migrating and closing the
database in `OnStartup` and `OnShutdown`, the per-test database, the telemetry wiring) and endpoints that run their
SQL directly. It has no tenants, revisions, events, bulk endpoints or opaque keys, which `sequel.txt` describes for
the microservices that need them.

**The JSON endpoints report "not found" as a boolean, not an error.** `Load`, `Store` and `Delete` return `found`,
`stored` and `deleted`. An error returned from an endpoint is logged as one, and a missing row is an ordinary
outcome for the caller to handle. The REST wrappers turn the booleans into 404s.

**The REST endpoints wrap the JSON handlers.** Each calls the matching JSON handler and maps its result to an HTTP
status (201 on create, 204 on store and delete, 404 when missing), so the SQL lives in one place.

**`Store` double-checks a zero row count.** MySQL reports the rows an `UPDATE` changed rather than the rows it
matched, so storing unchanged values within the same millisecond as the last update counts zero rows. `Store`
then checks whether the person exists before reporting that it was not found.

**The endpoints are open.** They sit on `:443` with no `requiredClaims`, so that the tour can drive them from a
browser. An application would gate them with `requiredClaims`, or move the ones meant only for other
microservices to `:444`.

**One migration, every supported database.** `resources/sql/1.sql` creates the `persons` table on MySQL, MariaDB,
PostgreSQL, CockroachDB, SQL Server and SQLite, with a unique index on `email`. Each engine is its own sequel driver
name, so the MySQL statements are marked `-- DRIVER: mysql mariadb` and the PostgreSQL ones `pgx cockroachdb`; a
MariaDB opened through a `mariadb://` data source name skips a statement marked `mysql` alone. Without a configured
`SQLDataSourceName`, a LOCAL deployment uses a `local.sqlite` file in the working directory, and tests use a testing
database per test app.

**The table and the migration sequence are new names.** An earlier, template-based version of this microservice
kept its rows in a `person` table under the migration sequence `person@a2b8aca8`. The current schema uses the
`persons` table and the sequence `yellowpages.persons`, so that a database the earlier version migrated runs the
new migration instead of keeping the old schema; the old table is left unused.
