package spec

import "strings"

// Argument is an input or output argument of a signature.
type Argument struct {
	Name string
	Type string
}

// EndType returns the final part of the type, excluding map, array and pointer markers.
// map[string]int -> int; []*User -> User
func (a *Argument) EndType() string {
	star := strings.LastIndex(a.Type, "*")
	bracket := strings.LastIndex(a.Type, "]")
	last := star
	if bracket > last {
		last = bracket
	}
	if last < 0 {
		last = -1
	}
	return a.Type[last+1:]
}
