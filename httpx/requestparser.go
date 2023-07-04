/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpx

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

var jsonNumberRegexp = regexp.MustCompile(`^(\-?)(0|([1-9][0-9]*))(\.[0-9]+)?([eE][\+\-]?[0-9]+)?$`)

// ParseRequestData parses the body and query arguments of an incoming request
// and populates a data object that represents its input arguments.
func ParseRequestData(r *http.Request, data any) error {
	contentType := r.Header.Get("Content-Type")
	// Parse JSON in the body
	if contentType == "application/json" {
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Parse form in body
	if contentType == "application/x-www-form-urlencoded" {
		err := r.ParseForm()
		if err != nil {
			return errors.Trace(err)
		}
		for k, vv := range r.PostForm {
			for _, v := range vv {
				err := readOneArg(k, v, data)
				if err != nil {
					return errors.Trace(err)
				}
			}
		}
	}

	// Parse query args
	for k, vv := range r.URL.Query() {
		for _, v := range vv {
			err := readOneArg(k, v, data)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}

	return nil
}

func readOneArg(k, v string, data any) error {
	// Arg names can be hierarchical: a[b][c] or a.b.c
	// Convert bracket notation to dot notation: a[b][c] -> a.b.c
	j := strings.ReplaceAll(k, "]", "")
	j = strings.ReplaceAll(j, "[", ".")
	countDots := strings.Count(j, ".")

	// Convert to JSON format: a.b.c -> {"a":{"b":{"c":
	j = `{"` + strings.ReplaceAll(j, ".", `":{"`) + `":`
	jPre := j

	switch {
	case v == "":
		j += `""`
	case v == "null" || v == "true" || v == "false":
		j += v
	case jsonNumberRegexp.MatchString(v):
		j += v
	case strings.HasPrefix(v, `[`) && strings.HasSuffix(v, `]`):
		var jArray []any
		err := json.Unmarshal([]byte(v), &jArray)
		if err != nil {
			return errors.Trace(err)
		}
		j += v
	case strings.HasPrefix(v, `{`) && strings.HasSuffix(v, `}`):
		var jMap map[string]any
		err := json.Unmarshal([]byte(v), &jMap)
		if err != nil {
			return errors.Trace(err)
		}
		j += v
	case strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`):
		v = strings.TrimPrefix(v, `"`)
		v = strings.TrimSuffix(v, `"`)
		fallthrough
	default:
		quotedValue, err := json.Marshal(v)
		if err != nil {
			return errors.Trace(err)
		}
		j += string(quotedValue)
	}

	// Close the braces
	j += strings.Repeat("}", countDots+1)

	// Override values in the data
	err := json.Unmarshal([]byte(j), data)
	if err != nil && strings.Contains(err.Error(), "cannot unmarshal number") {
		// json: cannot unmarshal number into Go struct field ... of type string [500]
		j = jPre + `"` + v + `"` + strings.Repeat("}", countDots+1)
		err = json.Unmarshal([]byte(j), data)
	}
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
