# Prompts

## Initial scaffold

Create a SQL-backed microservice named "yellowpages" in `exampleservices/yellowpages/` with hostname
`yellowpages.example` that persists `Person` records in a SQL database using the sequel package, following
`.claude/rules/sequel.txt`.

A `Person` has a key assigned by the database, a first name and a last name (required, at most 64 characters,
trimmed), an email (required, at most 256 characters, trimmed, unique), an optional birthday that must be in the
past, and created and updated times set by the database.

Add a secret `SQLDataSourceName` config, and open, migrate and close the database in `OnStartup` and `OnShutdown`.
Support MySQL, MariaDB, PostgreSQL, CockroachDB, SQL Server and SQLite in the migration, in a `persons` table.

Add JSON endpoints `Create`, `Load` (returning `found`), `Store` (returning `stored`), `Delete` (returning `deleted`)
and `List`, which filters by first name, last name and email, orders by key, and pages with a limit (100 by
default, at most 1000) and an offset.

Add REST endpoints that wrap them: `POST /persons`, `GET /persons/{key}`, `PUT /persons/{key}`,
`DELETE /persons/{key}` and `GET /persons`, returning 201, 200, 204 and 404 as appropriate.

## Demo web page

Add a web endpoint `Demo` at `/demo` that serves a REST API explorer from `resources/demo.html`. On POST it reads
the form values `method`, `path` and `body`, makes the request to `https://yellowpages.example/{path}` with that
method (sending the body as JSON for POST and PUT), and renders the response status and body under the form. The
page lists example links for each REST call, prefilled through the `method`, `path` and `body` query arguments.
