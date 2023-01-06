package shardedsql

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/microbus-io/fabric/errors"
)

// Shard wraps a SQL connection to a single shard.
type Shard struct {
	*sql.DB
	shardIndex int
	locked     bool
}

// open opens the connection to the database of the shard.
func (s *Shard) open(driver string, dataSource string) error {
	if s.DB != nil {
		return nil
	}
	// See https://github.com/go-sql-driver/mysql#dsn-data-source-name
	cfg, err := mysql.ParseDSN(dataSource)
	if err != nil {
		return errors.Trace(err)
	}
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["parseTime"] = "true"
	cfg.Params["timeout"] = "4s"
	cfg.Params["readTimeout"] = "8s"
	cfg.Params["writeTimeout"] = "8s"
	s.DB, err = sql.Open(driver, cfg.FormatDSN())
	if err != nil {
		return errors.Trace(err)
	}
	err = s.Ping()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Close is a noop because the lifecycle of the shard connection is managed.
func (s *Shard) Close() error {
	return nil
}

// ShardIndex is the index of the shard.
func (s *Shard) ShardIndex() int {
	return s.shardIndex
}

// ExecContext executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (s *Shard) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// Convert all time arguments to UTC
	for i, a := range args {
		if tm, ok := a.(time.Time); ok {
			args[i] = tm.UTC()
		}
	}
	return s.DB.ExecContext(ctx, query, args...)
}

// Exec executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (s *Shard) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.ExecContext(context.Background(), query, args...)
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (s *Shard) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	// Convert all time arguments to UTC
	for i, a := range args {
		if tm, ok := a.(time.Time); ok {
			args[i] = tm.UTC()
		}
	}
	return s.DB.QueryContext(ctx, query, args...)
}

// Query executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (s *Shard) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.QueryContext(context.Background(), query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row.
// QueryRowContext always returns a non-nil value.
// Errors are deferred until Row's Scan method is called.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards the rest.
func (s *Shard) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// Convert all time arguments to UTC
	for i, a := range args {
		if tm, ok := a.(time.Time); ok {
			args[i] = tm.UTC()
		}
	}
	return s.DB.QueryRowContext(ctx, query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always returns a non-nil value.
// Errors are deferred until Row's Scan method is called.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards the rest.
func (s *Shard) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.QueryRowContext(context.Background(), query, args...)
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
