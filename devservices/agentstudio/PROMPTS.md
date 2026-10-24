## Initial scaffold

Create a new microservice `agentstudio` in `devservices/` with hostname `agentstudio.dev`.

Add the following web endpoints:
- At `/`, show a list of flows (sourced from `foreman.core`) in a matrix table with columns for
  status, the task the flow stopped at (`StoppedTaskName`, empty while running), workflow name, error, and an
  action to drill in. Pagination and sorting
  use the bespa Table widget. Clicking a row opens the flow detail screen.
- At `/flow/{flowKey}`, render a detail page. The top shows global properties (flow key, thread
  key, workflow URL, status, step count, created/updated times, error or terminate/cancel reason). Below,
  two tabs: "DAG" (the Mermaid diagram of the history) and "Log" (the step history as a table).
- A step's detail page has Overview, Input, Output and (when the step parked on one) Interrupt tabs. A step that
  called a subgraph also has a "Subgraph" tab: `Snapshot` its `SubgraphKey` and show the child flow's status, a
  link to its flow page, and its final state, which is what the subgraph returned.

Use the [bespa](https://github.com/microbus-io/bespa) library to render the pages, with a
`replace` directive in `go.mod` pointing at the sibling checkout.

## Flow controls

On the flow detail page, offer app-bar actions gated by the flow's status: Resume while interrupted; a graceful
Cancel (`foreman.Cancel`) while running; a forceful Terminate (`foreman.Terminate`) while running or interrupted;
Continue and Fork once the flow is terminal (`completed`, `failed`, `terminated`, or `cancelled`). Cancel is not
offered while interrupted because a graceful cancel leaves an interrupted flow parked. Show `terminated` alongside
the other statuses in the list filter, status chips, and dashboard charts.
