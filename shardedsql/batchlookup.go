/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package shardedsql

import (
	"context"
	"database/sql"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// BatchLookup is a utility that optimizes the lookup of records given an arbitrary collection of keys.
type BatchLookup struct {
	shard     *Shard
	query     string
	before    []any
	keys      []any
	after     []any
	batchSize int
}

/*
NewBatchLookup creates a new batch lookup.
The query must be a SELECT ... IN (?) statement.
The keys must be specified as an array []any, []string or []int.
The results are returned in random order, not the order of the input keys.

Usage:

	blu := NewBatchLookup(shard, "SELECT * FROM table WHERE tenant_id=? AND id IN (?)", tenantID, keys)
	for blu.Next() {
		rows, err := blu.QueryContext(ctx)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(...)
			if err != nil {
				return err
			}
		}
	}
*/
func NewBatchLookup(shard *Shard, query string, args ...any) *BatchLookup {
	result := &BatchLookup{
		shard: shard,
		query: query,
	}

	found := false
	for _, arg := range args {
		switch v := arg.(type) {
		case []any:
			found = true
			result.keys = v
		case []int:
			found = true
			anyKeys := make([]any, len(v))
			for i := range v {
				anyKeys[i] = v[i]
			}
			result.keys = anyKeys
		case []string:
			found = true
			anyKeys := make([]any, len(v))
			for i := range v {
				anyKeys[i] = v[i]
			}
			result.keys = anyKeys
		default:
			if found {
				result.after = append(result.after, arg)
			} else {
				result.before = append(result.before, arg)
			}
		}
	}
	result.batchSize = 1000
	return result
}

// SetBatchSize sets the number of records to lookup at a time.
// The default is 1000. Most databases will not handle more than 1000.
func (blu *BatchLookup) SetBatchSize(n int) {
	if n > 0 {
		blu.batchSize = n
	}
}

// Next indicates if there is another batch of keys left to query.
func (blu *BatchLookup) Next() bool {
	return len(blu.keys) > 0
}

// Query runs the query for the next batch of keys.
func (blu *BatchLookup) Query() (rows *sql.Rows, err error) {
	return blu.QueryContext(context.Background())
}

// QueryContext runs the query for the next batch of keys.
func (blu *BatchLookup) QueryContext(ctx context.Context) (rows *sql.Rows, err error) {
	if len(blu.keys) == 0 {
		return nil, sql.ErrNoRows
	}

	// Get the args in the current batch
	var args []any
	var keyArgs []any
	args = append(args, blu.before...)
	if len(blu.keys) > blu.batchSize {
		keyArgs = blu.keys[:blu.batchSize]
		args = append(args, blu.keys[:blu.batchSize]...)
		blu.keys = blu.keys[blu.batchSize:]
	} else {
		keyArgs = blu.keys
		args = append(args, blu.keys...)
		blu.keys = nil
	}
	args = append(args, blu.after...)

	// Run the query
	questionMarks := "(?" + strings.Repeat(",?", len(keyArgs)-1) + ")"
	query := strings.Replace(blu.query, "(?)", questionMarks, 1)
	rows, err = blu.shard.QueryContext(ctx, query, args...)
	return rows, errors.Trace(err)
}
