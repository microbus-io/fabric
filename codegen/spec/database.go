/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package spec

import (
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// Database are the databases used by the microservice.
type Database struct {
	Name             string `yaml:"name"`
	Type             string `yaml:"type"`
	Runtime          bool   `yaml:"runtime"`
	IntegrationTests bool   `yaml:"integrationTests"`
}

// UnmarshalYAML parses and validates the YAML.
func (db *Database) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Unmarshal
	type different Database
	var x different
	x.IntegrationTests = true // Default
	x.Runtime = true          // Default
	err := unmarshal(&x)
	if err != nil {
		return errors.Trace(err)
	}
	*db = Database(x)

	db.Type = strings.ToLower(db.Type)
	if db.Type == "" {
		db.Type = "mariadb"
	}

	// Validate
	err = db.validate()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// validate validates the data after unmarshaling.
func (db *Database) validate() error {
	if !utils.IsUpperCaseIdentifier(db.Name) {
		return errors.Newf("database name '%s' must start with uppercase", db.Name)
	}
	if db.Type != "mysql" && db.Type != "mariadb" {
		return errors.Newf("unsupported database type '%s'", db.Type)
	}
	return nil
}

// IsSQL indicates if the database type is a SQL database.
func (db *Database) IsSQL() bool {
	return db.Type == "mysql" || db.Type == "mariadb"
}
