## Initial scaffold

Create a new microservice `agentstudio` in `devservices/` with hostname `agentstudio.dev`.

Add the following web endpoints:
- At `/flows`, show the flows (sourced from `foreman.core`) in two tabs, each a matrix table with columns for
  status, the task the flow stopped at (`StoppedTaskName`, empty while running), workflow name, error, and an
  action to drill in. The "Open" tab lists the running and interrupted flows (`ListOpen`). The "Closed" tab lists
  the flows closed within a window picked from a dropdown (8 hours, 24 hours by default, 7 days, 30 days): those not
  archived yet (`ListRecentlyClosed`), then the archived ones (`ListClosed`), dropping duplicates by flow key, with
  the time each closed. Keep the open and closed lists in separate tabs. Pagination and
  sorting use the bespa Table widget. Clicking a row opens the flow detail screen.
- At `/flow/{flowKey}`, render a detail page. The top shows global properties (flow key,
  workflow URL, status, step count, created/updated times, error or terminate/cancel reason). Below,
  two tabs: "DAG" (the Mermaid diagram of the history) and "Log" (the step history as a table).
- A step's detail page has Overview, Input, Output and (when the step parked on one) Interrupt tabs. A step that
  called a subgraph also has a "Subgraph" tab: `Snapshot` its `SubgraphKey` and show the child flow's status, a
  link to its flow page, and its final state, which is what the subgraph returned.

Use the [bespa](https://github.com/microbus-io/bespa) library to render the pages, with a
`replace` directive in `go.mod` pointing at the sibling checkout.

## Flow controls

On the flow detail page, offer app-bar actions gated by the flow's status: Resume while interrupted; a graceful
Cancel (`foreman.Cancel`) while running; a forceful Terminate (`foreman.Terminate`) while running or interrupted;
Fork once the flow is terminal (`completed`, `failed`, `terminated`, or `cancelled`), and Continue
(`foreman.Continue` of this flow's key) once it is `completed`. Resume, Cancel, Terminate and Continue take a root
flow's key, so read the flow's `Summary`: on a subgraph child, hide Resume, Cancel and Terminate, disable Continue,
and show a link to the root flow among the global properties. Fork needs the root closed, so offer it on a flow
page, and on a step's page, only once the root's status (read through `Summary`) is terminal. Cancel is not
offered while interrupted because a graceful cancel leaves an interrupted flow parked. Show `terminated` alongside
the other statuses in the list filter, status chips, and dashboard charts.
