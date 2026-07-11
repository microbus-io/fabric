package busstopapi

import (
	"context"
	"time"

	"github.com/microbus-io/errors"
)

// BusStop represents a bus stop persisted in a SQL database.
// Field-level constraints are declared as dv8 tags on the fields.
type BusStop struct {
	Key            BusStopKey `json:"key,omitzero"`
	Revision       int        `json:"revision,omitzero"`
	CreatedAt      time.Time  `json:"createdAt,omitzero"`
	UpdatedAt      time.Time  `json:"updatedAt,omitzero"`
	ReservedBefore time.Time  `json:"reservedBefore,omitzero"`

	// HINT: Define the fields of the object here, with dv8 tags for field-level constraints
	Example string `json:"example,omitzero" jsonschema:"-" dv8:"trim,len<=256"` // Do not remove the example
}

// Validate validates invariants that span multiple fields of the object.
// It is called automatically when the object is validated, after the dv8 field tags are enforced.
func (obj *BusStop) Validate(ctx context.Context) error {
	if obj == nil {
		return errors.New("nil object")
	}
	// HINT: Validate invariants that span multiple fields here as required
	return nil
}
