package shardedsql

import (
	"context"
	"database/sql"
)

// Stmt wraps the SQL statement to be able to convert all time arguments to UTC.
type Stmt struct {
	*sql.Stmt
}

// ExecContext executes a prepared statement with the given arguments and
// returns a Result summarizing the effect of the statement.
func (s *Stmt) ExecContext(ctx context.Context, args ...interface{}) (sql.Result, error) {
	return s.Stmt.ExecContext(ctx, argsToUTC(args)...)
}

// Exec executes a prepared statement with the given arguments and
// returns a Result summarizing the effect of the statement.
//
// Exec uses context.Background internally; to specify the context, use
// ExecContext.
func (s *Stmt) Exec(args ...interface{}) (sql.Result, error) {
	return s.Stmt.Exec(argsToUTC(args)...)
}

// QueryContext executes a prepared query statement with the given arguments
// and returns the query results as a *Rows.
func (s *Stmt) QueryContext(ctx context.Context, args ...interface{}) (*sql.Rows, error) {
	return s.Stmt.QueryContext(ctx, argsToUTC(args)...)
}

// Query executes a prepared query statement with the given arguments
// and returns the query results as a *Rows.
//
// Query uses context.Background internally; to specify the context, use
// QueryContext.
func (s *Stmt) Query(args ...interface{}) (*sql.Rows, error) {
	return s.Stmt.Query(argsToUTC(args)...)
}

// QueryRowContext executes a prepared query statement with the given arguments.
// If an error occurs during the execution of the statement, that error will
// be returned by a call to Scan on the returned *Row, which is always non-nil.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
func (s *Stmt) QueryRowContext(ctx context.Context, args ...interface{}) *sql.Row {
	return s.Stmt.QueryRowContext(ctx, argsToUTC(args)...)
}

// QueryRow executes a prepared query statement with the given arguments.
// If an error occurs during the execution of the statement, that error will
// be returned by a call to Scan on the returned *Row, which is always non-nil.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
//
// Example usage:
//
//	var name string
//	err := nameByUseridStmt.QueryRow(id).Scan(&name)
//
// QueryRow uses context.Background internally; to specify the context, use
// QueryRowContext.
func (s *Stmt) QueryRow(args ...interface{}) *sql.Row {
	return s.Stmt.QueryRow(argsToUTC(args)...)
}
