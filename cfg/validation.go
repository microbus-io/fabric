/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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

package cfg

import (
	"encoding/json"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Validate validates that the value matches the rule.
// The rule itself is assumed to be valid.
func Validate(rule string, value string) (ok bool) {
	// Type
	typ, _ := normalizedType(rule)

	// Specs
	spec := ""
	space := strings.Index(rule, " ")
	if space >= 0 {
		spec = strings.TrimSpace(rule[space+1:])
	}

	switch typ {
	case "str":
		if spec == "" {
			return true
		}
		match, err := regexp.MatchString(spec, value)
		return match && err == nil

	case "bool":
		_, err := strconv.ParseBool(value)
		return err == nil

	case "int":
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return false
		}
		if spec == "" {
			return true
		}
		re := regexp.MustCompile(`^[\[\(](.*),(.*)[\)\]]$`)
		subs := re.FindStringSubmatch(spec)
		if subs[1] != "" {
			low, _ := strconv.ParseInt(subs[1], 10, 64)
			if strings.HasPrefix(spec, "[") && val < low {
				return false
			}
			if strings.HasPrefix(spec, "(") && val <= low {
				return false
			}
		}
		if subs[2] != "" {
			high, _ := strconv.ParseInt(subs[2], 10, 64)
			if strings.HasSuffix(spec, "]") && val > high {
				return false
			}
			if strings.HasSuffix(spec, ")") && val >= high {
				return false
			}
		}
		return true

	case "float":
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return false
		}
		if spec == "" {
			return true
		}
		re := regexp.MustCompile(`^[\[\(](.*),(.*)[\)\]]$`)
		subs := re.FindStringSubmatch(spec)
		if subs[1] != "" {
			low, _ := strconv.ParseFloat(subs[1], 64)
			if strings.HasPrefix(spec, "[") && val < low {
				return false
			}
			if strings.HasPrefix(spec, "(") && val <= low {
				return false
			}
		}
		if subs[2] != "" {
			high, _ := strconv.ParseFloat(subs[2], 64)
			if strings.HasSuffix(spec, "]") && val > high {
				return false
			}
			if strings.HasSuffix(spec, ")") && val >= high {
				return false
			}
		}
		return true

	case "dur":
		val, err := time.ParseDuration(value)
		if err != nil {
			return false
		}
		if spec == "" {
			return true
		}
		re := regexp.MustCompile(`^[\[\(](.*),(.*)[\)\]]$`)
		subs := re.FindStringSubmatch(spec)
		if subs[1] != "" {
			low, _ := time.ParseDuration(subs[1])
			if strings.HasPrefix(spec, "[") && val < low {
				return false
			}
			if strings.HasPrefix(spec, "(") && val <= low {
				return false
			}
		}
		if subs[2] != "" {
			high, _ := time.ParseDuration(subs[2])
			if strings.HasSuffix(spec, "]") && val > high {
				return false
			}
			if strings.HasSuffix(spec, ")") && val >= high {
				return false
			}
		}
		return true

	case "set":
		set := strings.Split(spec, "|")
		for _, s := range set {
			if value == s {
				return true
			}
		}
		return false

	case "url":
		u, err := url.Parse(value)
		return err == nil && u.Hostname() != ""

	case "email":
		em, err := mail.ParseAddress(value)
		if err != nil {
			return false
		}
		re := regexp.MustCompile(`^.+@.+\..{2,}$`)
		return re.MatchString(em.Address)

	case "json":
		var x any
		err := json.Unmarshal([]byte(value), &x)
		return err == nil

	default:
		return false
	}
}

/*
checkRule validates that the validation rule itself is valid.

Valid rules are:

	str ^[a-zA-Z0-9]+$
	bool
	int [0,60]
	float [0.0,1.0)
	dur (0s,24h]
	set Red|Green|Blue
	url
	email
	json

Whereas the following types are synonymous:

	str, string, text, (empty)
	bool, boolean
	int, integer, long
	float, double, decimal, number
	dur, duration
*/
func checkRule(rule string) (ok bool) {
	rule = strings.TrimSpace(rule)

	// Type
	typ, ok := normalizedType(rule)
	if !ok {
		return false
	}

	// Specs
	spec := ""
	space := strings.Index(rule, " ")
	if space >= 0 {
		spec = strings.TrimSpace(rule[space+1:])
	}
	if spec == "" {
		return typ != "set" // Set must be specified
	}

	switch typ {
	case "str":
		_, err := regexp.Compile(spec)
		return err == nil
	case "int", "float", "dur":
		re := regexp.MustCompile(`^[\[\(](.*),(.*)[\)\]]$`)
		if !re.MatchString(spec) {
			return false
		}
		subs := re.FindStringSubmatch(spec)
		if len(subs) != 1+2 { // First capturing group is the entire regexp
			return false
		}
		for i := 1; i < 3; i++ {
			if subs[i] != "" {
				var err error
				switch typ {
				case "int":
					_, err = strconv.ParseInt(subs[i], 10, 64)
				case "float":
					_, err = strconv.ParseFloat(subs[i], 64)
				case "dur":
					_, err = time.ParseDuration(subs[i])
				}
				if err != nil {
					return false
				}
			}
		}
		return true
	case "set":
		return true
	default:
		return false
	}
}

// normalizedType returns the normalized type of the config property:
// str, bool, int, float, dur, set, url, email, json.
func normalizedType(rule string) (normalized string, ok bool) {
	var ruleType string
	space := strings.Index(rule, " ")
	if space >= 0 {
		ruleType = rule[:space]
	} else {
		ruleType = rule
	}
	ruleType = strings.ToLower(ruleType)
	switch ruleType {
	case "", "str", "string", "text":
		return "str", true
	case "bool", "boolean":
		return "bool", true
	case "int", "integer", "long":
		return "int", true
	case "float", "double", "decimal", "number":
		return "float", true
	case "dur", "duration":
		return "dur", true
	case "set", "url", "email", "json":
		return ruleType, true
	default:
		return "str", false
	}
}
