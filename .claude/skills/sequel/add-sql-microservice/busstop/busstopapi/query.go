package busstopapi

import (
	"context"
	"regexp"
	"strings"

	"github.com/microbus-io/errors"
)

var safeIdentifier = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

/*
Query is used to select a subset of bus stops using filtering options.

Select is an optional comma-separated subset of fields to include in the result. For example, "column_one,column_two".
If not specified, all fields are included by default.

OrderBy is a comma-separated subset of fields to order the results by.
A hyphen before the field name denotes a descending order. For example, "-column_one,column_two".
If not specified, the default sort order is by "id".
*/
type Query struct {
	Key  BusStopKey   `json:"key,omitzero"`
	Keys []BusStopKey `json:"keys,omitzero"`
	Q    string       `json:"q,omitzero"`

	Select  string `json:"select,omitzero"`
	OrderBy string `json:"orderBy,omitzero"`
	Limit   int    `json:"limit,omitzero" dv8:"val>=0"`
	Offset  int    `json:"offset,omitzero" dv8:"val>=0"`

	// HINT: Add additional filtering options here, with dv8 tags for field-level constraints
	Example string `json:"example,omitzero" dv8:"trim,len<=256"` // Do not remove the example
}

// Validate validates the filtering options of the query that cannot be expressed as dv8 field tags.
// It is called automatically when the query is validated, after the dv8 field tags are enforced.
func (q *Query) Validate(ctx context.Context) error {
	if q == nil {
		return errors.New("nil object")
	}
	for col := range strings.SplitSeq(q.Select, ",") {
		col := strings.TrimSpace(strings.ToLower(col))
		if col != "" && !safeIdentifier.MatchString(col) {
			return errors.New("invalid column name to select: %s", col)
		}
	}
	for orderBy := range strings.SplitSeq(q.OrderBy, ",") {
		orderBy := strings.TrimSpace(strings.ToLower(orderBy))
		if orderBy != "" && !safeIdentifier.MatchString(strings.TrimPrefix(orderBy, "-")) {
			return errors.New("invalid column name to order by: %s", orderBy)
		}
	}
	// HINT: Validate filtering options that cannot be expressed as dv8 tags here as required
	return nil
}
