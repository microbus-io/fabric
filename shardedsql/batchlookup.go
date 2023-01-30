/*
Copyright 2023 Microbus LLC and various contributors

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
	"fmt"
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
The query must be a SELECT {fields} FROM {table} WHERE {key_field} IN (?) statement.
The keys must be specified as an array []any, []string or []int.

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
	result.batchSize = 500
	return result
}

// SetBatchSize sets the number of records to lookup at a time.
// The default is 500. Most databases will not handle more than 1000.
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

	// Identify the index field so we can sort
	indexField := ""
	if len(keyArgs) > 1 {
		ucQuery := strings.ToUpper(blu.query)
		p := strings.Index(ucQuery, " IN ")
		if p >= 0 {
			q := strings.LastIndex(ucQuery[:p], " ")
			if q >= 0 {
				indexField = blu.query[q+1 : p]
			}
		}
	}

	// Run the query
	questionMarks := "(?" + strings.Repeat(",?", len(keyArgs)-1) + ")"
	query := strings.Replace(blu.query, "(?)", questionMarks, 1)

	if indexField != "" {
		query += " ORDER BY FIND_IN_SET(" + indexField + ",'"
		for i, a := range keyArgs {
			if i > 0 {
				query += fmt.Sprintf(",%v", a)
			} else {
				query += fmt.Sprintf("%v", a)
			}
		}
		query += "')"
	}

	rows, err = blu.shard.QueryContext(ctx, query, args...)
	return rows, errors.Trace(err)
}
