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

package spec

import (
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// Database are the databases used by the microservice.
type Database struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// UnmarshalYAML parses and validates the YAML.
func (db *Database) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Unmarshal
	type different Database
	var x different
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
