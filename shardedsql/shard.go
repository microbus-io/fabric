/*
Copyright 2023 Microbus Open Source Foundation and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shardedsql

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/errors"
)

// Shard wraps a SQL connection to a single shard.
type Shard struct {
	*sql.DB
	shardIndex int
	locked     bool
}

// Close is a noop because the lifecycle of the shard connection is managed.
func (s *Shard) Close() error {
	return nil
}

// ShardIndex is the index of the shard.
func (s *Shard) ShardIndex() int {
	return s.shardIndex
}

// Locked indicates if the shard accepts new allocation of sharding keys.
func (s *Shard) Locked() bool {
	return s.locked
}

// argsToUTC converts time arguments to UTC in order to avoid issues with time zone conversion.
// See https://dev.mysql.com/doc/refman/8.0/en/datetime.html .
func argsToUTC(args []any) []any {
	for i, a := range args {
		switch v := a.(type) {
		case time.Time:
			args[i] = v.UTC()
		case clock.NullTime:
			args[i] = clock.NewNullTimeUTC(v.Time)
		}
	}
	return args
}

// ExecContext executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (s *Shard) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.DB.ExecContext(ctx, query, argsToUTC(args)...)
}

// Exec executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (s *Shard) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.DB.Exec(query, argsToUTC(args)...)
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (s *Shard) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return s.DB.QueryContext(ctx, query, argsToUTC(args)...)
}

// Query executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (s *Shard) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.DB.Query(query, argsToUTC(args)...)
}

// QueryRowContext executes a query that is expected to return at most one row.
// QueryRowContext always returns a non-nil value.
// Errors are deferred until Row's Scan method is called.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards the rest.
func (s *Shard) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return s.DB.QueryRowContext(ctx, query, argsToUTC(args)...)
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always returns a non-nil value.
// Errors are deferred until Row's Scan method is called.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards the rest.
func (s *Shard) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.DB.QueryRow(query, argsToUTC(args)...)
}

// Prepare creates a prepared statement for later queries or executions.
// Multiple queries or executions may be run concurrently from the
// returned statement.
// The caller must call the statement's Close method
// when the statement is no longer needed.
//
// Prepare uses context.Background internally; to specify the context, use
// PrepareContext.
func (s *Shard) Prepare(query string) (*Stmt, error) {
	stmt, err := s.DB.Prepare(query)
	return &Stmt{stmt}, errors.Trace(err)
}

// PrepareContext creates a prepared statement for later queries or executions.
// Multiple queries or executions may be run concurrently from the
// returned statement.
// The caller must call the statement's Close method
// when the statement is no longer needed.
//
// The provided context is used for the preparation of the statement, not for the
// execution of the statement.
func (s *Shard) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	stmt, err := s.DB.PrepareContext(ctx, query)
	return &Stmt{stmt}, errors.Trace(err)
}

// MigrateSchema compares the migrations against the list of migrations already executed,
// then executes the new migrations in order.
// The statements are guaranteed to run in order of the sequence number within the context of a
// globally unique name. Good practice is to use the name of the owner microservice.
// Sequence names are limited to 256 ASCII characters.
func (s *Shard) MigrateSchema(ctx context.Context, statementSequence *StatementSequence) error {
	// Init the schema migration table
	_, err := s.Exec(
		`CREATE TABLE IF NOT EXISTS microbus_schema_migrations (
			name VARCHAR(256) CHARACTER SET ascii NOT NULL,
			seq INT NOT NULL,
			completed BOOL NOT NULL DEFAULT FALSE,
			completed_on DATETIME(3),
			locked_until DATETIME(3) NOT NULL DEFAULT NOW(3),
			PRIMARY KEY (name, seq)
		)`)
	if err != nil {
		return errors.Trace(err)
	}

	// Query for the high watermark
	var nullableWatermark sql.NullInt32
	row := s.QueryRowContext(ctx, "SELECT MAX(seq) FROM microbus_schema_migrations WHERE name=? AND completed=TRUE", statementSequence.Name)
	err = row.Scan(&nullableWatermark)
	if err != nil {
		return errors.Trace(err)
	}
	watermark := 0
	if nullableWatermark.Valid {
		watermark = int(nullableWatermark.Int32)
	}

	// Execute the migrations
	sequenceNumbersToRun := statementSequence.Order()
	for len(sequenceNumbersToRun) > 0 {
		seqNum := sequenceNumbersToRun[0]
		if seqNum <= watermark {
			sequenceNumbersToRun = sequenceNumbersToRun[1:]
			continue
		}

		// Insert new migrations into the database first
		// Ignore duplicate key violations
		_, err = s.ExecContext(ctx, `INSERT IGNORE INTO microbus_schema_migrations (name, seq, locked_until) VALUES (?, ?, NOW(3))`, statementSequence.Name, seqNum)
		if err != nil {
			return errors.Trace(err)
		}

		// See if completed by another process
		row := s.QueryRowContext(ctx, "SELECT completed FROM microbus_schema_migrations WHERE name=? AND seq=?", statementSequence.Name, seqNum)
		var completed bool
		err := row.Scan(&completed)
		if err != nil {
			return errors.Trace(err)
		}
		if completed {
			sequenceNumbersToRun = sequenceNumbersToRun[1:]
			continue
		}

		// Try to obtain a lock
		res, err := s.ExecContext(ctx,
			`UPDATE microbus_schema_migrations SET locked_until=DATE_ADD(NOW(3), INTERVAL 15 SECOND)
			WHERE name=? AND seq=? AND locked_until<NOW(3) AND completed=FALSE`,
			statementSequence.Name, seqNum)
		if err != nil {
			return errors.Trace(err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return errors.Trace(err)
		}
		if affected == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Obtained lock, execute migration in a goroutine
		done := make(chan error)
		go func() {
			statement := strings.ReplaceAll(statementSequence.Statements[seqNum], "\r", "")
			for _, stmt := range strings.Split(statement, ";\n") {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				_, e := s.ExecContext(ctx, stmt)
				if e != nil {
					done <- e
					return
				}
			}
			done <- nil
		}()

		// Wait for it to finish
		exit := false
		for !exit {
			select {
			case err = <-done:
				exit = true
			case <-time.After(5 * time.Second):
				// Extend the lock while the migration is in progress
				_, err = s.ExecContext(ctx,
					`UPDATE microbus_schema_migrations SET locked_until=DATE_ADD(NOW(3), INTERVAL 15 SECOND) WHERE name=? AND seq=?`,
					statementSequence.Name, seqNum)
				if err != nil {
					exit = true
				}
			}
		}

		if err != nil {
			// Release the lock
			_, _ = s.ExecContext(ctx,
				`UPDATE microbus_schema_migrations SET locked_until=NOW(3) WHERE name=? AND seq=?`,
				statementSequence.Name, seqNum)
			return errors.Trace(err, statementSequence.Statements[seqNum])
		}

		// Mark as complete
		_, err = s.ExecContext(ctx,
			`UPDATE microbus_schema_migrations SET locked_until=NOW(3), completed_on=NOW(3), completed=TRUE WHERE name=? AND seq=?`,
			statementSequence.Name, seqNum)
		if err != nil {
			return errors.Trace(err)
		}
		sequenceNumbersToRun = sequenceNumbersToRun[1:]
	}
	return nil
}
