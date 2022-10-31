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

// ShortPackage returns only the last portion of the full package path.
func (v *Version) ShortPackage() string {
	return strings.TrimPrefix(v.Package, filepath.Dir(v.Package)+"/")
}
