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

package kitchenapi

import (
	"github.com/microbus-io/fabric/cmd/gencreds/testdata/weird/weirdapi"
	"github.com/microbus-io/fabric/define"
)

// HINT: This file is the single source of truth for the microservice's API. After editing it, run
// cmd/genservice on the microservice's directory (the parent of this api package) to regenerate client.go,
// intermediate.go, mock.go, mock_test.go, and manifest.yaml. Do not hand-edit those generated files.

// Hostname is the default hostname of the microservice.
const Hostname = "kitchen.fixture"

// Name is the decorative PascalCase name of the microservice.
const Name = "Kitchen"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 1

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `Kitchen is a gencreds fixture exercising every detected call pattern.`

// ApiKey is a plain (non-secret) identifier used by the kitchen fixture.
var ApiKey = define.Config{ // MARKER: ApiKey
	Value: string(""),
}

// Threshold caps the kitchen fixture's in-flight requests.
var Threshold = define.Config{ // MARKER: Threshold
	Value:      string(""),
	Default:    "10",
	Validation: "int [1,]",
	Callback:   true,
}

// RequestsTotal counts requests handled by the kitchen fixture, labelled by status.
var RequestsTotal = define.Metric{ // MARKER: RequestsTotal
	Kind: define.Counter, Value: int(0), Labels: []string{"status"},
	OTelName: "kitchen_requests_total",
}

// QueueDepth records the current depth of the kitchen queue.
var QueueDepth = define.Metric{ // MARKER: QueueDepth
	Kind: define.Gauge, Value: int(0),
	OTelName: "kitchen_queue_depth", Observable: true,
}

// LatencySeconds records request latency in seconds.
var LatencySeconds = define.Metric{ // MARKER: LatencySeconds
	Kind: define.Histogram, Value: float64(0),
	Buckets:  []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
	OTelName: "kitchen_latency_seconds",
}

// OnMyEvent is fired when something happens.
var OnMyEvent = define.OutboundEvent{ // MARKER: OnMyEvent
	Host: Hostname, Method: "POST", Route: ":417/on-my-event",
	In: OnMyEventIn{}, Out: OnMyEventOut{},
}

// OnMyEventIn are the input arguments of OnMyEvent.
type OnMyEventIn struct {
	Note string `json:"note,omitzero"`
}

// OnMyEventOut are the output arguments of OnMyEvent.
type OnMyEventOut struct {
	OK bool `json:"ok,omitzero"`
}

// MyFunc handles the MyFunc endpoint and exercises every call pattern.
var MyFunc = define.Function{ // MARKER: MyFunc
	Host: Hostname, Method: "POST", Route: ":443/my-func",
	In: MyFuncIn{}, Out: MyFuncOut{},
}

// MyFuncIn are the input arguments of MyFunc.
type MyFuncIn struct {
	Input string `json:"input,omitzero"`
}

// MyFuncOut are the output arguments of MyFunc.
type MyFuncOut struct {
	Output string `json:"output,omitzero"`
}

// SelfPing handles peer-to-peer self-pings.
var SelfPing = define.Function{ // MARKER: SelfPing
	Host: Hostname, Method: "POST", Route: ":444/self-ping",
	In: SelfPingIn{}, Out: SelfPingOut{},
}

// SelfPingIn are the input arguments of SelfPing.
type SelfPingIn struct{}

// SelfPingOut are the output arguments of SelfPing.
type SelfPingOut struct{}

// AltHostFn registers on a non-own host via the //host form.
var AltHostFn = define.Function{ // MARKER: AltHostFn
	Host: Hostname, Method: "GET", Route: "//alt.kitchen:0/alt-fn",
	In: AltHostFnIn{}, Out: AltHostFnOut{},
}

// AltHostFnIn are the input arguments of AltHostFn.
type AltHostFnIn struct{}

// AltHostFnOut are the output arguments of AltHostFn.
type AltHostFnOut struct{}

// OnSomething is the hook handler for weirdapi.OnSomething inbound events.
var OnSomething = define.InboundEvent{ // MARKER: OnSomething
	Source: weirdapi.OnSomething,
}
