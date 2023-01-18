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
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// Databases are names for the databases used by the microservice.
type Databases struct {
	MySQL string `yaml:"mysql"`
}

// UnmarshalYAML parses and validates the YAML.
func (db *Databases) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Unmarshal
	type different Databases
	var x different
	err := unmarshal(&x)
	if err != nil {
		return errors.Trace(err)
	}
	*db = Databases(x)

	// Validate
	err = db.validate()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// validate validates the data after unmarshaling.
func (db *Databases) validate() error {
	if db.MySQL != "" && !utils.IsUpperCaseIdentifier(db.MySQL) {
		return errors.Newf("MySQL database name '%s' must start with uppercase", db.MySQL)
	}
	return nil
}
