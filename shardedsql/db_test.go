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
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

var testingDB TestingDB

func TestMain(m *testing.M) {
	err := testingDB.Open("mariadb")
	if err != nil {
		fmt.Fprintf(os.Stderr, "--- FAIL: %+v\n", err)
		os.Exit(2)
	}
	code := m.Run()
	testingDB.Close()
	os.Exit(code)
}

func Test_Migrations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrations := NewStatementSequence("migrations")
	// The order of insertion should not matter
	migrations.Insert(22, "ALTER TABLE migrations ADD v INTEGER")
	migrations.Insert(11, "CREATE TABLE migrations (k INTEGER)")
	err := testingDB.MigrateSchema(ctx, migrations)
	assert.NoError(t, err)

	// Add a new migration after executing the first two
	migrations.Insert(33, "ALTER TABLE migrations ADD w INTEGER")
	err = testingDB.MigrateSchema(ctx, migrations)
	assert.NoError(t, err)

	// Migrations with a lower sequence than the high watermark should not be executed
	migrations.Insert(32, "DROP TABLE migrations")
	err = testingDB.MigrateSchema(ctx, migrations)
	assert.NoError(t, err)

	for i, sh := range testingDB.Shards() {
		// Migrations 11,22,33 should complete
		row := sh.QueryRow("SELECT COUNT(*) FROM microbus_schema_migrations WHERE name='migrations' AND completed=TRUE")
		var count int
		err = row.Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 3, count)

		// Migration 32 should not added to the database at all
		row = sh.QueryRow("SELECT COUNT(*) FROM microbus_schema_migrations WHERE name='migrations' AND completed=FALSE")
		err = row.Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)

		// The table should exist with all 3 columns
		_, err = sh.Exec("INSERT INTO migrations (k,v,w) VALUES (?,?,?)", i, i, i)
		assert.NoError(t, err)
	}
}

func Test_FailingMigration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrations := NewStatementSequence("failingmigration")
	migrations.Insert(999, "INVALID SQL")
	err := testingDB.MigrateSchema(ctx, migrations)
	assert.Error(t, err)

	for _, sh := range testingDB.Shards() {
		// Migrations should not complete but be listed
		row := sh.QueryRow("SELECT COUNT(*) FROM microbus_schema_migrations WHERE name='failingmigration' AND completed=FALSE")
		var count int
		err = row.Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	}
}

func Test_UnknownShard(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	shard := testingDB.ShardOf(ctx, -1)
	assert.Nil(t, shard)
	shard = testingDB.Shard(-1)
	assert.Nil(t, shard)
}

func Test_Sharding(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrations := NewStatementSequence("sharding")
	migrations.Insert(1, "CREATE TABLE sharding (k INTEGER, v INTEGER, PRIMARY KEY (k))")
	err := testingDB.MigrateSchema(ctx, migrations)
	assert.NoError(t, err)

	for i := 0; i < 32; i++ {
		shardingKey, err := testingDB.Allocate()
		assert.NoError(t, err)
		sh := testingDB.ShardOf(ctx, shardingKey)
		assert.NotNil(t, sh)

		// Add a record to the shard
		res, err := sh.Exec("INSERT INTO sharding (k, v) VALUES (?, ?)", shardingKey, i)
		assert.NoError(t, err)
		affected, err := res.RowsAffected()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)

		// Query
		rows, err := testingDB.Query(shardingKey, "SELECT k FROM sharding WHERE k=?", shardingKey)
		assert.NoError(t, err)
		assert.True(t, rows.Next())
		var k int
		err = rows.Scan(&k)
		assert.NoError(t, err)
		assert.Equal(t, shardingKey, k)
		assert.False(t, rows.Next())

		// QueryRow
		row := testingDB.QueryRow(shardingKey, "SELECT k FROM sharding WHERE k=?", shardingKey)
		err = row.Scan(&k)
		assert.NoError(t, err)
		assert.Equal(t, shardingKey, k)

		// Execute
		res, err = testingDB.Exec(shardingKey, "DELETE FROM sharding WHERE k=?", shardingKey)
		assert.NoError(t, err)
		affected, err = res.RowsAffected()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	}
}

func Test_MaxOpenConnections(t *testing.T) {
	// No parallel

	ctx := context.Background()
	migrations := NewStatementSequence("maxopenconnections")
	migrations.Insert(1, "CREATE TABLE maxopenconnections (k INTEGER AUTO_INCREMENT, PRIMARY KEY (k))")
	err := testingDB.MigrateSchema(ctx, migrations)
	assert.NoError(t, err)

	sh := testingDB.Shard(1)

	_, err = sh.Exec("INSERT INTO maxopenconnections VALUES ()")
	assert.NoError(t, err)
	_, err = sh.Exec("INSERT INTO maxopenconnections VALUES ()")
	assert.NoError(t, err)

	// Reach the limit of open connections
	openRows := []*sql.Rows{}
	maxOpen, _ := testingDB.connectionLimits(testingDB.refCount)
	for i := 0; i < maxOpen; i++ {
		rows, err := sh.QueryContext(ctx, "SELECT * FROM maxopenconnections")
		assert.NoError(t, err)
		openRows = append(openRows, rows)
	}

	// Next connection should fail
	quickCtx, cancel := context.WithTimeout(ctx, time.Millisecond*500)
	_, err = sh.QueryContext(quickCtx, "SELECT * FROM maxopenconnections")
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, quickCtx.Err())
	cancel()

	// Add a DB client so connection limits are increased
	assert.Equal(t, 1, testingDB.refCount)
	db2, err := Open(ctx, testingDB.Driver(), testingDB.DataSource())
	assert.NoError(t, err)
	assert.Equal(t, 2, testingDB.refCount)

	maxOpen, _ = testingDB.connectionLimits(testingDB.refCount)
	assert.True(t, maxOpen > len(openRows))

	// Should be able to open another connection now
	rows, err := sh.QueryContext(ctx, "SELECT * FROM maxopenconnections")
	assert.NoError(t, err)
	openRows = append(openRows, rows)

	for _, rows := range openRows {
		rows.Close()
	}

	db2.Close()
	assert.Equal(t, 1, testingDB.refCount)
}

func Test_Singleton(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db2, err := Open(ctx, testingDB.Driver(), testingDB.DataSource())
	assert.NoError(t, err)
	defer db2.Close()
	assert.Equal(t, testingDB.DB, db2)
}

func Test_Concurrent(t *testing.T) {
	// No parallel

	ctx := context.Background()
	migrations := NewStatementSequence("concurrent")
	migrations.Insert(1, "CREATE TABLE concurrent (k INTEGER, num INTEGER, PRIMARY KEY (k))")
	err := testingDB.MigrateSchema(ctx, migrations)
	assert.NoError(t, err)

	// 32 workers are competing for only 4 pooled connections
	var executions int32
	var wg sync.WaitGroup
	finishAt := time.Now().Add(time.Second)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(finishAt) {
				k, err := testingDB.Allocate()
				assert.NoError(t, err)

				res, err := testingDB.ExecContext(ctx, k, "INSERT INTO concurrent (k, num) VALUES (?, 1)", k)
				assert.NoError(t, err)
				affected, err := res.RowsAffected()
				assert.NoError(t, err)
				assert.Equal(t, int64(1), affected)
				atomic.AddInt32(&executions, 1)

				k = rand.Intn(k) + 1
				res, err = testingDB.ExecContext(ctx, k, "UPDATE concurrent SET num=num+1 WHERE k=?", k)
				assert.NoError(t, err)
				affected, err = res.RowsAffected() // may be 0 due to race condition
				assert.NoError(t, err)
				atomic.AddInt32(&executions, int32(affected))
			}
		}()
	}
	wg.Wait()

	for _, shard := range testingDB.Shards() {
		row := shard.QueryRow("SELECT SUM(num) FROM concurrent")
		var sum int32
		err := row.Scan(&sum)
		assert.NoError(t, err)
		executions -= sum
	}
	assert.Equal(t, int32(0), executions)
}
