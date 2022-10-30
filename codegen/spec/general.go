package spec

import (
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// General are general properties of the microservice.
type General struct {
	Host        string `yaml:"host"`
	Description string `yaml:"description"`
}

// Validate indicates if the specs are valid.
func (g *General) Validate() error {
	err := utils.ValidateHostName(g.Host)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.Contains(g.Description, "`") {
		return errors.New("backquote character not allowed")
	}
	return nil
}
