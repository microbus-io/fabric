/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package spec

import (
	"strings"
)

// conformDesc cleans up the description by removing back-quotes and extra spaces.
// It also guarantees that the description starts with a certain prefix and that it's not empty.
func conformDesc(desc string, ifEmpty string, mustStartWith string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		desc = ifEmpty
	}

	desc = strings.ReplaceAll(desc, "`", "'")

	if !strings.HasPrefix(desc, mustStartWith) {
		desc = mustStartWith + " - " + desc
	}

	split := strings.Split(desc, "\n")
	for i := range split {
		split[i] = strings.TrimRight(split[i], " \r\t")
	}
	return strings.Join(split, "\n")
}
