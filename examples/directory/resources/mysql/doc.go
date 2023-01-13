// Code generated by Microbus. DO NOT EDIT.

// Package mysql includes schema definition and migration scripts for the MySQL database.
package mysql

/*
SQL files places in this directory are executed during database initialization.
Each SQL script is executed only once, in order of the number in its file name.
Files must be named 1.sql, 2.sql, etc.
A script may contain multiple statements separated by a semi-colon that is followed
by a new line.

Typical schema definition and migration use cases:

	CREATE TABLE persons (
		tenant_id INT NOT NULL,
		person_id BIGINT NOT NULL AUTO_INCREMENT,
		name VARCHAR(256) CHARACTER SET ascii,
		created DATETIME(3) NOT NULL DEFAULT NOW(3),
		PRIMARY KEY (tenant_id, person_id),
		INDEX idx_name (name ASC)
	)

	ALTER TABLE persons
		DROP COLUMN created,
		ADD COLUMN first_name VARCHAR(128) CHARACTER SET ascii,
		ADD COLUMN last_name VARCHAR(128) CHARACTER SET ascii

	CREATE INDEX idx_last_name ON persons (last_name ASC)
*/
