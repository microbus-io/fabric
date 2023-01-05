package shardedsql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/rand"
)

/*
TestingDB is a temporary sharded database to be used for testing purposes.
By default, it
It uses the following data source pattern to create 3 shards in a single database server:

	root:secret1234@tcp(localhost:3306)/testing_{hhmmss}_{random}_%d

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
It attempts to connect to the database on localhost on the default port (3306 for MySQL),
using the default admin user ("root" for MySQL) with password "secret1234".
*/
func (db *TestingDB) Open(driverName string) (err error) {
	db.dbHost = "root:secret1234@tcp(localhost:3306)/"
	db.dbNameFormat = "testing_" + time.Now().Format("150405") + "_" + rand.AlphaNum32(6) + "_%d"

	// Create 3 databases, one per shard
	rootDB, err := sql.Open(driverName, db.dbHost)
	if err != nil {
		return errors.Trace(err)
	}
	for i := 1; i < 3; i++ {
		_, err = rootDB.Exec("DROP DATABASE IF EXISTS " + fmt.Sprintf(db.dbNameFormat, i))
		if err != nil {
			rootDB.Close()
			return errors.Trace(err)
		}
		_, err = rootDB.Exec("CREATE DATABASE " + fmt.Sprintf(db.dbNameFormat, i))
		if err != nil {
			rootDB.Close()
			return errors.Trace(err)
		}
	}
	rootDB.Close()

	// Open the sharded database
	db.DB, err = Open(driverName, db.dbHost+db.dbNameFormat)
	if err != nil {
		return errors.Trace(err)
	}
	// Register the shards
	for i := 1; i < 3; i++ {
		db.RegisterShard(i, false)
	}
	return nil
}

// Close cleans up after testing is done.
func (db *TestingDB) Close() error {
	// Close the sharded database
	err := db.DB.Close()
	if err != nil {
		return errors.Trace(err)
	}

	// Delete the temporary databases
	rootDB, err := sql.Open(db.driver, db.dbHost)
	if err != nil {
		return errors.Trace(err)
	}
	defer rootDB.Close()
	for i := 1; i <= 3; i++ {
		_, err = rootDB.Exec("DROP DATABASE IF EXISTS " + fmt.Sprintf(db.dbNameFormat, i))
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Delete leftover testing databases not from the last 2 hours
	rows, err := rootDB.Query("SHOW DATABASES")
	if err != nil {
		return errors.Trace(err)
	}
	defer rows.Close()
	now := time.Now()
	thisHour := now.Format("15")
	prevHour := now.Add(time.Duration(-time.Hour)).Format("15")
	for rows.Next() {
		var x string
		err = rows.Scan(&x)
		if err == nil &&
			strings.HasPrefix(x, "testing_") &&
			!strings.HasPrefix(x, "testing_"+thisHour) &&
			!strings.HasPrefix(x, "testing_"+prevHour) {
			_, err = rootDB.Exec("DROP DATABASE IF EXISTS " + x)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}
	return nil
}
