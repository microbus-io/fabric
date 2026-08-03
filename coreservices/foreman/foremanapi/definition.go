package foremanapi

import (
	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/fabric/define"
	"time"
)

// HINT: This file is the single source of truth for the microservice's API. After editing it, run
// cmd/genservice on the microservice's directory (the parent of this api package) to regenerate client.go,
// intermediate.go, mock.go, mock_test.go, and manifest.yaml. Do not hand-edit those generated files.

// Hostname is the default hostname of the microservice.
const Hostname = "foreman.core"

// Name is the decorative PascalCase name of the microservice.
const Name = "Foreman"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 54

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `Foreman orchestrates agentic workflow execution.`

/*
Shards declares the database shards that hold flows and steps, one entry per shard. Each carries its own
Index (>= 1, unique, stable across restarts and identical on every replica), DSN (used verbatim, no
templating), VirtualCPUs (the CPU count of that shard's database server, which sizes its connection budget
and placement weight), and Cordoned (excludes the shard from new-flow placement while everything already
resident keeps running). Shards can be added but never removed, and a change takes effect only on restart,
after a coordinated restart of every replica. Left empty, a LOCAL deployment falls back to a single SQLite
file shard.
*/
var Shards = define.Config{ // MARKER: Shards
	Value:      []ShardSpec{},
	Default:    "[]",
	Secret:     true,
	Validation: "json",
}

// Workers is the maximum number of concurrent workers that process flow steps. Leave at -1 to let the engine derive the ceiling from each shard's connection budget and measured round-trip time. 0 stands up a replica that creates, awaits, and serves reads but never executes a task.
var Workers = define.Config{ // MARKER: Workers
	Value:      int(0),
	Default:    "-1",
	Validation: "int [-1,]",
}

// TimeBudget is the default time budget for a single task step's execution, applied as the timeout on the task dispatch call. A flow may override it per-flow via FlowOptions.TimeBudget, up to a hard 15m ceiling; a task endpoint may declare a shorter budget of its own via sub.TimeBudget.
var TimeBudget = define.Config{ // MARKER: TimeBudget
	Value:      time.Duration(0),
	Default:    "2m",
	Validation: "dur [1s,15m]",
}

// DefaultPriority is the priority assigned to a flow when the caller does not specify one. Priority is an integer >= 1, lower numbers run first.
var DefaultPriority = define.Config{ // MARKER: DefaultPriority
	Value:      int(0),
	Default:    "5",
	Validation: "int [1,]",
}

// MaxOpenConns pins every shard's connection pool to exactly this many open connections. Leave at 0 to let the engine size each pool from that shard's VirtualCPUs. Set it only when the connection budget is constrained by something the engine cannot see, such as a shared database or an external pooler.
var MaxOpenConns = define.Config{ // MARKER: MaxOpenConns
	Value:      int(0),
	Default:    "0",
	Validation: "int [0,]",
}

// Create creates a flow for a workflow and immediately runs it, returning the running flow's key. There is no separate start step. Set Opts.ThreadKey to join an existing thread; for a deferred start, have the entry task call flow.Interrupt and Resume it when ready.
var Create = define.Function{ // MARKER: Create
	Host: Hostname, Method: "ANY", Route: ":444/create",
	In: CreateIn{}, Out: CreateOut{},
}

// CreateIn are the input arguments of Create.
type CreateIn struct { // MARKER: Create
	WorkflowURL  string                `json:"workflowURL,omitzero"`
	InitialState any                   `json:"initialState,omitzero"`
	Opts         *workflow.FlowOptions `json:"opts,omitzero"`
}

// CreateOut are the output arguments of Create.
type CreateOut struct { // MARKER: Create
	FlowKey string `json:"flowKey,omitzero"`
}

// Snapshot returns the current outcome of a flow.
var Snapshot = define.Function{ // MARKER: Snapshot
	Host: Hostname, Method: "GET", Route: ":444/snapshot",
	In: SnapshotIn{}, Out: SnapshotOut{},
}

// SnapshotIn are the input arguments of Snapshot.
type SnapshotIn struct { // MARKER: Snapshot
	FlowKey string `json:"flowKey,omitzero"`
}

// SnapshotOut are the output arguments of Snapshot.
type SnapshotOut struct { // MARKER: Snapshot
	Outcome *workflow.FlowOutcome `json:"outcome,omitzero"`
}

// Fingerprint returns a short opaque hash that changes when the flow's status, step count, or any step's updated_at changes — across the flow and any nested subgraph descendants.
var Fingerprint = define.Function{ // MARKER: Fingerprint
	Host: Hostname, Method: "GET", Route: ":444/fingerprint",
	In: FingerprintIn{}, Out: FingerprintOut{},
}

// FingerprintIn are the input arguments of Fingerprint.
type FingerprintIn struct { // MARKER: Fingerprint
	FlowKey string `json:"flowKey,omitzero"`
}

// FingerprintOut are the output arguments of Fingerprint.
type FingerprintOut struct { // MARKER: Fingerprint
	Fingerprint string `json:"fingerprint,omitzero"`
	Status      string `json:"status,omitzero"`
}

// Resume continues an interrupted flow, delivering resumeData to the task that armed flow.Interrupt.
var Resume = define.Function{ // MARKER: Resume
	Host: Hostname, Method: "POST", Route: ":444/resume",
	In: ResumeIn{}, Out: ResumeOut{},
}

// ResumeIn are the input arguments of Resume.
type ResumeIn struct { // MARKER: Resume
	FlowKey    string `json:"flowKey,omitzero"`
	ResumeData any    `json:"resumeData,omitzero"`
}

// ResumeOut are the output arguments of Resume.
type ResumeOut struct { // MARKER: Resume
}

// Cancel cancels a flow that is not yet in a terminal status.
var Cancel = define.Function{ // MARKER: Cancel
	Host: Hostname, Method: "POST", Route: ":444/cancel",
	In: CancelIn{}, Out: CancelOut{},
}

// CancelIn are the input arguments of Cancel.
type CancelIn struct { // MARKER: Cancel
	FlowKey string `json:"flowKey,omitzero"`
	Reason  string `json:"reason,omitzero"`
}

// CancelOut are the output arguments of Cancel.
type CancelOut struct { // MARKER: Cancel
}

// Fork clones a terminal flow's prefix up to the given step into a new, self-contained running flow and re-executes from that step with optional stateOverrides applied to it. The original flow is never modified. The fork point may be any recorded step, including one inside a subgraph. The fork inherits the origin flow's scheduling and baggage.
var Fork = define.Function{ // MARKER: Fork
	Host: Hostname, Method: "POST", Route: ":444/fork",
	In: ForkIn{}, Out: ForkOut{},
}

// ForkIn are the input arguments of Fork.
type ForkIn struct { // MARKER: Fork
	StepKey        string `json:"stepKey,omitzero"`
	StateOverrides any    `json:"stateOverrides,omitzero"`
}

// ForkOut are the output arguments of Fork.
type ForkOut struct { // MARKER: Fork
	NewFlowKey string `json:"newFlowKey,omitzero"`
}

// History returns the step-by-step execution history of a flow.
var History = define.Function{ // MARKER: History
	Host: Hostname, Method: "GET", Route: ":444/history",
	In: HistoryIn{}, Out: HistoryOut{},
}

// HistoryIn are the input arguments of History.
type HistoryIn struct { // MARKER: History
	FlowKey string `json:"flowKey,omitzero"`
}

// HistoryOut are the output arguments of History.
type HistoryOut struct { // MARKER: History
	Steps []FlowStep `json:"steps,omitzero"`
}

// Step returns the full detail of one execution step, including the state, changes and interrupt payload that History omits.
var Step = define.Function{ // MARKER: Step
	Host: Hostname, Method: "GET", Route: ":444/step",
	In: StepIn{}, Out: StepOut{},
}

// StepIn are the input arguments of Step.
type StepIn struct { // MARKER: Step
	StepKey string `json:"stepKey,omitzero"`
}

// StepOut are the output arguments of Step.
type StepOut struct { // MARKER: Step
	Step *FlowStep `json:"step,omitzero"`
}

// List queries flows by status or workflow URL. Set Query.Cursor to the previous call's NextCursor to paginate.
var List = define.Function{ // MARKER: List
	Host: Hostname, Method: "GET", Route: ":444/list",
	In: ListIn{}, Out: ListOut{},
}

// ListIn are the input arguments of List.
type ListIn struct { // MARKER: List
	Query Query `json:"query,omitzero"`
}

// ListOut are the output arguments of List.
type ListOut struct { // MARKER: List
	Flows []FlowSummary `json:"flows,omitzero"`
	// NextCursor is the opaque pagination cursor for the next page; pass it back as Query.Cursor.
	// Empty when every shard has been drained.
	NextCursor string `json:"nextCursor,omitzero"`
}

// Delete removes a flow and its steps from the database. The flow must not be running. Subgraph and thread lineage references become dangling.
var Delete = define.Function{ // MARKER: Delete
	Host: Hostname, Method: "POST", Route: ":444/delete",
	In: DeleteIn{}, Out: DeleteOut{},
}

// DeleteIn are the input arguments of Delete.
type DeleteIn struct { // MARKER: Delete
	FlowKey string `json:"flowKey,omitzero"`
}

// DeleteOut are the output arguments of Delete.
type DeleteOut struct { // MARKER: Delete
}

// Purge deletes flows matching the query, except those currently running. Capped at 10000 flows per call.
var Purge = define.Function{ // MARKER: Purge
	Host: Hostname, Method: "POST", Route: ":444/purge",
	In: PurgeIn{}, Out: PurgeOut{},
}

// PurgeIn are the input arguments of Purge.
type PurgeIn struct { // MARKER: Purge
	Query Query `json:"query,omitzero"`
}

// PurgeOut are the output arguments of Purge.
type PurgeOut struct { // MARKER: Purge
	// Deleted is the count of flows actually deleted (excluding running flows skipped by the guard).
	Deleted int `json:"deleted,omitzero"`
}

// ShardInfo returns per-shard health (latency, row counts, error) for every database shard.
var ShardInfo = define.Function{ // MARKER: ShardInfo
	Host: Hostname, Method: "GET", Route: ":444/shard-info",
	In: ShardInfoIn{}, Out: ShardInfoOut{},
}

// ShardInfoIn are the input arguments of ShardInfo.
type ShardInfoIn struct { // MARKER: ShardInfo
}

// ShardInfoOut are the output arguments of ShardInfo.
type ShardInfoOut struct { // MARKER: ShardInfo
	Shards []ShardSummary `json:"shards,omitzero"`
}

// Await blocks until the flow stops (i.e. is no longer created, pending, or running), then returns the outcome.
var Await = define.Function{ // MARKER: Await
	Host: Hostname, Method: "POST", Route: ":444/wait-for-stop",
	In: AwaitIn{}, Out: AwaitOut{},
}

// AwaitIn are the input arguments of Await.
type AwaitIn struct { // MARKER: Await
	FlowKey string `json:"flowKey,omitzero"`
}

// AwaitOut are the output arguments of Await.
type AwaitOut struct { // MARKER: Await
	Outcome *workflow.FlowOutcome `json:"outcome,omitzero"`
}

/*
Poll returns a flow's current outcome, waiting up to the request's time budget for it to stop. Unlike Await, a
timeout is not an error: a still-running flow returns a running outcome (Outcome.Stopped() is false) so a caller
bridging an open-ended flow to a bounded request can answer within its budget and re-poll immediately.
*/
var Poll = define.Function{ // MARKER: Poll
	Host: Hostname, Method: "POST", Route: ":444/poll",
	In: PollIn{}, Out: PollOut{},
}

// PollIn are the input arguments of Poll.
type PollIn struct { // MARKER: Poll
	FlowKey string `json:"flowKey,omitzero"`
}

// PollOut are the output arguments of Poll.
type PollOut struct { // MARKER: Poll
	Outcome *workflow.FlowOutcome `json:"outcome,omitzero"`
}

// Run creates a new flow, starts it, and blocks until it stops. Returns the terminal outcome.
var Run = define.Function{ // MARKER: Run
	Host: Hostname, Method: "POST", Route: ":444/run",
	In: RunIn{}, Out: RunOut{},
}

// RunIn are the input arguments of Run.
type RunIn struct { // MARKER: Run
	WorkflowURL  string                `json:"workflowURL,omitzero"`
	InitialState any                   `json:"initialState,omitzero"`
	Opts         *workflow.FlowOptions `json:"opts,omitzero"`
}

// RunOut are the output arguments of Run.
type RunOut struct { // MARKER: Run
	Outcome *workflow.FlowOutcome `json:"outcome,omitzero"`
}

// Continue creates a new running flow from the latest completed flow in a thread, merged with additional state using the graph's reducers. The threadKey can be any flowKey belonging to the thread. The new flow belongs to the same thread and inherits its policy (priority/fairness/budget/baggage); use Create with Opts.ThreadKey to set policy explicitly instead.
var Continue = define.Function{ // MARKER: Continue
	Host: Hostname, Method: "POST", Route: ":444/continue",
	In: ContinueIn{}, Out: ContinueOut{},
}

// ContinueIn are the input arguments of Continue.
type ContinueIn struct { // MARKER: Continue
	ThreadKey       string `json:"threadKey,omitzero"`
	AdditionalState any    `json:"additionalState,omitzero"`
}

// ContinueOut are the output arguments of Continue.
type ContinueOut struct { // MARKER: Continue
	NewFlowKey string `json:"newFlowKey,omitzero"`
}

// HistoryMermaid renders an HTML page with a Mermaid diagram of the flow's execution history.
var HistoryMermaid = define.Web{ // MARKER: HistoryMermaid
	Host: Hostname, Method: "GET", Route: ":444/history-mermaid",
}

// AckTimeouts counts task dispatches that hit a 404 ack-timeout (no microservice acked the dispatch), keyed by the task endpoint and the outcome: "retry" when the foreman re-probed within the step's time budget, "giveup" when the budget horizon was spent and the step was failed. The "giveup" series is the alertable "a microservice is missing" signal. Named to parallel the framework's microbus_client_timeout_requests; the Prometheus exporter appends the _total suffix.
var AckTimeouts = define.Metric{ // MARKER: AckTimeouts
	Kind: define.Counter, Value: int(0), Labels: []string{"task_url", "outcome"},
	OTelName: "microbus_foreman_timeout_requests",
}
