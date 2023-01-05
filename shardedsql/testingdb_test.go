package shardedsql

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_OpenClose(t *testing.T) {
	var testingDB TestingDB
	err := testingDB.Open("mysql")
	assert.NoError(t, err)
	dbNameFormat := strings.TrimSuffix(testingDB.dbNameFormat, "_%d")

	root, err := sql.Open("mysql", "root:secret1234@tcp(localhost:3306)/")
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

	// Close the testing database
	err = testingDB.Close()
	assert.NoError(t, err)

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
