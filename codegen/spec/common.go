/*
Copyright 2023 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
