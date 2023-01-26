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
	"math"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/lru"
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
	cache      *lru.Cache[int, int]
}

/*
Open creates a sharded database and opens up connections to all its shards.

The database driver name will be used to open the connections.
Only "mariadb" and "mysql" are supported at this time.

The data source name must include %d indicating where to insert the shard index.
For example:

	// A different host per shard
	username:password@tcp(db-shard%d.host:3306)/db
	// A different database name per shard on a single host
	username:password@tcp(db.host:3306)/db%d
*/
func Open(ctx context.Context, driver string, dataSource string) (*DB, error) {
	if driver == "mariadb" {
		driver = "mysql"
	}
	if driver != "mysql" {
		return nil, errors.Newf("driver '%s' is not supported", driver)
	}
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
	db1, err := openDatabase(ctx, driver, fmt.Sprintf(dataSource, 1))
	if err != nil {
		return nil, errors.Trace(err)
	}
	// Init the shards table
	_, err = db1.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS microbus_shards (
			id INT NOT NULL,
			locked BOOL NOT NULL DEFAULT FALSE,
			PRIMARY KEY (id)
		)`)
	if err != nil {
		db1.Close()
		return nil, errors.Trace(err)
	}
	// Init the sharding keys table
	_, err = db1.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS microbus_sharding_keys (
			id BIGINT NOT NULL AUTO_INCREMENT,
			shard_id INT NOT NULL,
			PRIMARY KEY (id)
		)`)
	if err != nil {
		db1.Close()
		return nil, errors.Trace(err)
	}
	// Register shard 1
	_, err = db1.ExecContext(ctx, `INSERT IGNORE INTO microbus_shards (id, locked) VALUES (1, FALSE)`)
	if err != nil {
		db1.Close()
		return nil, errors.Trace(err)
	}

	// Prepare the database struct
	db := &DB{
		driver:     driver,
		dataSource: dataSource,
		refCount:   1,
		cache:      lru.NewCache[int, int](),
		shards: map[int]*Shard{
			1: {
				shardIndex: 1,
				locked:     false,
				DB:         db1,
			},
		},
	}
	db.cache.SetMaxWeight(1024 * 1024)
	db.cache.SetMaxAge(time.Hour * 24)

	// Open the rest of the shards
	rows, err := db1.QueryContext(ctx, `SELECT id, locked FROM microbus_shards WHERE id!=1`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	for rows.Next() {
		var shard Shard
		err = rows.Scan(&shard.shardIndex, &shard.locked)
		if err != nil {
			rows.Close()
			db1.Close()
			return nil, errors.Trace(err)
		}
		db.shards[shard.shardIndex] = &shard
	}
	for _, shard := range db.shards {
		if shard.DB != nil {
			continue
		}
		shard.DB, err = openDatabase(ctx, db.driver, fmt.Sprintf(db.dataSource, shard.shardIndex))
		if err != nil {
			for _, shardToClose := range db.shards {
				if shardToClose.DB != nil {
					shardToClose.DB.Close()
					shardToClose.DB = nil
				}
			}
			return nil, errors.Trace(err)
		}
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

// ShardOf returns the shard that maps to the sharding key, or nil if the key has not been allocated to
// a shard or if the shard is unknown.
func (db *DB) ShardOf(ctx context.Context, shardingKey int) *Shard {
	if cachedIndex, ok := db.cache.Load(shardingKey); ok {
		return db.Shard(cachedIndex)
	}
	shard1 := db.Shard(1)
	row := shard1.QueryRowContext(ctx, `SELECT shard_id FROM microbus_sharding_keys WHERE id=?`, shardingKey)
	var shardIndex int
	err := row.Scan(&shardIndex)
	if err != nil {
		// Possibly sql.ErrNoRows
		return nil
	}
	db.cache.Store(shardingKey, shardIndex)
	return db.Shard(shardIndex)
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
	randomShardIndex := unlocked[rand.Intn(len(unlocked))]
	return db.AllocateTo(randomShardIndex)
}

// AllocateTo creates a new sharding key and assigns it to a specified shard.
// The key is assigned even if the shard is locked.
func (db *DB) AllocateTo(shardIndex int) (shardingKey int, err error) {
	shard1 := db.Shard(1)
	res, err := shard1.Exec(`INSERT INTO microbus_sharding_keys (shard_id) VALUES (?)`, shardIndex)
	if err != nil {
		return 0, errors.Trace(err)
	}
	lastInsertID, err := res.LastInsertId()
	if err != nil {
		return 0, errors.Trace(err)
	}
	db.cache.Store(int(lastInsertID), shardIndex)
	return int(lastInsertID), nil
}

// openHost opens the connection to the shard's server without selecting a specific database.
func openHost(ctx context.Context, driver string, dataSource string) (*sql.DB, error) {
	cfg, err := mysql.ParseDSN(dataSource)
	if err != nil {
		return nil, errors.Trace(err)
	}
	cfg.DBName = ""
	return openDatabase(ctx, driver, cfg.FormatDSN())
}

// openDatabase opens the connection to the shard's database.
func openDatabase(ctx context.Context, driver string, dataSource string) (*sql.DB, error) {
	cfg, err := mysql.ParseDSN(dataSource)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	// See https://github.com/go-sql-driver/mysql#dsn-data-source-name
	cfg.Params["parseTime"] = "true"
	cfg.Params["timeout"] = "4s"
	cfg.Params["readTimeout"] = "8s"
	cfg.Params["writeTimeout"] = "8s"
	sqlDB, err := sql.Open(driver, cfg.FormatDSN())
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = sqlDB.Ping()
	if err != nil && cfg.DBName != "" {
		var sqlErr *mysql.MySQLError
		if !errors.As(err, &sqlErr) || sqlErr.Number != 1049 {
			return nil, errors.Trace(err)
		}

		// Unknown database, create new one
		cfg, err := mysql.ParseDSN(dataSource)
		if err != nil {
			return nil, errors.Trace(err)
		}
		dbRoot, err := openHost(ctx, driver, dataSource)
		if err != nil {
			return nil, errors.Trace(err)
		}
		_, err = dbRoot.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+cfg.DBName)
		dbRoot.Close()
		if err != nil {
			return nil, errors.Trace(err)
		}

		// Retry
		sqlDB, err = openDatabase(ctx, driver, dataSource)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	return sqlDB, nil
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
	shard := db.ShardOf(ctx, shardingKey)
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
	shard := db.ShardOf(ctx, shardingKey)
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
	shard := db.ShardOf(ctx, shardingKey)
	return shard.ExecContext(ctx, query, args...)
}

// MigrateSchema migrates the schema of all shards in parallel.
// The statements are guaranteed to run in order of the sequence number within the context of a
// globally unique name. Good practice is to use the name of the owner microservice.
// Sequence names are limited to 256 ASCII characters.
func (db *DB) MigrateSchema(ctx context.Context, statementSequence *StatementSequence) (err error) {
	shards := db.Shards()
	if len(shards) == 1 && shards[1] != nil {
		err = shards[1].MigrateSchema(ctx, statementSequence)
		return errors.Trace(err)
	}
	errs := make(chan error, len(shards))
	for _, shard := range shards {
		shard := shard
		go func() {
			e := shard.MigrateSchema(ctx, statementSequence)
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
