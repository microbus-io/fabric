package utils

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
	k = strings.ReplaceAll(k, "]", "")
	k = strings.ReplaceAll(k, "[", ".")
	countDots := strings.Count(k, ".")

	// Convert to JSON format: a.b.c -> {"a":{"b":{"c":
	k = `{"` + strings.ReplaceAll(k, ".", `":{"`) + `":`

	switch {
	case v == "":
		k += `""`
	case v == "null" || v == "true" || v == "false":
		k += v
	case jsonNumberRegexp.MatchString(v):
		k += v
	case strings.HasPrefix(v, `[`) && strings.HasSuffix(v, `]`):
		var jArray []any
		err := json.Unmarshal([]byte(v), &jArray)
		if err != nil {
			return errors.Trace(err)
		}
		k += v
	case strings.HasPrefix(v, `{`) && strings.HasSuffix(v, `}`):
		var jMap map[string]any
		err := json.Unmarshal([]byte(v), &jMap)
		if err != nil {
			return errors.Trace(err)
		}
		k += v
	case strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`):
		v = strings.TrimPrefix(v, `"`)
		v = strings.TrimSuffix(v, `"`)
		fallthrough
	default:
		quotedValue, err := json.Marshal(v)
		if err != nil {
			return errors.Trace(err)
		}
		k += string(quotedValue)
	}

	// Close the braces
	k += strings.Repeat("}", countDots+1)

	// Override values in the data
	err := json.Unmarshal([]byte(k), data)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
