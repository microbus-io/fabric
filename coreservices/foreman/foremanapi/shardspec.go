package foremanapi

import (
	"github.com/microbus-io/dwarf/engine"
)

// ShardSpec declares one database shard: its index, connection string, connection budget, placement weight, and
// whether it is cordoned off from new-flow placement.
type ShardSpec = engine.ShardSpec
