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

package main

import (
	"testing"

	"github.com/microbus-io/fabric/codegen/spec"
	"github.com/microbus-io/testarossa"
	"gopkg.in/yaml.v3"
)

func TestCodegen_FindReplaceReturnedErrors(t *testing.T) {
	t.Parallel()

	testCases := []string{
		`
return err
`, `
return errors.Trace(err)
`,

		`
	return err
`, `
	return errors.Trace(err)
`,

		`
	return 1, map[string]bool{}, err
`, `
	return 1, map[string]bool{}, errors.Trace(err)
`,

		`
	return err // No trace
`, `
	return err // No trace
`,

		`
	if err := doSomething(); err!=nil {
		return err
	}
`, `
	if err := doSomething(); err!=nil {
		return errors.Trace(err)
	}
`,
	}

	for i := 0; i < len(testCases); i += 2 {
		modified := findReplaceReturnedErrors(testCases[i])
		testarossa.Equal(t, testCases[i+1], modified)
	}
}

func TestCodegen_FindReplaceImportErrors(t *testing.T) {
	t.Parallel()

	testCases := []string{
		`
import "errors"
`, `
import "github.com/microbus-io/fabric/errors"
`,

		`
import (
	"errors"
	"fmt"
)
`, `
import (
	"fmt"

	"github.com/microbus-io/fabric/errors"
)
`,

		`
import (
	"fmt"
	"errors"
)
`, `
import (
	"fmt"

	"github.com/microbus-io/fabric/errors"
)
`,

		`
import (
	"fmt"
	"errors"
	"net"
)
`, `
import (
	"fmt"
	"net"

	"github.com/microbus-io/fabric/errors"
)
`,

		`
import (
	"errors"
)
`, `
import (
	"github.com/microbus-io/fabric/errors"
)
`,

		`
import (
	"fmt"
)
`, `
import (
	"fmt"
)
`,

		`
import "fmt"
`, `
import "fmt"
`,
	}

	for i := 0; i < len(testCases); i += 2 {
		modified := findReplaceImportErrors(testCases[i])
		testarossa.Equal(t, testCases[i+1], modified, "test case %d", (i/2)+1)
	}
}

func TestCodegen_FindReplaceSignatureAndDescription(t *testing.T) {
	t.Parallel()

	code := `
/*
Add two numbers.
*/
func (svc *Service) Add(ctx context.Context, x int, y int) (result int, err error) {
	return x+y, nil
}

/*
Concat two strings.
*/
func (svc *Service) Concat(ctx context.Context, x string, y string) (result string, err error) {
	return x+y, nil
}

/*
Not negates a boolean.
*/
func (svc *Service) Not(b bool) bool {
	return !b
}

func (svc *Service) NotFound(w http.ResponseWriter, r *http.Request) (err error) {
	w.WriteHeader(http.StatusNotFound)
	return
}

// TickTock runs periodically.
func (svc *Service) TickTock(ctx context.Context) (err error) {
	return
}
`

	var addSig spec.Signature
	yaml.Unmarshal([]byte("Add(x int, y int) (result int)"), &addSig)
	var concatSig spec.Signature
	yaml.Unmarshal([]byte("Concat(s1 string, s2 string) (merged string)"), &concatSig)
	var notFoundSig spec.Signature
	yaml.Unmarshal([]byte("NotFound()"), &notFoundSig)
	var tickTockSig spec.Signature
	yaml.Unmarshal([]byte("TickTock()"), &tickTockSig)

	specs := &spec.Service{
		Functions: []*spec.Handler{
			{
				Type:        "function",
				Signature:   &addSig,
				Description: "Add two numbers.",
			},
			{
				Type:        "function",
				Signature:   &concatSig,
				Description: "Concat merges two strings.",
			},
		},
		Webs: []*spec.Handler{
			{
				Type:        "web",
				Signature:   &notFoundSig,
				Description: "NotFound returns 404.",
			},
		},
		Tickers: []*spec.Handler{
			{
				Type:        "ticker",
				Signature:   &tickTockSig,
				Description: "TickTock runs periodically.",
			},
		},
	}

	modified := findReplaceSignature(specs, code)
	modified = findReplaceDescription(specs, modified)
	expected := `
/*
Add two numbers.
*/
func (svc *Service) Add(ctx context.Context, x int, y int) (result int, err error) {
	return x+y, nil
}

/*
Concat merges two strings.
*/
func (svc *Service) Concat(ctx context.Context, s1 string, s2 string) (merged string, err error) {
	return x+y, nil
}

/*
Not negates a boolean.
*/
func (svc *Service) Not(b bool) bool {
	return !b
}

/*
NotFound returns 404.
*/
func (svc *Service) NotFound(w http.ResponseWriter, r *http.Request) (err error) {
	w.WriteHeader(http.StatusNotFound)
	return
}

/*
TickTock runs periodically.
*/
func (svc *Service) TickTock(ctx context.Context) (err error) {
	return
}
`
	testarossa.Equal(t, expected, modified)
}
