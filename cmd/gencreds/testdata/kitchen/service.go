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

package kitchen

import (
	"context"
	"net/http"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/pub"

	"github.com/microbus-io/fabric/cmd/gencreds/testdata/kitchen/kitchenapi"
	"github.com/microbus-io/fabric/cmd/gencreds/testdata/weird/weirdapi"
)

var (
	_ context.Context
	_ http.Request
	_ errors.TracedError
	_ kitchenapi.Client
	_ weirdapi.Client
)

/*
Service implements kitchen, a gencreds fixture exercising every detected call pattern.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
MyFunc handles the MyFunc endpoint and exercises every call pattern.
*/
func (svc *Service) MyFunc(ctx context.Context, input string) (output string, err error) { // MARKER: MyFunc
	// Pattern 1: Direct chain - typed Client.
	_, _ = weirdapi.NewClient(svc).Plain(ctx)

	// Pattern 2: Variable-bound Client.
	c := weirdapi.NewClient(svc)
	_ = c.PathArg(ctx, "id-1")

	// Pattern 3: Pass to a helper as a parameter.
	svc.callViaParam(ctx, c)

	// Pattern 4: ForHost("literal") - produces a per-endpoint hostname override.
	_ = weirdapi.NewClient(svc).ForHost("alt.host").GreedyArg(ctx, "tail")

	// Pattern 5: ForHost(varExpr) - produces hostname: "*" override.
	dynamicHost := input
	_ = weirdapi.NewClient(svc).ForHost(dynamicHost).PeriodInPath(ctx)

	// Pattern 6: Multicast.
	for r := range weirdapi.NewMulticastClient(svc).Plain(ctx) {
		_, _ = r.Get()
	}

	// Pattern 7: Helper-method expansion (PlainAndAck wraps Plain).
	_, _ = weirdapi.NewClient(svc).PlainAndAck(ctx)

	// Pattern 8: Self-call peer broadcast - kitchenapi.NewMulticastClient self-call.
	for r := range kitchenapi.NewMulticastClient(svc).SelfPing(ctx) {
		_ = r.Get()
	}

	// Pattern 9: Self-call with ForHost(varExpr) - dispatches to another host
	// implementing the same protocol.
	_ = kitchenapi.NewClient(svc).ForHost(dynamicHost).SelfPing(ctx)

	// Pattern 10: Outbound trigger of own event.
	for r := range kitchenapi.NewMulticastTrigger(svc).OnMyEvent(ctx, "note") {
		_, _ = r.Get()
	}

	// Pattern 11: Raw svc.Request with inline pub.X options (literal URL).
	_, _ = svc.Request(ctx, pub.GET("https://other.host/some-path"))

	// Pattern 12: Raw svc.Request with []pub.Option slice composite, then spread.
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL("https://" + dynamicHost + "/dispatched"),
	}
	_, _ = svc.Request(ctx, opts...)

	// Pattern 13: Raw svc.Request with append-built options.
	more := []pub.Option{pub.Method("PUT")}
	more = append(more, pub.URL("https://"+dynamicHost+"/appended"))
	_, _ = svc.Request(ctx, more...)

	// Pattern 14: Trust-root call on another service.
	_ = weirdapi.NewClient(svc).TrustRoot(ctx, "shutdown")

	return input + "-out", nil
}

// callViaParam exercises the parameter-typed binding pattern. The parameter
// `client` is typed as weirdapi.Client and is the receiver of an endpoint
// method call.
func (svc *Service) callViaParam(ctx context.Context, client weirdapi.Client) {
	_ = client.InternalPort(ctx)
}

/*
SelfPing handles peer-to-peer self-pings.
*/
func (svc *Service) SelfPing(ctx context.Context) (err error) { // MARKER: SelfPing
	return nil
}

/*
AltHostFn registers on a non-own host via the //host form.
*/
func (svc *Service) AltHostFn(ctx context.Context) (err error) { // MARKER: AltHostFn
	return nil
}

/*
OnSomething is the hook handler for weirdapi.OnSomething inbound events.
*/
func (svc *Service) OnSomething(ctx context.Context, detail string) (ok bool, err error) {
	return true, nil
}

/*
OnObserveQueueDepth emits the observed value of the QueueDepth metric.

QueueDepth records the current depth of the kitchen queue.
*/
func (svc *Service) OnObserveQueueDepth(ctx context.Context) (err error) { // MARKER: QueueDepth
	return svc.RecordQueueDepth(ctx, 0)
}

/*
OnChangedThreshold is called when the Threshold config property changes.

Threshold caps the kitchen fixture's in-flight requests.
*/
func (svc *Service) OnChangedThreshold(ctx context.Context) (err error) { // MARKER: Threshold
	return nil
}
