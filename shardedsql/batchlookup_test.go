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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BatchLookup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrations := NewStatementSequence("batchlookup")
	migrations.Insert(1, "CREATE TABLE batchlookup (k INTEGER, PRIMARY KEY (k))")
	err := testingDB.MigrateSchema(ctx, migrations)
	assert.NoError(t, err)

	// Insert some records
	n := 101
	var keys []int
	shard1 := testingDB.Shard(1)
	for i := 0; i < n; i++ {
		_, err := shard1.ExecContext(ctx, "INSERT INTO batchlookup (k) VALUES (?)", i)
		assert.NoError(t, err)
		if i != 5 { // Exclude record 5
			keys = append(keys, i)
		}
	}

	// Batch lookup should return all records (excluding 5) in order
	at := 0
	batches := 0
	blu := NewBatchLookupInts(shard1, "SELECT k FROM batchlookup WHERE k IN (?)", keys)
	blu.SetBatchSize(7)
	for blu.Next() {
		batches++
		rows, err := blu.QueryContext(ctx)
		assert.NoError(t, err)
		for rows.Next() {
			var k int
			err := rows.Scan(&k)
			assert.NoError(t, err)
			assert.Equal(t, at, k)
			at++
			if at == 5 {
				// Record 5 is excluded
				at++
			}
		}
	}
	assert.Equal(t, n, at)
	assert.Equal(t, (n+6)/7, batches)
}
