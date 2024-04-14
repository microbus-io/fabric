/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package spec

import "strings"

// Type is a complex type used in a function.
type Type struct {
	Name    string
	Exists  bool
	Package string
}

// PackageSuffix returns only the last portion of the full package path.
func (t *Type) PackageSuffix() string {
	p := strings.LastIndex(t.Package, "/")
	if p < 0 {
		return t.Package
	}
	return t.Package[p+1:]
}
