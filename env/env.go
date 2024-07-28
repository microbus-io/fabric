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

// Package env manages the loading of environment variables.
// Variables are first searched for in an in-memory stack, then in a file `env.yaml` in the current working directory, and finally in the OS.
package env

import (
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	pushed = map[string][]string{}
	mux    sync.Mutex
)

// Lookup returns the value of the environment variable.
// It looks first in the in-memory stack, then in env.yaml file, and finally in the OS variables.
// Environment value keys are case-sensitive.
func Lookup(key string) (string, bool) {
	// First, look in the stack
	mux.Lock()
	vals, ok := pushed[key]
	mux.Unlock()
	if ok && len(vals) > 0 {
		return vals[len(vals)-1], true
	}
	// Next, look in env.yaml file
	if file, err := os.Open("env.yaml"); err == nil {
		var inFile map[string]string
		if err := yaml.NewDecoder(file).Decode(&inFile); err == nil {
			if val, ok := inFile[key]; ok {
				return val, true
			}
		}
	}
	return os.LookupEnv(key)
}

// Get returns the value of the environment variable.
// It looks first in the in-memory stack, then in env.yaml file, and finally in the OS variables.
// Environment value keys are case-sensitive.
func Get(key string) string {
	val, _ := Lookup(key)
	return val
}

// Push pushes a new value to the in-memory stack.
// Pushing and popping to the stack is valuable in tests.
// Environment value keys are case-sensitive.
func Push(key string, value string) {
	mux.Lock()
	pushed[key] = append(pushed[key], value)
	mux.Unlock()
}

// Pop pops the last value pushed to the in-memory stack.
// Pushing and popping to the stack is valuable in tests.
// Environment value keys are case-sensitive.
func Pop(key string) {
	mux.Lock()
	defer mux.Unlock()
	pushed[key] = pushed[key][:len(pushed[key])-1] // Can panic if underflow
}
