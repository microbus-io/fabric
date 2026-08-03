package foreman

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/microbus-io/dwarf/engine"
	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/frame"

	"github.com/microbus-io/fabric/coreservices/foreman/foremanapi"
)

/*
Service implements the foreman.core microservice.

The foreman orchestrates agentic workflow execution. It is a thin Microbus adapter over an embedded
dwarf workflow engine: the engine owns all orchestration logic (scheduling, execution, fan-out/fan-in,
transitions, retries, subgraphs, breakers, backpressure), while this service translates bus endpoints to
engine calls and injects bus-flavored implementations of the engine's Host interface.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove

	// engine is the embedded dwarf workflow engine. All orchestration logic lives here; the service
	// is a thin adapter. Built and started in OnStartup, drained in OnShutdown. The service itself
	// implements engine.Host (see host.go), so it is injected via SetHost(svc).
	engine *engine.Engine
}

// OnStartup is called when the microservice is started up. It builds the dwarf engine from the current
// config, injects this service as the engine's Host plus the connector's observability providers, and
// starts it. Under the TESTING deployment it starts against isolated databases keyed by the Microbus plane
// (shared by every replica in the test app); otherwise it opens the configured shards.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	// All of the setters below are construction-time (pre-Startup), so their error returns are always nil
	// here; the real failure surfaces from Startup. The exception is SetShard, which validates the operator's
	// spec, so its error is propagated.
	testing := svc.Deployment() == connector.TESTING
	var eng *engine.Engine
	if testing {
		// The plane keys the isolated, auto-dropped test databases, so a multi-replica shared-state fixture
		// resolves to the same ones. No shard is registered: the engine opens a single in-memory shard.
		eng = engine.NewEngineUnderTest(svc.Plane())
	} else {
		eng = engine.NewEngine()
		for _, shard := range svc.resolveShards() {
			err = eng.SetShard(shard)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}
	if w := svc.Workers(); w >= 0 {
		eng.SetWorkers(w)
	}
	if n := svc.MaxOpenConns(); n > 0 {
		eng.SetMaxOpenConns(n)
	}
	eng.SetEngineID(engineID(svc.ID()))
	eng.SetTimeBudget(svc.TimeBudget())
	eng.SetDefaultPriority(svc.DefaultPriority())
	eng.SetHost(svc)
	eng.SetLogger(svc.Logger())
	eng.SetMeterProvider(svc.MeterProvider())
	eng.SetTracerProvider(svc.TracerProvider())
	svc.engine = eng

	err = eng.Startup(ctx)
	return errors.Trace(err)
}

// resolveShards returns the shard set to open, falling back to a single local SQLite file shard when a
// LOCAL deployment declares none.
func (svc *Service) resolveShards() []foremanapi.ShardSpec {
	shards := svc.Shards()
	if len(shards) == 0 && svc.Deployment() == connector.LOCAL {
		shards = []foremanapi.ShardSpec{{Index: 1, DSN: "file:shard_1.local.sqlite"}}
	}
	return shards
}

// engineID derives the engine's peer-registry identity from the microservice's instance ID. The engine
// requires a positive int64 that is unique among the replicas sharing the databases, which is exactly what
// the instance ID guarantees.
func engineID(id string) int64 {
	sum := sha256.Sum256([]byte(id))
	return int64(binary.BigEndian.Uint64(sum[:8])&math.MaxInt64) | 1
}

// OnShutdown is called when the microservice is shut down. It drains the engine (workers, timer,
// refiller) and closes its database connections.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	if svc.engine != nil {
		err = svc.engine.Shutdown(ctx)
		svc.engine = nil
	}
	return errors.Trace(err)
}

// maxTimeBudget is the ceiling on a per-flow FlowOptions.TimeBudget, rejected (not clamped) at every
// inbound flow-creating call.
const maxTimeBudget = 15 * time.Minute

// resolveOptions validates the caller's time budget, injects the caller's actor claims as opaque baggage,
// and defaults the fairness key to the caller's tenant. Every inbound flow-creating endpoint
// (Create/Run/Continue) routes through it.
func (svc *Service) resolveOptions(ctx context.Context, opts *workflow.FlowOptions) (*workflow.FlowOptions, error) {
	if opts == nil {
		opts = &workflow.FlowOptions{}
	}
	if opts.TimeBudget > maxTimeBudget {
		return nil, errors.New("time budget %s exceeds the maximum %s", opts.TimeBudget, maxTimeBudget, http.StatusBadRequest)
	}
	var claims map[string]any
	frame.Of(ctx).ParseActor(&claims)
	if len(claims) > 0 {
		opts.Baggage = claims
	}
	if opts.FairnessKey == "" {
		if tid, _ := frame.Of(ctx).Tenant(); tid != 0 {
			opts.FairnessKey = strconv.Itoa(tid)
		}
	}
	return opts, nil
}

/*
Create creates a flow for a workflow and immediately runs it, returning the running flow's key. There is no separate start step. Set Opts.ThreadKey to join an existing thread; for a deferred start, have the entry task call flow.Interrupt and Resume it when ready.
*/
func (svc *Service) Create(ctx context.Context, workflowURL string, initialState any, opts *workflow.FlowOptions) (flowKey string, err error) { // MARKER: Create
	ro, err := svc.resolveOptions(ctx, opts)
	if err != nil {
		return "", errors.Trace(err)
	}
	return svc.engine.Create(ctx, workflowURL, initialState, ro)
}

/*
Snapshot returns the current outcome of a flow.
*/
func (svc *Service) Snapshot(ctx context.Context, flowKey string) (outcome *workflow.FlowOutcome, err error) { // MARKER: Snapshot
	return svc.engine.Snapshot(ctx, flowKey)
}

/*
Fingerprint returns a short opaque hash that changes when the flow's status, step count, or any step's updated_at changes — across the flow and any nested subgraph descendants.
*/
func (svc *Service) Fingerprint(ctx context.Context, flowKey string) (fingerprint string, status string, err error) { // MARKER: Fingerprint
	return svc.engine.Fingerprint(ctx, flowKey)
}

/*
Resume continues an interrupted flow, delivering resumeData to the task that armed flow.Interrupt.
*/
func (svc *Service) Resume(ctx context.Context, flowKey string, resumeData any) (err error) { // MARKER: Resume
	return svc.engine.Resume(ctx, flowKey, resumeData)
}

/*
Cancel cancels a flow that is not yet in a terminal status.
*/
func (svc *Service) Cancel(ctx context.Context, flowKey string, reason string) (err error) { // MARKER: Cancel
	return svc.engine.Cancel(ctx, flowKey, reason)
}

/*
Fork clones a terminal flow's prefix up to the given step into a new, self-contained running flow and re-executes from that step with optional stateOverrides applied to it. The original flow is never modified. The fork point may be any recorded step, including one inside a subgraph. The fork inherits the origin flow's scheduling and baggage.
*/
func (svc *Service) Fork(ctx context.Context, stepKey string, stateOverrides any) (newFlowKey string, err error) { // MARKER: Fork
	// A fork inherits the origin flow's scheduling and baggage (so it re-runs as the original actor); it
	// takes no options, so resolveOptions does not apply here.
	return svc.engine.Fork(ctx, stepKey, stateOverrides)
}

/*
History returns the step-by-step execution history of a flow.
*/
func (svc *Service) History(ctx context.Context, flowKey string) (steps []foremanapi.FlowStep, err error) { // MARKER: History
	return svc.engine.History(ctx, flowKey)
}

/*
Step returns the full detail of one execution step, including the state, changes and interrupt payload that History omits.
*/
func (svc *Service) Step(ctx context.Context, stepKey string) (step *foremanapi.FlowStep, err error) { // MARKER: Step
	return svc.engine.Step(ctx, stepKey)
}

/*
List queries flows by status or workflow URL. Set Query.Cursor to the previous call's NextCursor to paginate.
*/
func (svc *Service) List(ctx context.Context, query foremanapi.Query) (flows []foremanapi.FlowSummary, nextCursor string, err error) { // MARKER: List
	return svc.engine.List(ctx, query)
}

/*
Delete removes a flow and its steps from the database. The flow must not be running. Subgraph and thread lineage references become dangling.
*/
func (svc *Service) Delete(ctx context.Context, flowKey string) (err error) { // MARKER: Delete
	return svc.engine.Delete(ctx, flowKey)
}

/*
Purge deletes flows matching the query, except those currently running. Capped at 10000 flows per call.
*/
func (svc *Service) Purge(ctx context.Context, query foremanapi.Query) (deleted int, err error) { // MARKER: Purge
	return svc.engine.Purge(ctx, query)
}

/*
ShardInfo returns per-shard health (latency, row counts, error) for every database shard.
*/
func (svc *Service) ShardInfo(ctx context.Context) (shards []foremanapi.ShardSummary, err error) { // MARKER: ShardInfo
	return svc.engine.ShardInfo(ctx)
}

/*
Await blocks until the flow stops (i.e. is no longer created, pending, or running), then returns the outcome.
*/
func (svc *Service) Await(ctx context.Context, flowKey string) (outcome *workflow.FlowOutcome, err error) { // MARKER: Await
	return svc.engine.Await(ctx, flowKey)
}

// pollBudgetFraction is the share of the request's remaining time budget that Poll spends awaiting the flow,
// leaving the rest as headroom to return the running outcome before the caller's own deadline fires.
const pollBudgetFraction = 4.0 / 5.0

/*
Poll returns a flow's current outcome, waiting up to the request's time budget for it to stop. Unlike Await, a
timeout is not an error: a still-running flow returns a running outcome (Outcome.Stopped() is false) so a caller
bridging an open-ended flow to a bounded request can answer within its budget and re-poll immediately.
*/
func (svc *Service) Poll(ctx context.Context, flowKey string) (outcome *workflow.FlowOutcome, err error) { // MARKER: Poll
	if deadline, ok := ctx.Deadline(); ok {
		window := time.Duration(float64(time.Until(deadline)) * pollBudgetFraction)
		if window > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, window)
			defer cancel()
		}
	}
	return svc.engine.Poll(ctx, flowKey)
}

/*
Run creates a new flow, starts it, and blocks until it stops. Returns the terminal outcome.
*/
func (svc *Service) Run(ctx context.Context, workflowURL string, initialState any, opts *workflow.FlowOptions) (outcome *workflow.FlowOutcome, err error) { // MARKER: Run
	ro, err := svc.resolveOptions(ctx, opts)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// engine.Run returns the new flow's key first; the foreman's Run endpoint does not expose it (callers
	// needing the key use Create+Await, since Create now auto-runs), so discard it here.
	_, outcome, err = svc.engine.Run(ctx, workflowURL, initialState, ro)
	return outcome, err
}

/*
Continue creates a new running flow from the latest completed flow in a thread, merged with additional state using the graph's reducers. The threadKey can be any flowKey belonging to the thread. The new flow belongs to the same thread and inherits its policy (priority/fairness/budget/baggage); use Create with Opts.ThreadKey to set policy explicitly instead.
*/
func (svc *Service) Continue(ctx context.Context, threadKey string, additionalState any) (newFlowKey string, err error) { // MARKER: Continue
	return svc.engine.Continue(ctx, threadKey, additionalState)
}

/*
HistoryMermaid renders an HTML page with a Mermaid diagram of the flow's execution history.
*/
func (svc *Service) HistoryMermaid(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: HistoryMermaid
	flowKey := r.URL.Query().Get("flowKey")
	if flowKey == "" {
		return errors.New("flowKey is required", http.StatusBadRequest)
	}

	steps, err := svc.engine.History(r.Context(), flowKey)
	if err != nil {
		return errors.Trace(err)
	}

	mmd := workflow.NewFlowRenderer(steps).WithLinks("step").Render()

	if r.URL.Query().Get("format") == "raw" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, mmd)
		return nil
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Flow History - %s</title>
<script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
<style>
body { font-family: sans-serif; margin: 2em; background: #fafafa; }
.mermaid { background: #fff; padding: 1em; border-radius: 8px; border: 1px solid #ddd; }
</style>
</head>
<body>
<pre class="mermaid">
%s
</pre>
<script>mermaid.initialize({startOnLoad:true, securityLevel:'loose'});</script>
</body>
</html>`, flowKey, mmd)
	return nil
}
