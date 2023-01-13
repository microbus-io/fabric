package shardedsql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/rand"
)

const numTestingShards = 3

/*
TestingDB is a new sharded database for testing purposes.
It connects to the database on 127.0.0.1 on the default port (3306 for MySQL)
using the default admin user ("root" for MySQL) with password "secret1234".

For MySQL, it uses the following data source name pattern:

	root:secret1234@tcp(127.0.0.1:3306)/testing_{hhmmss}_{random}_%d

Usage:

	var testingDB TestingDB
	testingDB.Open("mysql")
	defer testingDB.Close()
*/
type TestingDB struct {
	*DB
	dbNameFormat string
	dbHost       string
}

/*
TestingDB creates a new sharded database for testing purposes.
It attempts to connect to the database on 127.0.0.1 on the default port (3306 for MySQL),
using the default admin user ("root" for MySQL) with password "secret1234".
*/
func (db *TestingDB) Open(driverName string) (err error) {
	ctx := context.Background()

	db.dbHost = "root:secret1234@tcp(127.0.0.1:3306)/"
	db.dbNameFormat = "testing_" + time.Now().Format("150405") + "_" + rand.AlphaNum32(6) + "_%d"

	// Open the sharded database
	shardedDB, err := Open(ctx, driverName, db.dbHost+db.dbNameFormat)
	if err != nil {
		return errors.Trace(err)
	}

	// Register two more shards
	_, err = shardedDB.Shard(1).ExecContext(ctx, `INSERT IGNORE INTO microbus_shards (id,locked) VALUES (2,FALSE),(3,FALSE)`)
	shardedDB.Close()
	if err != nil {
		return errors.Trace(err)
	}

	// Reopen the database so the new shards are discovered
	db.DB, err = Open(ctx, driverName, db.dbHost+db.dbNameFormat)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Close cleans up after testing is done.
func (db *TestingDB) Close() error {
	ctx := context.Background()

	// Close the sharded database
	if db.DB == nil {
		return nil
	}
	err := db.DB.Close()
	if err != nil {
		return errors.Trace(err)
	}

	// Delete the temporary databases
	dbRoot, err := openHost(ctx, db.driver, db.dbHost)
	if err != nil {
		return errors.Trace(err)
	}
	defer dbRoot.Close()
	for i := 1; i <= numTestingShards; i++ {
		_, err = dbRoot.ExecContext(ctx, "DROP DATABASE IF EXISTS "+fmt.Sprintf(db.dbNameFormat, i))
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Delete leftover testing databases not from the last 2 hours
	rows, err := dbRoot.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return errors.Trace(err)
	}
	defer rows.Close()
	now := time.Now()
	thisHour := now.Format("15")
	prevHour := now.Add(time.Duration(-time.Hour)).Format("15")
	var toDrop []string
	for rows.Next() {
		var x string
		err = rows.Scan(&x)
		if err == nil &&
			strings.HasPrefix(x, "testing_") &&
			!strings.HasPrefix(x, "testing_"+thisHour) &&
			!strings.HasPrefix(x, "testing_"+prevHour) {
			toDrop = append(toDrop, x)
		}
	}
	for _, x := range toDrop {
		_, err = dbRoot.ExecContext(ctx, "DROP DATABASE IF EXISTS "+x)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
