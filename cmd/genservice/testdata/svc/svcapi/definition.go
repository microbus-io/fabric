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

// Package svcapi is the service fixture exercising intermediate.go generation for the endpoint feature
// kinds: functions, web handlers, tasks, and workflows.
package svcapi

import (
	"net/netip"
	"time"

	"github.com/microbus-io/fabric/define"

	"github.com/microbus-io/fabric/cmd/genservice/testdata/pressuretest/srcapi"
)

// Hostname is the default hostname of the microservice.
const Hostname = "svc.example"

// Name is the decorative PascalCase name of the microservice.
const Name = "Svc"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 1

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `The svc microservice is a fixture exercising every feature kind for code generation.`

// Greet returns a greeting for a name.
var Greet = define.Function{
	Host: Hostname, Method: "POST", Route: ":443/greet",
	RequiredClaims: "roles.user",
	TimeBudget:     5 * time.Second,
	In:             GreetIn{}, Out: GreetOut{},
}

// GreetIn are the input arguments of Greet.
type GreetIn struct {
	Name string `json:"name,omitzero"`
}

// GreetOut are the output arguments of Greet.
type GreetOut struct {
	Greeting string `json:"greeting,omitzero"`
}

// Adopt registers a pet and returns the adoption time. It exercises qualification of a domain type
// (Pet -> svcapi.Pet) and an external type (time.Time) in the generated service-package files.
var Adopt = define.Function{
	Host: Hostname, Method: "POST", Route: ":443/adopt",
	In: AdoptIn{}, Out: AdoptOut{},
}

// AdoptIn are the input arguments of Adopt.
type AdoptIn struct {
	Pet Pet `json:"pet,omitzero"`
}

// AdoptOut are the output arguments of Adopt.
type AdoptOut struct {
	Since time.Time `json:"since,omitzero"`
}

// Pet is a domain type adopted by the microservice.
type Pet struct {
	Name string `json:"name,omitzero"`
	Age  int    `json:"age,omitzero"`
}

// Ping checks liveness; it takes and returns nothing.
var Ping = define.Function{
	Host: Hostname, Method: "POST", Route: ":443/ping",
	LoadBalancing: define.None,
	In:            PingIn{}, Out: PingOut{},
}

// PingIn are the input arguments of Ping.
type PingIn struct{}

// PingOut are the output arguments of Ping.
type PingOut struct{}

// Dashboard serves an HTML dashboard on any method (ANY -> 4-arg web client).
var Dashboard = define.Web{
	Host: Hostname, Method: "ANY", Route: "/dashboard",
}

// Status serves a plain status page (GET -> 2-arg web client, no body).
var Status = define.Web{
	Host: Hostname, Method: "GET", Route: "/status",
}

// Upload accepts a file upload (POST -> 3-arg web client, with body).
var Upload = define.Web{
	Host: Hostname, Method: "POST", Route: "/upload",
}

// ProcessStep is a workflow task that processes an item.
var ProcessStep = define.Task{
	Host: Hostname, Method: "POST", Route: ":428/process-step",
	In: ProcessStepIn{}, Out: ProcessStepOut{},
}

// ProcessStepIn are the input arguments of ProcessStep.
type ProcessStepIn struct {
	Item string `json:"item,omitzero"`
}

// ProcessStepOut are the output arguments of ProcessStep.
type ProcessStepOut struct {
	Done bool `json:"done,omitzero"`
}

// ReviewStep is a workflow task whose output read-modify-writes the shared "count" state field.
var ReviewStep = define.Task{
	Host: Hostname, Method: "POST", Route: ":428/review-step",
	In: ReviewStepIn{}, Out: ReviewStepOut{},
}

// ReviewStepIn are the input arguments of ReviewStep.
type ReviewStepIn struct {
	Count int `json:"count,omitzero"`
}

// ReviewStepOut are the output arguments of ReviewStep.
type ReviewStepOut struct {
	CountOut int `json:"count,omitzero"`
}

// MainFlow is the top-level workflow graph.
var MainFlow = define.Workflow{
	Host: Hostname, Method: "GET", Route: ":428/main-flow",
	In: MainFlowIn{}, Out: MainFlowOut{},
}

// MainFlowIn are the input arguments of MainFlow.
type MainFlowIn struct {
	Item string `json:"item,omitzero"`
	Pet  Pet    `json:"pet,omitzero"`
}

// MainFlowOut are the output arguments of MainFlow.
type MainFlowOut struct {
	Done  bool      `json:"done,omitzero"`
	Since time.Time `json:"since,omitzero"`
}

// OnSrcEvent handles the upstream srcapi.OnSrcEvent event.
var OnSrcEvent = define.InboundEvent{
	Source: srcapi.OnSrcEvent,
}

// OnPeerSeen fires when a peer is observed. Peer is a non-scalar field whose package (net/netip) is
// imported by no other feature, pinning that an outbound event's field types reach client.go (its
// Trigger) but never intermediate.go, which generates no handler for an outbound event.
var OnPeerSeen = define.OutboundEvent{
	Host: Hostname, Method: "POST", Route: ":417/on-peer-seen",
	In: OnPeerSeenIn{}, Out: OnPeerSeenOut{},
}

// OnPeerSeenIn are the input arguments of OnPeerSeen.
type OnPeerSeenIn struct {
	Peer netip.Addr `json:"peer,omitzero"`
}

// OnPeerSeenOut are the output arguments of OnPeerSeen.
type OnPeerSeenOut struct {
	Ack bool `json:"ack,omitzero"`
}

// RequestsTotal counts requests handled, labelled by status.
var RequestsTotal = define.Metric{
	Kind: define.Counter, Value: int(0), Labels: []string{"status"},
	OTelName: "svc_requests_total",
}

// QueueDepth records the current queue depth, observed just-in-time via OnObserveQueueDepth.
var QueueDepth = define.Metric{
	Kind: define.Gauge, Value: int(0),
	OTelName: "svc_queue_depth", Observable: true,
}

// LatencySeconds records request latency in seconds.
var LatencySeconds = define.Metric{
	Kind: define.Histogram, Value: float64(0),
	Buckets:  []float64{0.1, 0.5, 1, 5},
	OTelName: "svc_latency_seconds",
}

// APIKey is a stored credential; never logged.
var APIKey = define.Config{
	Value:  string(""),
	Secret: true,
}

// MaxItems caps the number of items processed per run; changes fire OnChangedMaxItems.
var MaxItems = define.Config{
	Value:      int(0),
	Default:    "100",
	Validation: "int [1,1000]",
	Callback:   true,
}

// DenyList is a newline-separated denylist; exercises a multi-line config default round-tripping
// through a backtick raw string (real newlines, not a literal \n) into the manifest and getter.
var DenyList = define.Config{
	Value: string(""),
	Default: `/admin
/.git
*.env`,
}

// Mascot is the pet representing the microservice; exercises a struct-valued config.
var Mascot = define.Config{
	Value:      Pet{},
	Default:    `{"name":"Rex","age":3}`,
	Validation: "json",
}

// RefreshInterval controls how often state is refreshed.
var RefreshInterval = define.Config{
	Value:      time.Duration(0),
	Default:    "1m",
	Validation: "dur (0s,24h]",
}

// Reconcile runs a periodic reconciliation.
var Reconcile = define.Ticker{
	Interval: 30 * time.Second,
}
