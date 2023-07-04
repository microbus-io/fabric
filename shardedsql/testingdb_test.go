/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package shardedsql

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_OpenClose(t *testing.T) {
	var testingDB TestingDB
	err := testingDB.Open("mariadb")
	assert.NoError(t, err)
	dbNameFormat := strings.TrimSuffix(testingDB.dbNameFormat, "_%d")

	root, err := sql.Open("mysql", "root:secret1234@tcp(127.0.0.1:3306)/")
	assert.NoError(t, err)
	defer root.Close()

	// Databases should have been created
	rows, err := root.Query("SHOW DATABASES")
	assert.NoError(t, err)
	defer rows.Close()
	found := 0
	for rows.Next() {
		var x string
		err = rows.Scan(&x)
		assert.NoError(t, err)
		if strings.HasPrefix(x, dbNameFormat) {
			found++
		}
	}
	assert.Equal(t, testingDB.NumShards(), found)

	// Validate the connections to all the shards
	assert.Equal(t, numTestingShards, testingDB.NumShards())
	for _, shard := range testingDB.Shards() {
		err = shard.Ping()
		assert.NoError(t, err)
	}

	// Close the testing database
	err = testingDB.Close()
	assert.NoError(t, err)

	// Validate the connections were closed
	for _, shard := range testingDB.Shards() {
		assert.Nil(t, shard.DB)
	}

	// All databases should have been deleted
	rows, err = root.Query("SHOW DATABASES")
	assert.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var x string
		err = rows.Scan(&x)
		assert.NoError(t, err)
		assert.False(t, strings.HasPrefix(x, dbNameFormat))
	}
}
