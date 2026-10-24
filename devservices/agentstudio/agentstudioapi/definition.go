package agentstudioapi

import (
	"github.com/microbus-io/fabric/define"
)

// HINT: This file is the single source of truth for the microservice's API. After editing it, run
// cmd/genservice on the microservice's directory (the parent of this api package) to regenerate client.go,
// intermediate.go, mock.go, mock_test.go, and manifest.yaml. Do not hand-edit those generated files.

// Hostname is the default hostname of the microservice.
const Hostname = "agentstudio.dev"

// Name is the decorative PascalCase name of the microservice.
const Name = "AgentStudio"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 10

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `AgentStudio is a developer console for inspecting flows running under the Foreman.`

// ListFlows renders an HTML page with a paginated, sortable table of flows.
var ListFlows = define.Web{ // MARKER: ListFlows
	Host: Hostname, Method: "ANY", Route: "/flows",
}

// FlowDetail renders an HTML page with the details, DAG diagram, and step log of a flow.
var FlowDetail = define.Web{ // MARKER: FlowDetail
	Host: Hostname, Method: "ANY", Route: "/flows/{flowKey}",
}

// StepDetail renders an HTML page with the details of one execution step.
var StepDetail = define.Web{ // MARKER: StepDetail
	Host: Hostname, Method: "ANY", Route: "/steps/{stepKey}",
}

// ListWorkflows renders an HTML page listing the workflows available in the system.
var ListWorkflows = define.Web{ // MARKER: ListWorkflows
	Host: Hostname, Method: "ANY", Route: "/workflows",
}

// WorkflowDetail renders an HTML page with the structure and definition of a single workflow graph.
var WorkflowDetail = define.Web{ // MARKER: WorkflowDetail
	Host: Hostname, Method: "ANY", Route: "/workflows/{workflowURL...}",
}

// RunWorkflow renders a form to create and start a workflow with an initial state and FlowOptions.
var RunWorkflow = define.Web{ // MARKER: RunWorkflow
	Host: Hostname, Method: "ANY", Route: "/run-workflow",
}

// ContinueFlow renders a form to continue a completed flow's thread with additional state.
var ContinueFlow = define.Web{ // MARKER: ContinueFlow
	Host: Hostname, Method: "ANY", Route: "/continue-flow",
}

// ResumeFlow renders a form to resume an interrupted flow with a resume payload.
var ResumeFlow = define.Web{ // MARKER: ResumeFlow
	Host: Hostname, Method: "ANY", Route: "/resume-flow",
}

// ForkFromStep renders a form to fork a terminal flow from a specific recorded step into a new flow with optional state overrides.
var ForkFromStep = define.Web{ // MARKER: ForkFromStep
	Host: Hostname, Method: "ANY", Route: "/fork-from-step",
}

// PollFlow returns a JSON status payload driving the FlowDetail live-update progress bar.
var PollFlow = define.Web{ // MARKER: PollFlow
	Host: Hostname, Method: "GET", Route: "/poll-flow",
}

// TaskDetail renders an HTML page with the metadata of a single task in a workflow graph.
var TaskDetail = define.Web{ // MARKER: TaskDetail
	Host: Hostname, Method: "ANY", Route: "/task-detail",
}

// Dashboard renders an HTML page with operator dashboards for flows and workflows.
var Dashboard = define.Web{ // MARKER: Dashboard
	Host: Hostname, Method: "ANY", Route: "/dashboard",
}

// Assets serves the bespa CSS and JavaScript assets at /bespa/.
var Assets = define.Web{ // MARKER: Assets
	Host: Hostname, Method: "GET", Route: "//bespa/{path...}",
}
