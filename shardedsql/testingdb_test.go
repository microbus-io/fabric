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
