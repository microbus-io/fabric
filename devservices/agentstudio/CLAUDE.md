# agentstudio.dev

AgentStudio is a developer-facing console for inspecting flows running under the Foreman.
It renders pages with the [bespa](https://github.com/microbus-io/bespa) backend-SPA library.

## Flow diagram color tokens

The flow execution-history diagram on the FlowDetail page is rendered by `foremanapi.FlowRenderer`.
AgentStudio passes BESPA Material Design 3 palette references as CSS `var()` color values directly to
the renderer:

```go
WithPrimaryColors("rgb(var(--md-sys-color-primary-container))",   "rgb(var(--md-sys-color-on-primary-container))")
WithSecondaryColors("rgb(var(--md-sys-color-secondary-container))", "rgb(var(--md-sys-color-on-secondary-container))")
WithErrorColors("rgb(var(--md-sys-color-error-container))",       "rgb(var(--md-sys-color-on-error-container))")
WithAttentionColors("rgb(var(--md-sys-color-tertiary-container))", "rgb(var(--md-sys-color-on-tertiary-container))")
```

The bespa mermaid widget's var-expansion pass (`bespa/mermaid/varexpand.go`) detects the `var()`
references inside the renderer's classDef and style directives, rewrites the values to
`currentColor`, and emits bridge CSS rules that resolve the original references through the host
page's cascade. The diagram tracks the BESPA theme live, re-coloring on a light/dark toggle without
any work in AgentStudio.

The var-expansion exists because Mermaid's flowchart parser tokenizes classDef values with a narrow
grammar that rejects CSS function syntax. Without the bridge, every `var()` reference would fail
parsing with a `PS` (paren-start) token error. The widget hides that grammar quirk so callers can use
CSS variables anywhere they would in a normal stylesheet.

The status nodes use the MD3 `*-container` tonal variants — softer brand tones paired with the
`on-*-container` text color, which is the MD3 convention for badges and chips. `completed`/`running`
specifically use `primary-container` rather than the `ok`/`on-ok` success extension, because completed
is the dominant state in a happy-path flow and should read as the brand voice rather than as a
"success" green.

## Live updates

The FlowDetail page tracks an in-flight flow through a bespa `Progress` widget that long-polls
`PollFlow`. The bar is purely a transport for the polling loop — its visual role is the spinner
while work is in flight, gone the moment the flow stops.

**Long-poll fingerprint.** `PollFlow` blocks up to `pollFlowMaxWait` (15s, comfortably under the
HTTP ingress' 20s `TimeBudget`) and only returns early when a per-flow fingerprint changes against
the caller's `since=…` query value. The fingerprint is `computePollToken(status, history)` —
`status + flatStepCount + maxStepUpdatedAtMillis`, walked recursively through `SubHistory`. It
changes on the three things that actually move the diagram: flow status, a new step, or any
step transition (which advances the step's `UpdatedAt`). Activity that leaves every step's
`UpdatedAt` unchanged does not alter the rendered graph, so it does not wake the poll. Idle ticks
sleep `pollFlowTick` (250ms) between snapshots and bail immediately on
`r.Context().Done()` when the browser tab closes.

**Baseline ordering invariant.** `FlowDetail` captures the `since` baseline (a `Fingerprint`
call) *before* the `Snapshot`/`History` reads that render the graph — not after. The render and the
baseline are two separate reads of a live flow, so if the flow completes between them and the
baseline is taken last, the first `PollFlow` finds its fingerprint already equal to `since`, never
appends `flowrefresh`, and the graph is left frozen on the last-rendered (often a `pending` step)
state until a manual refresh. Taking the baseline first guarantees `since` is never newer than the
rendered graph: a stale render always differs from the next poll, so `flowrefresh` fires and the
diagram converges. The cost is a possible harmless redundant redraw when the flow finishes inside
the tiny `Fingerprint`→`History` window (the graph is already current; it just repaints once). This
is why forking from a near-terminal step used to stick — the short remaining run finished before the
late baseline was read — while forking the entry step did not.

**Two state vars, two redraw scopes.** The action URL returned to the client carries two
independent pieces:

- `flowrefresh=<token>` — appended whenever the fingerprint differs. The mermaid widget and each
  appbar button bind `RedrawIfChanged(r, "flowrefresh")` so they swap on every real change. This
  is the high-cadence channel; it can fire many times during a running flow.
- `flowstopped=1` — appended only on the terminal-state transition (and never cleared). The
  progress widget binds `HideIfEq(r, "flowstopped", "1").RedrawIfChanged(r, "flowstopped")` so the
  bar is left untouched across the noisy stream of `flowrefresh` updates and only redraws once —
  to the empty placeholder — when the flow stops. This is what eliminated the bar's per-poll
  flash.

**Initial render seeding.** A flow opened in a terminal state has `flowstopped=1` written into the
page's state at render time, so `HideIfEq` immediately resolves the progress widget to a
`Tag.When(false)` empty placeholder. No spurious first poll and no transient flicker on
already-finished flows.

**Fork path.** A terminal flow is immutable, so recovery is a Fork: the Fork button (flow-level,
forking from the entry step) and the per-step Fork affordance both call `foreman.Fork`, which spawns a
new running flow and redirects to that new flow's detail page. The redirect is a full navigation, not a
partial redraw, so the fresh server-side render builds the new flow's page from scratch with
`flowstopped` unset and the bar live again, with no explicit un-hide path needed. The original flow's
page is never mutated.

**Bespa swap mechanics, briefly.** `page_applyRedrawnElements` swaps elements by `data-id`. A
hidden widget still emits `<span class="Empty" data-id="X">` (see `widget/tag.go` `Tag.When(false)`),
so a widget that flips from visible to hidden swaps the original DOM with an empty placeholder of
the same id — that's how the progress bar visibly disappears. A widget without `RedrawIfChanged`
on a particular state key is not emitted on a redraw triggered by that key, and the existing DOM
stays in place. The mermaid widget therefore only repaints on `flowrefresh` changes; appbar
buttons only swap on `flowrefresh` changes; the progress bar only swaps on `flowstopped`
changes.
