package main

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// PkgDedup chooses unique aliases for a list of packages that may contain packages with the same suffix.
type PkgDedup map[string]string

// Add a package path to the collection.
func (dd PkgDedup) Add(pkgPath string) {
	_, ok := dd[pkgPath]
	if ok {
		return
	}

	aliasOf := func(pkg string) string {
		alias := pkg
		p := strings.LastIndex(pkg, "/")
		if p > 0 {
			alias = pkg[p+1:]
		}
		if unicode.IsDigit([]rune(alias)[len([]rune(alias))-1]) {
			alias += "x"
		}
		return alias
	}

	sorted := []string{}
	for p := range dd {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)

	alias := aliasOf(pkgPath)
	found := 0
	for _, p := range sorted {
		if alias == aliasOf(p) {
			found++
			dd[p] = aliasOf(p) + strconv.Itoa(found)
		}
	}
	if found == 0 {
		dd[pkgPath] = alias
	} else {
		dd[pkgPath] = alias + strconv.Itoa(found+1)
	}
}
