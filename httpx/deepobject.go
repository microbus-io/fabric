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

package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

var jsonNumberRegexp = regexp.MustCompile(`^(\-?)(0|([1-9][0-9]*))(\.[0-9]+)?([eE][\+\-]?[0-9]+)?$`)

// EncodeDeepObject encodes an object into string representation with bracketed nested fields names.
// For example, color[R]=100&color[G]=200&color[B]=150 .
func EncodeDeepObject(obj any) (url.Values, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(obj)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var m map[string]any
	err = json.NewDecoder(&buf).Decode(&m)
	if err != nil {
		return nil, errors.Trace(err)
	}
	result := make(url.Values)
	encodeOne("", m, result)
	return result, nil
}

func encodeOne(prefix string, obj any, values url.Values) {
	var val string
	switch fieldObj := obj.(type) {
	case map[string]any:
		for k, v := range fieldObj {
			if prefix == "" {
				encodeOne(k, v, values)
			} else {
				encodeOne(prefix+"["+k+"]", v, values)
			}
		}
		return
	case string:
		val = fieldObj
	case bool:
		val = strconv.FormatBool(fieldObj)
	case int64:
		val = strconv.FormatInt(fieldObj, 10)
	case int:
		val = strconv.FormatInt(int64(fieldObj), 10)
	case float64:
		val = strconv.FormatFloat(fieldObj, 'g', -1, 64)
	case float32:
		val = strconv.FormatFloat(float64(fieldObj), 'g', -1, 64)
	default:
		if obj == nil {
			val = "null"
		} else {
			val = fmt.Sprintf("%v", fieldObj)
		}
	}
	values.Set(prefix, val)
}

// DecodeDeepObject decodes an object from a string representation with bracketed nested fields names.
// For example, color[R]=100&color[G]=200&color[B]=150 .
func DecodeDeepObject(values url.Values, obj any) error {
	for k, vv := range values {
		for _, v := range vv {
			err := decodeOne(k, v, obj)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}
	return nil
}

func decodeOne(k, v string, data any) error {
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
	if jErr, ok := err.(*json.UnmarshalTypeError); ok {
		if strings.HasPrefix(jErr.Value, "string") {
			j = jPre + v + strings.Repeat("}", countDots+1)
			err = json.Unmarshal([]byte(j), data)
		} else {
			j = jPre + `"` + v + `"` + strings.Repeat("}", countDots+1)
			err = json.Unmarshal([]byte(j), data)
		}
	}
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
