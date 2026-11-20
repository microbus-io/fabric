/*
Copyright (c) 2023-2025 Microbus LLC and various contributors

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

package transport

import (
	"fmt"
	"io"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
)

// trie is a prefix tree that keeps track of all subscriptions, supporting wildcards in the path.
// The trie data structure is safe for concurrent use.
type trie struct {
	children map[string]*trie
	parent   *trie
	queues   map[string]*ringList
	segment  string
	mux      sync.Mutex // Only used by root
}

// IsEmpty indicates if the trie is empty.
func (pt *trie) IsEmpty() bool {
	pt.mux.Lock()
	empty := len(pt.children) == 0 && len(pt.queues) == 0
	pt.mux.Unlock()
	return empty
}

// Sub sets a handler to process messages delivered to the subject.
// If a queue is specified, only one of the subscribers is passed any given message.
// If no queue is specified, all subscribers are passed all messages.
// The returned unsub function removes interest in the subject.
func (pt *trie) Sub(subject string, queue string, handler MsgHandler) (unsub func()) {
	pt.mux.Lock()
	j := pt
	for seg := range strings.SplitSeq(subject, ".") {
		if seg == "" {
			continue
		}
		next := j.children[seg]
		if next == nil {
			if j.children == nil {
				j.children = map[string]*trie{}
			}
			next = &trie{
				segment: seg,
				parent:  j,
			}
			j.children[seg] = next
		}
		j = next
	}
	if j.queues == nil {
		j.queues = map[string]*ringList{
			queue: {},
		}
	} else if j.queues[queue] == nil {
		j.queues[queue] = &ringList{}
	}
	gem := j.queues[queue].Insert(handler)
	pt.mux.Unlock()
	return func() {
		pt.unsub(j, queue, gem)
	}
}

// nodeUnsub removes interest in the subscription and trims empty nodes going up towards the root of the trie.
func (pt *trie) unsub(node *trie, queue string, gem *stone) {
	var j *trie
	pt.mux.Lock()
	ring := node.queues[queue]
	if ring.Remove(gem) && ring.IsEmpty() {
		delete(node.queues, queue)
		j = node
	}
	for j != nil {
		empty := len(j.children) == 0 && len(j.queues) == 0
		parent := j.parent
		if empty && parent != nil {
			delete(parent.children, j.segment)
		}
		if !empty {
			break
		}
		j = parent
	}
	pt.mux.Unlock()
}

// walkBuffers are the scratch slices of a trie walk.
type walkBuffers struct {
	cur    []*trie
	next   []*trie
	suffix []*trie
}

var walkPool = sync.Pool{
	New: func() any {
		return &walkBuffers{
			cur:    make([]*trie, 0, 16),
			next:   make([]*trie, 0, 16),
			suffix: make([]*trie, 0, 8),
		}
	},
}

// appendNodeHandlers selects handlers from each of the queues associated with the trie node.
// One handler is selected from each named queue.
// All handlers are selected from the unnamed queue.
// This method must be called under a lock.
func (pt *trie) appendNodeHandlers(appendTo []MsgHandler) []MsgHandler {
	for q, namedRing := range pt.queues {
		if q == "" {
			continue
		}
		head := namedRing.Rotate()
		if head == nil {
			continue // Ring is empty
		}
		appendTo = append(appendTo, head.Handler)
	}
	unnamedRing, ok := pt.queues[""]
	if ok {
		appendTo = appendAllHandlers(appendTo, unnamedRing)
	}
	return appendTo
}

// appendAllHandlers appends every handler in the ring, rotating it a full turn.
func appendAllHandlers(appendTo []MsgHandler, ring *ringList) []MsgHandler {
	head := ring.Rotate()
	if head == nil {
		return appendTo
	}
	appendTo = append(appendTo, head.Handler)
	for {
		h := ring.Rotate()
		if h == head {
			break
		}
		appendTo = append(appendTo, h.Handler)
	}
	return appendTo
}

// appendMergedHandlers selects handlers from the queues of several matching trie nodes.
// One handler is selected from each named queue across all the nodes, every member being equally likely.
// All handlers are selected from the unnamed queues.
// This method must be called under a lock.
func appendMergedHandlers(appendTo []MsgHandler, nodes []*trie) []MsgHandler {
	queues := 0
	for _, j := range nodes {
		queues += len(j.queues)
	}
	appendTo = slices.Grow(appendTo, queues)
	for i, j := range nodes {
		for q, ring := range j.queues {
			if q == "" {
				appendTo = appendAllHandlers(appendTo, ring)
				continue
			}
			if ring.IsEmpty() || queueInAny(nodes[:i], q) {
				continue // Selected from with an earlier node
			}
			// Gather the queue's members at the later nodes
			total := ring.Len()
			for _, later := range nodes[i+1:] {
				if r := later.queues[q]; r != nil {
					total += r.Len()
				}
			}
			// Pick a ring in proportion to its members, then rotate it
			chosen := ring
			if total > ring.Len() {
				pick := rand.IntN(total) - ring.Len()
				for _, later := range nodes[i+1:] {
					if pick < 0 {
						break
					}
					if r := later.queues[q]; r != nil && !r.IsEmpty() {
						chosen = r
						pick -= r.Len()
					}
				}
			}
			appendTo = append(appendTo, chosen.Rotate().Handler)
		}
	}
	return appendTo
}

// queueInAny indicates if any of the nodes has members in the named queue.
func queueInAny(nodes []*trie, queue string) bool {
	for _, j := range nodes {
		if r := j.queues[queue]; r != nil && !r.IsEmpty() {
			return true
		}
	}
	return false
}

// Handlers returns the message handlers matching the indicated subject, considering wildcards.
func (pt *trie) Handlers(subject string) (handlers []MsgHandler) {
	buf := walkPool.Get().(*walkBuffers)
	cur := append(buf.cur[:0], pt)
	next := buf.next[:0]
	suffix := buf.suffix[:0]
	pt.mux.Lock()
	for seg := range strings.SplitSeq(subject, ".") {
		if len(cur) == 0 {
			break
		}
		if seg == "" {
			continue
		}
		// A wildcard in the subject matches only through the wildcard children, as in NATS
		wildcard := seg == "*" || seg == ">"
		for _, j := range cur {
			if !wildcard {
				if segChild := j.children[seg]; segChild != nil {
					next = append(next, segChild)
				}
			}
			if starChild := j.children["*"]; starChild != nil {
				next = append(next, starChild)
			}
			if suffixChild := j.children[">"]; suffixChild != nil {
				suffix = append(suffix, suffixChild)
			}
		}
		cur, next = next, cur[:0]
	}
	switch {
	case len(cur) == 1 && len(suffix) == 0:
		handlers = cur[0].appendNodeHandlers(handlers)
	case len(cur) == 0 && len(suffix) == 1:
		handlers = suffix[0].appendNodeHandlers(handlers)
	case len(cur)+len(suffix) > 1:
		cur = append(cur, suffix...)
		handlers = appendMergedHandlers(handlers, cur)
	}
	pt.mux.Unlock()
	// Keep any array that grew for the next walk
	buf.cur, buf.next, buf.suffix = cur[:0], next[:0], suffix[:0]
	walkPool.Put(buf)
	return handlers
}

// String prints the trie to a string.
func (pt *trie) String() string {
	var sb strings.Builder
	pt.mux.Lock()
	pt.nodePrint(&sb, 0)
	pt.mux.Unlock()
	return sb.String()
}

// nodePrint prints the data of the node to a writer.
// This method must be called under a lock.
func (pt *trie) nodePrint(w io.Writer, indent int) {
	seg := pt.segment
	if seg == "" {
		seg = "~"
	}
	fmt.Fprintf(w, "%s%s", strings.Repeat(" ", indent), seg)
	if len(pt.queues) > 0 {
		fmt.Fprint(w, " {")
	}
	for q, ring := range pt.queues {
		idx := 0
		if ring != nil {
			idx = ring.index
		}
		fmt.Fprintf(w, "%d:%s", idx, q)
	}
	if len(pt.queues) > 0 {
		fmt.Fprint(w, "}")
	}
	fmt.Fprint(w, "\n")
	for _, next := range pt.children {
		next.nodePrint(w, indent+2)
	}
}
