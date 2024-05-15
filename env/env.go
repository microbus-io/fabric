// Package env wraps the handling of OS environment variables.
// Before looking for an environment variable in the OS, the package looks for it in an env.yaml file in the current working directory.
// In addition, it allows for pushing and popping values for testing purposes.
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
	defer mux.Unlock()
	pushed[key] = append(pushed[key], value)
}

// Pop pops the last value pushed to the in-memory stack.
// Pushing and popping to the stack is valuable in tests.
// Environment value keys are case-sensitive.
func Pop(key string) {
	mux.Lock()
	defer mux.Unlock()
	pushed[key] = pushed[key][:len(pushed[key])-1] // Can panic if underflow
}
