package foremanapi

import (
	"github.com/microbus-io/dwarf/workflow"
)

// Query is the filter shape shared by ListOpen, ListClosed and Purge.
type Query = workflow.Query
