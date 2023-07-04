/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package spec

import (
	"path/filepath"
	"strings"
)

// Version keeps the versioning information of the code.
type Version struct {
	Package   string
	Version   int    `json:"ver"`
	SHA256    string `json:"sha256"`
	Timestamp string `json:"ts"`
}

// PackageSuffix returns only the last portion of the full package path.
func (v *Version) PackageSuffix() string {
	return strings.TrimPrefix(v.Package, filepath.Dir(v.Package)+"/")
}
