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
