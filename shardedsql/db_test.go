package shardedsql

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testingDB TestingDB

func TestMain(m *testing.M) {
	testingDB.Open("mysql")
	code := m.Run()
	testingDB.Close()
	os.Exit(code)
}

func Test_Migrations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrations := []*SchemaMigration{
		// The order of the list should not matter if sequences are in order
		{
			Name:      "migrations",
			Sequence:  22,
			Statement: `ALTER TABLE migrations ADD v INTEGER`,
		},
		{
			Name:      "migrations",
			Sequence:  11,
			Statement: `CREATE TABLE migrations (k INTEGER)`,
		},
	}
	err := testingDB.MigrateSchema(ctx, migrations)
	assert.NoError(t, err)

	// Add a new migration after executing the first two
	migrations = append(migrations, &SchemaMigration{
		Name:      "migrations",
		Sequence:  33,
		Statement: `ALTER TABLE migrations ADD w INTEGER`,
	})
	err = testingDB.MigrateSchema(ctx, migrations)
	assert.NoError(t, err)

	for i, sh := range testingDB.Shards() {
		// All migrations should complete
		row := sh.QueryRow("SELECT COUNT(*) FROM microbus_schema_migrations WHERE name='migrations' AND completed=TRUE")
		var count int
		err = row.Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, len(migrations), count)

		// The table should exist with all 3 columns
		_, err = sh.Exec("INSERT INTO migrations (k,v,w) VALUES (?,?,?)", i, i, i)
		assert.NoError(t, err)
	}
}

func Test_FailingMigration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrations := []*SchemaMigration{
		{
			Name:      "failingmigration",
			Sequence:  999,
			Statement: `INVALID SQL`,
		},
	}
	err := testingDB.MigrateSchema(ctx, migrations)
	assert.Error(t, err)

	for _, sh := range testingDB.Shards() {
		// Migrations should not complete but be listed
		row := sh.QueryRow("SELECT COUNT(*) FROM microbus_schema_migrations WHERE name='failingmigration' AND completed=FALSE")
		var count int
		err = row.Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, len(migrations), count)
	}
}

func Test_Sharding(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrations := []*SchemaMigration{
		{
			Name:      "sharding",
			Sequence:  1,
			Statement: `CREATE TABLE sharding (k INTEGER AUTO_INCREMENT, v INTEGER, PRIMARY KEY (k))`,
		},
	}
	err := testingDB.MigrateSchema(ctx, migrations)
	assert.NoError(t, err)

	for i := 0; i < 32; i++ {
		shardingKey, err := testingDB.Allocate()
		assert.NoError(t, err)
		sh, err := testingDB.ShardOf(shardingKey)
		assert.NoError(t, err)

		// Add a record to the shard
		res, err := sh.Exec("INSERT INTO sharding (v) VALUES (?)", i)
		assert.NoError(t, err)
		insertID, err := res.LastInsertId()
		assert.NoError(t, err)

		// Query
		rows, err := testingDB.Query(shardingKey, "SELECT k FROM sharding WHERE k=?", insertID)
		assert.NoError(t, err)
		assert.True(t, rows.Next())
		var k int64
		err = rows.Scan(&k)
		assert.NoError(t, err, "shard", i)
		assert.Equal(t, insertID, k)
		assert.False(t, rows.Next())

		// QueryRow
		row, err := testingDB.QueryRow(shardingKey, "SELECT k FROM sharding WHERE k=?", insertID)
		assert.NoError(t, err)
		err = row.Scan(&k)
		assert.NoError(t, err)
		assert.Equal(t, insertID, k)

		// Execute
		res, err = testingDB.Exec(shardingKey, "DELETE FROM sharding WHERE k=?", insertID)
		assert.NoError(t, err)
		affected, err := res.RowsAffected()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	}
}

func Test_MaxOpenConnections(t *testing.T) {
	// No parallel

	ctx := context.Background()
	migrations := []*SchemaMigration{
		{
			Name:      "maxopenconnections",
			Sequence:  1,
			Statement: `CREATE TABLE maxopenconnections (k INTEGER AUTO_INCREMENT, PRIMARY KEY (k))`,
		},
	}
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
	db2, err := Open(testingDB.Driver(), testingDB.DataSource())
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

	db2, err := Open(testingDB.Driver(), testingDB.DataSource())
	assert.NoError(t, err)
	defer db2.Close()
	assert.Equal(t, testingDB.DB, db2)
}
