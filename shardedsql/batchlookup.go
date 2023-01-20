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
	"sort"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// BatchLookup is a utility that optimizes the lookup of records given an arbitrary collection of keys.
type BatchLookup struct {
	shard     *Shard
	query     string
	keys      []any
	batchSize int
}

/*
NewBatchLookupInts creates a new batch lookup.
The query must be a SELECT ... IN (?) statement with a single ?.

Usage:

	blu := NewBatchLookup(shard, "SELECT ... IN (?)", keys)
	for blu.Next() {
		rows, err := blu.QueryContext(ctx)
		for rows.Next() {
			err = rows.Scan(...)
		}
	}
*/
func NewBatchLookupInts(shard *Shard, query string, keys []int) *BatchLookup {
	sortedKeys := make([]any, len(keys))
	for i, k := range keys {
		sortedKeys[i] = k
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i].(int) < sortedKeys[j].(int)
	})
	return &BatchLookup{
		shard:     shard,
		query:     query,
		keys:      sortedKeys,
		batchSize: 1000,
	}
}

/*
NewBatchLookupStrings creates a new batch lookup.
The query must be a SELECT ... IN (?) statement with a single ?.

Usage:

	blu := NewBatchLookup(shard, "SELECT ... IN (?)", keys)
	for blu.Next() {
		rows, err := blu.QueryContext(ctx)
		for rows.Next() {
			err = rows.Scan(...)
		}
	}
*/
func NewBatchLookupStrings(shard *Shard, query string, keys []string) *BatchLookup {
	sortedKeys := make([]any, len(keys))
	for i, k := range keys {
		sortedKeys[i] = k
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i].(int) < sortedKeys[j].(int)
	})
	return &BatchLookup{
		shard:     shard,
		query:     query,
		keys:      sortedKeys,
		batchSize: 1000,
	}
}

// SetBatchSize sets the number of records to lookup at a time.
// The default is 1000.
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

	// Get the keys in the current btach
	var keys []any
	if len(blu.keys) > blu.batchSize {
		keys = blu.keys[:blu.batchSize]
		blu.keys = blu.keys[blu.batchSize:]
	} else {
		keys = blu.keys
		blu.keys = nil
	}

	// Run the query
	questionMarks := "(?" + strings.Repeat(",?", len(keys)-1) + ")"
	query := strings.Replace(blu.query, "(?)", questionMarks, 1)
	rows, err = blu.shard.QueryContext(ctx, query, keys...)
	return rows, errors.Trace(err)
}
