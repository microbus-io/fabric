/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/microbus-io/testarossa"
)

// updateGoldens rewrites the committed diagrams from genworkflowmmd output. Run after an intentional change to
// the generator or to the renderer:
//
//	go test ./cmd/genworkflowmmd/ -update
var updateGoldens = flag.Bool("update", false, "rewrite committed diagrams from genworkflowmmd output")

// TestRun renders the fixture's workflows end to end: it generates, compiles and runs the throwaway program
// against the fixture package, so a change in the renderer's API fails here rather than in housekeeping.
func TestRun(t *testing.T) {
	assert := testarossa.For(t)
	dir, err := filepath.Abs("testdata/flowsvc")
	if !assert.NoError(err) {
		return
	}
	goldens := map[string][]byte{}
	for _, name := range []string{"APPROVAL.mmd", "REVIEW.mmd"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if !*updateGoldens && !assert.NoError(err, "missing golden %s (run with -update)", name) {
			return
		}
		goldens[name] = data
	}
	t.Cleanup(func() {
		if *updateGoldens {
			return
		}
		// Restore the goldens so a failing run does not leave the tree modified
		for name, data := range goldens {
			os.WriteFile(filepath.Join(dir, name), data, 0o644)
		}
	})

	err = run(dir, false)
	if !assert.NoError(err) {
		return
	}

	for name, want := range goldens {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if !assert.NoError(err) {
			continue
		}
		assert.NotZero(len(got), "%s is empty", name)
		if !*updateGoldens {
			assert.Equal(string(want), string(got), "%s drift (run `go test ./cmd/genworkflowmmd/ -update`)", name)
		}
	}
	_, err = os.Stat(filepath.Join(dir, "HELPER.mmd"))
	assert.True(os.IsNotExist(err), "a method that is not a graph builder must not be rendered")
	_, err = os.Stat(filepath.Join(dir, "tmp"))
	assert.True(os.IsNotExist(err), "the tmp directory must be removed")
}
