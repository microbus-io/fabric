package shardedsql

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/rand"
)

var (
	singletonsMap = map[string]*DB{}
	singletonMux  sync.Mutex
)

// DB is a sharded database.
type DB struct {
	driver     string
	dataSource string
	shards     map[int]*Shard
	refCount   int
	mux        sync.Mutex
}

/*
Open creates a sharded database and opens up connections to all its shards.

The database driver name will be used to open the connections.
Only "mysql" is supported at this time.

The data source name must include %d indicating where to insert the shard index.
For example:

	// A different host per shard
	username:password@tcp(shard%d.mysql.host:3306)/db
	// A different database name per shard on a single host
	username:password@tcp(localhost:3306)/db%d
*/
func Open(driver string, dataSource string) (*DB, error) {
	if !strings.Contains(dataSource, "%d") {
		return nil, errors.New("missing '%d' in data source")
	}
	singletonMux.Lock()
	defer singletonMux.Unlock()

	cached, ok := singletonsMap[driver+"|"+dataSource]
	if ok {
		cached.mux.Lock()
		defer cached.mux.Unlock()
		cached.refCount++
		cached.adjustConnectionLimits()
		return cached, nil
	}

	// Open connection to shard 1
	shard1, err := sql.Open(driver, fmt.Sprintf(dataSource, 1))
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Init the shards table
	_, err = shard1.Exec(
		`CREATE TABLE IF NOT EXISTS microbus_shards (
			id INT NOT NULL,
			locked BOOL NOT NULL DEFAULT FALSE,
			PRIMARY KEY (id)
		)`)
	if err != nil {
		shard1.Close()
		return nil, errors.Trace(err)
	}
	// Init the sharding keys table
	_, err = shard1.Exec(
		`CREATE TABLE IF NOT EXISTS microbus_sharding_keys (
			id BIGINT NOT NULL AUTO_INCREMENT,
			shard_id INT NOT NULL,
			PRIMARY KEY (id)
		)`)
	if err != nil {
		shard1.Close()
		return nil, errors.Trace(err)
	}
	// Register shard 1
	_, err = shard1.Exec(`INSERT IGNORE INTO microbus_shards (id, locked) VALUES (1, FALSE)`)
	if err != nil {
		shard1.Close()
		return nil, errors.Trace(err)
	}

	// Identify all shards
	rows, err := shard1.Query(`SELECT id, locked FROM microbus_shards`)
	if err != nil {
		shard1.Close()
		return nil, errors.Trace(err)
	}
	shards := []*Shard{}
	for rows.Next() {
		shard := &Shard{}
		err = rows.Scan(&shard.shardIndex, &shard.locked)
		if err != nil {
			shard1.Close()
			return nil, errors.Trace(err)
		}
		shards = append(shards, shard)
	}
	shard1.Close()

	// Open all shards
	for _, shard := range shards {
		err = shard.open(driver, fmt.Sprintf(dataSource, shard.shardIndex))
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	db := &DB{
		driver:     driver,
		dataSource: dataSource,
		shards:     map[int]*Shard{},
		refCount:   1,
	}
	for _, shard := range shards {
		db.shards[shard.shardIndex] = shard
	}
	db.adjustConnectionLimits()
	singletonsMap[driver+"|"+dataSource] = db
	return db, nil
}

// Close releases the reference to this database.
// The connections themselves are not closed because they may be in use by other clients.
func (db *DB) Close() error {
	singletonMux.Lock()
	defer singletonMux.Unlock()
	db.mux.Lock()
	defer db.mux.Unlock()

	db.refCount--
	if db.refCount == 0 {
		for _, shard := range db.shards {
			shard.DB.Close()
			shard.DB = nil
		}
		delete(singletonsMap, db.driver+"|"+db.dataSource)
		return nil
	}
	db.adjustConnectionLimits()
	return nil
}

// NumShards is the total number of shards.
func (db *DB) NumShards() int {
	db.mux.Lock()
	defer db.mux.Unlock()

	return len(db.shards)
}

// Shards returns a list of all registered shards.
func (db *DB) Shards() map[int]*Shard {
	db.mux.Lock()
	defer db.mux.Unlock()

	clone := make(map[int]*Shard, len(db.shards))
	for index, shard := range db.shards {
		clone[index] = shard
	}
	return clone
}

// DataSource is the data source name used to create this database.
func (db *DB) DataSource() string {
	return db.dataSource
}

// Driver is the name of the database driver used to create this database.
func (db *DB) Driver() string {
	return db.driver
}

// Shard returns the shard specified by its index (one-based).
// The shard is a connection to a database.
func (db *DB) Shard(index int) *Shard {
	db.mux.Lock()
	defer db.mux.Unlock()

	return db.shards[index]
}

// ShardOf returns the shard that maps to the sharding key.
func (db *DB) ShardOf(shardingKey int) (*Shard, error) {
	shard1 := db.Shard(1)
	row := shard1.QueryRow(`SELECT shard_id FROM microbus_sharding_keys WHERE id=?`, shardingKey)
	var shardIndex int
	err := row.Scan(&shardIndex)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return db.Shard(shardIndex), nil
}

// Allocate creates a new sharding key and assigns it to a random unlocked shard.
func (db *DB) Allocate() (shardingKey int, err error) {
	unlocked := []int{}
	db.mux.Lock()
	for _, shard := range db.shards {
		if !shard.locked {
			unlocked = append(unlocked, shard.shardIndex)
		}
	}
	db.mux.Unlock()
	if len(unlocked) == 0 {
		return 0, errors.New("all shards are locked")
	}
	// Random shard
	// TODO: weighted random
	randomShardIndex := unlocked[rand.Intn(len(unlocked))]

	shard1 := db.Shard(1)
	res, err := shard1.Exec(`INSERT INTO microbus_sharding_keys (shard_id) VALUES (?)`, randomShardIndex)
	if err != nil {
		return 0, errors.Trace(err)
	}
	lastInsertID, err := res.LastInsertId()
	if err != nil {
		return 0, errors.Trace(err)
	}
	return int(lastInsertID), nil
}

// RegisterShard registers a new shard.
func (db *DB) RegisterShard(shardIndex int, locked bool) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	if db.shards[shardIndex] != nil {
		// Shard already registered
		return nil
	}

	shard1 := db.shards[1]
	_, err := shard1.Exec(`INSERT IGNORE INTO microbus_shards (id, locked) VALUES (?, ?)`, shardIndex, locked)
	if err != nil {
		return errors.Trace(err)
	}

	newShard := &Shard{
		shardIndex: shardIndex,
		locked:     locked,
	}
	err = newShard.open(db.driver, fmt.Sprintf(db.dataSource, shardIndex))
	if err != nil {
		return errors.Trace(err)
	}
	db.shards[shardIndex] = newShard
	db.adjustConnectionLimits()

	return nil
}

// connectionLimits returns the maximum number of connections in the idle connection pool,
// and the maximum number of open connections to the database, based on the ref count.
//
//	n	maxIdle	maxOpen
//	1	1	4
//	2	2	6
//	5	3	8
//	10	4	10
//	17	5	12
//	26	6	14
//	37	7	16
//	50	8	18
//	65	9	20
//	82	10	22
//	101	11	24
//	...
//	1025	33	68
//	...
func (db *DB) connectionLimits(refCount int) (maxOpen, maxIdle int) {
	maxIdleF := math.Ceil(math.Sqrt(float64(refCount)))
	maxOpenF := math.Ceil(maxIdleF*2) + 2
	return int(maxOpenF), int(maxIdleF)
}

// adjustConnectionLimits adjusts the maximum number of connections in the idle connection pool,
// and the maximum number of open connections to the database.
func (db *DB) adjustConnectionLimits() {
	maxOpen, maxIdle := db.connectionLimits(db.refCount)
	for _, shard := range db.shards {
		shard.SetMaxOpenConns(maxOpen)
		shard.SetMaxIdleConns(maxIdle)
	}
}

// Query calls Query on the shard matching the sharding key.
// An error is returned if a shard cannot be found for the sharding key.
func (db *DB) Query(shardingKey int, query string, args ...interface{}) (*sql.Rows, error) {
	return db.QueryContext(context.Background(), shardingKey, query, args...)
}

// QueryContext calls QueryContext on the shard matching the sharding key.
// An error is returned if a shard cannot be found for the sharding key.
func (db *DB) QueryContext(ctx context.Context, shardingKey int, query string, args ...interface{}) (*sql.Rows, error) {
	shard, err := db.ShardOf(shardingKey)
	if err != nil {
		return nil, err
	}
	return shard.QueryContext(ctx, query, args...)
}

// QueryRow calls QueryRow on the shard matching the sharding key.
// An error is returned if a shard cannot be found for the sharding key.
func (db *DB) QueryRow(shardingKey int, query string, args ...interface{}) (*sql.Row, error) {
	return db.QueryRowContext(context.Background(), shardingKey, query, args...)
}

// QueryRowContext calls QueryRowContext on the shard matching the sharding key.
// An error is returned if a shard cannot be found for the sharding key.
func (db *DB) QueryRowContext(ctx context.Context, shardingKey int, query string, args ...interface{}) (*sql.Row, error) {
	shard, err := db.ShardOf(shardingKey)
	if err != nil {
		return nil, err
	}
	return shard.QueryRowContext(ctx, query, args...), nil
}

// Exec calls Exec on the shard matching the sharding key.
// An error is returned if a shard cannot be found for the sharding key.
func (db *DB) Exec(shardingKey int, query string, args ...interface{}) (sql.Result, error) {
	return db.ExecContext(context.Background(), shardingKey, query, args...)
}

// ExecContext calls ExecContext on the shard matching the sharding key.
// An error is returned if a shard cannot be found for the sharding key.
func (db *DB) ExecContext(ctx context.Context, shardingKey int, query string, args ...interface{}) (sql.Result, error) {
	shard, err := db.ShardOf(shardingKey)
	if err != nil {
		return nil, err
	}
	return shard.ExecContext(ctx, query, args...)
}

// MigrateSchema migrates the schema of all shards in parallel.
func (db *DB) MigrateSchema(ctx context.Context, migrations []*SchemaMigration) (err error) {
	shards := db.Shards()
	errs := make(chan error, len(shards))
	for _, shard := range shards {
		shard := shard
		go func() {
			e := shard.MigrateSchema(ctx, migrations)
			errs <- errors.Trace(e)
		}()
	}
	for i := 0; i < len(shards); i++ {
		e := <-errs
		if e != nil {
			err = e
		}
	}
	return err
}
