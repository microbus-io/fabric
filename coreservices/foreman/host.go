package foreman

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"

	"github.com/microbus-io/fabric/coreservices/accesstoken/accesstokenapi"
	"github.com/microbus-io/fabric/pub"
)

// This file implements the dwarf engine.Host interface for the Microbus transport. The engine owns the
// orchestration; these two methods are the only seam to the bus:
//   - LoadGraph   - GET the workflow graph over the bus.
//   - ExecuteTask - mint the actor token from baggage, POST the flow to the task, retry on ack-timeout.

// ackTimeoutRetryProbes is how many times a task dispatch that keeps hitting a 404 ack-timeout is re-probed
// across the step's time budget: the re-probe interval is budget/ackTimeoutRetryProbes, fixed (not
// exponential). A missing host is absent, not overloaded - probing is a cheap ack-timeout that runs no
// handler - so there is nothing to back off from; what matters is detecting its return quickly and
// uniformly. Deriving the interval from the budget keeps the cadence proportional to any horizon and needs
// no operator config: the horizon is simply the task's own time budget.
const ackTimeoutRetryProbes = 8

// LoadGraph fetches a workflow graph by issuing a GET to its URL over the bus and decoding the
// {"graph": ...} wrapper the workflow endpoint serves. Implements engine.Host.
func (svc *Service) LoadGraph(ctx context.Context, workflowURL string) (*workflow.Graph, error) {
	u := workflowURL
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	httpRes, err := svc.Request(ctx, pub.Method("GET"), pub.URL(u))
	if err != nil {
		return nil, errors.Trace(err)
	}
	var wrapper struct {
		Graph workflow.Graph `json:"graph"`
	}
	err = json.NewDecoder(httpRes.Body).Decode(&wrapper)
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = wrapper.Graph.Validate()
	if err != nil {
		return nil, errors.Trace(err, http.StatusBadRequest)
	}
	return &wrapper.Graph, nil
}

// ExecuteTask dispatches one task over the bus: it mints an actor token from the flow's baggage, POSTs
// the flow carrier to the task URL, and applies the task's returned flow back onto the carrier so the
// engine sees the changes and any control signals (goto/retry/sleep/interrupt/subgraph) the task armed.
// A transport error is returned undecorated, with one exception: a 404 ack-timeout (no microservice
// hosting the task) arms flow.Retry on the carrier and returns nil, so the engine re-probes until the
// step's time budget is spent. Every other backoff (rate-limit, availability) is owned by the task, which
// reads its own signal and arms flow.Retry. Implements engine.Host.
func (svc *Service) ExecuteTask(ctx context.Context, taskURL string, flow *workflow.Flow) error {
	// The step's time budget is the ack-timeout retry horizon. The engine bounded ctx with it (the dispatch
	// deadline); read it here, before any work consumes the dispatch, as the full budget. (Not frame.TimeBudget:
	// the engine sets a ctx deadline, not a frame header - that header only exists on the task handler's side.)
	var retryHorizon time.Duration
	if deadline, ok := ctx.Deadline(); ok {
		retryHorizon = time.Until(deadline)
	}
	actorToken, err := svc.mintActorToken(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	body, err := json.Marshal(flow)
	if err != nil {
		return errors.Trace(err)
	}
	u := taskURL
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(u),
		pub.Body(body),
		pub.ContentType("application/json"),
	}
	if actorToken != "" {
		opts = append(opts, pub.Token(actorToken))
	}
	// The engine already bounded ctx with the step's time budget; svc.Request honors that deadline.
	httpRes, err := svc.Request(ctx, opts...)
	if err != nil {
		// The task body never ran on an ack-timeout, so it cannot arm its own retry; the foreman arms it on the
		// carrier (which holds the step's creation time) and returns nil, re-probing at a fixed budget/N interval
		// until the budget horizon is spent. flow.Retry returns false once the next probe would overshoot the
		// horizon, so a permanently-missing microservice fails the step instead of looping. The ack-timeout is
		// recorded either way: "retry" while re-probing, "giveup" when the horizon is spent (the alertable
		// "a microservice is missing" signal). The engine cannot emit this - it never sees the ack-timeout.
		if isAckTimeout(err) {
			if retryHorizon > 0 && flow.Retry(retryHorizon/ackTimeoutRetryProbes, 1.0, 0, retryHorizon) {
				svc.IncrementAckTimeouts(ctx, 1, taskURL, "retry")
				return nil
			}
			svc.IncrementAckTimeouts(ctx, 1, taskURL, "giveup")
		}
		return errors.Trace(err)
	}
	respBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return errors.Trace(err)
	}
	// Unmarshal directly onto the carrier: Flow.UnmarshalJSON replaces its state, changes, and control
	// signals from the task's returned flow, in place, which is exactly what the engine reads back.
	err = json.Unmarshal(respBody, flow)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}


// isAckTimeout reports whether err is a unicast ack-timeout: the 404 the connector raises when no
// microservice acks the task dispatch (no responder), as distinct from a 404 returned by a task that
// did run. The connector phrases the ack-timeout error as "ack timeout: <canonical>".
func isAckTimeout(err error) bool {
	return errors.StatusCode(err) == http.StatusNotFound && strings.Contains(err.Error(), "ack timeout")
}

// mintActorToken mints a fresh access token from the actor claims carried in the flow's baggage (captured
// from the original caller at Create), so the downstream task runs as that actor. Returns an empty token
// when the flow has no actor claims. The iss/idp swap mirrors the legacy foreman: the minted token's
// issuer is the actor's original identity provider.
func (svc *Service) mintActorToken(ctx context.Context) (string, error) {
	baggage := workflow.BaggageFrom(ctx)
	if baggage.Len() == 0 {
		return "", nil
	}
	var actorClaims map[string]any
	err := baggage.Parse(&actorClaims)
	if err != nil {
		return "", errors.Trace(err)
	}
	iss, _ := actorClaims["iss"].(string)
	iss = stripProto(iss)
	actorClaims["iss"] = actorClaims["idp"]
	delete(actorClaims, "idp")
	token, err := accesstokenapi.NewClient(svc).ForHost(iss).Mint(ctx, actorClaims)
	if err != nil {
		return "", errors.Trace(err)
	}
	return token, nil
}

// stripProto removes the scheme (e.g. "https://") from a URL, returning the bare host/path.
func stripProto(u string) string {
	_, right, cut := strings.Cut(u, "://")
	if !cut {
		return u
	}
	return right
}
