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

package middleware

import (
	"strings"

	"github.com/microbus-io/fabric/connector"
)

// Middleware returns a function that can pre or post process the request or response.
// The processor should generally call the next function in the chain.
type Middleware func(next connector.HTTPHandler) connector.HTTPHandler

type link struct {
	name string
	mw   Middleware
}

// Chain is an ordered collection of named middleware.
// The chain allows locating middleware by a name. It is advised to use a unique name for each middleware.
type Chain struct {
	links []link
}

// Append adds a middleware to the end of the chain.
func (ch *Chain) Append(name string, mw Middleware) {
	ch.links = append(ch.links, link{name: name, mw: mw})
}

// Prepend adds a middleware to the beginning of the chain.
func (ch *Chain) Prepend(name string, mw Middleware) {
	ch.links = append([]link{{name: name, mw: mw}}, ch.links...)
}

// InsertAfter inserts a middleware after the last occurrence of the named middleware, if found.
func (ch *Chain) InsertAfter(afterName string, name string, mw Middleware) (ok bool) {
	foundAt := ch.locate(afterName)
	if foundAt >= 0 {
		links := make([]link, 0, len(ch.links)+1)
		links = append(links, ch.links[:foundAt+1]...)
		links = append(links, link{name: name, mw: mw})
		links = append(links, ch.links[foundAt+1:]...)
		ch.links = links
	}
	return foundAt >= 0
}

// InsertBefore inserts a middleware before the first occurrence of the named middleware, if found.
func (ch *Chain) InsertBefore(beforeName string, name string, mw Middleware) (ok bool) {
	foundAt := ch.locate(beforeName)
	if foundAt >= 0 {
		links := make([]link, 0, len(ch.links)+1)
		links = append(links, ch.links[:foundAt]...)
		links = append(links, link{name: name, mw: mw})
		links = append(links, ch.links[foundAt:]...)
		ch.links = links
	}
	return foundAt >= 0
}

// Delete removes the first occurrence of the named middleware, if found.
func (ch *Chain) Delete(name string) (ok bool) {
	foundAt := ch.locate(name)
	if foundAt >= 0 {
		ch.links = append(ch.links[:foundAt], ch.links[foundAt+1:]...)
	}
	return foundAt >= 0
}

// Replace replaces the first occurrence of the named middleware, if found.
func (ch *Chain) Replace(name string, mw Middleware) (ok bool) {
	foundAt := ch.locate(name)
	if foundAt >= 0 {
		ch.links[foundAt] = link{name: name, mw: mw}
	}
	return foundAt >= 0
}

// Exists indicates if a middleware with the given name exists in the chain.
func (ch *Chain) Exists(name string) (ok bool) {
	return ch.locate(name) >= 0
}

// locate finds the index of the first middleware with the given name.
func (ch *Chain) locate(name string) int {
	for i := range ch.links {
		if strings.EqualFold(ch.links[i].name, name) {
			return i
		}
	}
	return -1
}

// String returns the names of the middleware in the chain, in order of their appearance.
func (ch *Chain) String() string {
	var sb strings.Builder
	for i := range ch.links {
		if i > 0 {
			sb.WriteString(" -> ")
		}
		sb.WriteString(ch.links[i].name)
	}
	return sb.String()
}

// Handlers returns the ordered list of middleware handlers.
func (ch *Chain) Handlers() (handlers []Middleware) {
	handlers = make([]Middleware, len(ch.links))
	for i := range ch.links {
		handlers[i] = ch.links[i].mw
	}
	return handlers
}

// Clear removes all middleware from the chain.
func (ch *Chain) Clear() {
	ch.links = nil
}
