package connector

import (
	"testing"
)

func TestConnector_Implements(t *testing.T) {
	t.Parallel()

	c := NewConnector()
	_ = Service(c)
}
