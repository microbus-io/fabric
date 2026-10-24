package banksupportapi

import "github.com/microbus-io/fabric/define"

var _ = define.None

// HINT: This file is the single source of truth for the microservice's API. After editing it, run
// cmd/genservice on the microservice's directory (the parent of this api package) to regenerate client.go,
// intermediate.go, mock.go, mock_test.go, and manifest.yaml. Do not hand-edit those generated files.

// Hostname is the default hostname of the microservice.
const Hostname = "banksupport.example"

// Name is the decorative PascalCase name of the microservice.
const Name = "BankSupport"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 4

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `BankSupport is an LLM agent that answers a signed-in customer's banking questions using their own balance and transactions as tools, gated by actor-claims authentication.`

// Login renders the login screen and, on valid credentials, mints a customer bearer token and redirects to the demo.
var Login = define.Web{ // MARKER: Login
	Host: Hostname, Method: "ANY", Route: "/login",
}

// Logout clears the customer session cookie and returns to the login screen.
var Logout = define.Web{ // MARKER: Logout
	Host: Hostname, Method: "ANY", Route: "/logout",
}

// Demo is the signed-in support console where a customer asks natural-language banking questions.
var Demo = define.Web{ // MARKER: Demo
	Host: Hostname, Method: "ANY", Route: "/demo",
	RequiredClaims: "roles.customer",
}

// Balance returns the signed-in customer's current balance. It derives the customer from the actor claim, never
// from an argument, and is exposed to the LLM as a tool.
var Balance = define.Function{ // MARKER: Balance
	Host: Hostname, Method: "ANY", Route: "/balance",
	RequiredClaims: "roles.customer",
	In:             BalanceIn{}, Out: BalanceOut{},
}

// BalanceIn are the input arguments of Balance.
type BalanceIn struct { // MARKER: Balance
}

// BalanceOut are the output arguments of Balance.
type BalanceOut struct { // MARKER: Balance
	BalanceCents int    `json:"balanceCents,omitzero" jsonschema_description:"BalanceCents is the customer's current balance in cents; negative means overdrawn"`
	Holder       string `json:"holder,omitzero" jsonschema_description:"Holder is the account holder's name"`
}

// Transactions returns the signed-in customer's transactions in a date range. It derives the customer from the
// actor claim, never from an argument, and is exposed to the LLM as a tool for spend analysis over time.
var Transactions = define.Function{ // MARKER: Transactions
	Host: Hostname, Method: "ANY", Route: "/transactions",
	RequiredClaims: "roles.customer",
	In:             TransactionsIn{}, Out: TransactionsOut{},
}

// TransactionsIn are the input arguments of Transactions.
type TransactionsIn struct { // MARKER: Transactions
	FromDate string `json:"fromDate,omitzero" jsonschema_description:"FromDate is the inclusive start of the range in YYYY-MM-DD format; empty means no lower bound"`
	ToDate   string `json:"toDate,omitzero" jsonschema_description:"ToDate is the exclusive end of the range in YYYY-MM-DD format; empty means no upper bound"`
}

// TransactionsOut are the output arguments of Transactions.
type TransactionsOut struct { // MARKER: Transactions
	Transactions []TxnView `json:"transactions,omitzero" jsonschema_description:"Transactions are the customer's ledger entries in the requested range, most recent first"`
}

/*
Support is the durable workflow that answers a customer's banking question. Its RunSupport task runs the LLM
tool-calling loop (ChatLoop) with the Balance and Transactions endpoints as tools and produces a structured
verdict: advice text, whether the card should be blocked, and a 0-10 risk score. It is a workflow rather than a
synchronous call because the multi-turn agent conversation routinely exceeds a single request's time budget;
each turn is its own durable, independently-budgeted foreman step.
*/
var Support = define.Workflow{ // MARKER: Support
	Host: Hostname, Method: "GET", Route: ":428/support",
	In: SupportIn{}, Out: SupportOut{},
}

// SupportIn are the input arguments of Support.
type SupportIn struct { // MARKER: Support
	Query string `json:"query,omitzero" jsonschema_description:"Query is the customer's natural-language banking question"`
}

// SupportOut are the output arguments of Support.
type SupportOut struct { // MARKER: Support
	Advice    string `json:"advice,omitzero" jsonschema_description:"Advice is the agent's natural-language answer"`
	BlockCard bool   `json:"blockCard,omitzero" jsonschema_description:"BlockCard is true when the agent recommends blocking the customer's card"`
	Risk      int    `json:"risk,omitzero" jsonschema_description:"Risk is a 0-10 assessment of the account's financial risk"`
}

/*
RunSupport is the single task of the Support workflow. It runs llm.core's ChatLoop as a subgraph with the
Balance and Transactions tools and parses the model's final message into the structured verdict.
*/
var RunSupport = define.Task{ // MARKER: RunSupport
	Host: Hostname, Method: "POST", Route: ":428/run-support",
	In: RunSupportIn{}, Out: RunSupportOut{},
}

// RunSupportIn are the input arguments of RunSupport.
type RunSupportIn struct { // MARKER: RunSupport
	Query string `json:"query,omitzero"`
}

// RunSupportOut are the output arguments of RunSupport.
type RunSupportOut struct { // MARKER: RunSupport
	Advice    string `json:"advice,omitzero"`
	BlockCard bool   `json:"blockCard,omitzero"`
	Risk      int    `json:"risk,omitzero"`
}

// DemoStatus long-polls a launched Support workflow via the foreman's Poll and returns the structured verdict
// when the flow has stopped, or a running status so the demo page can re-poll immediately with no client delay.
var DemoStatus = define.Function{ // MARKER: DemoStatus
	Host: Hostname, Method: "ANY", Route: "/demo-status",
	RequiredClaims: "roles.customer",
	In:             DemoStatusIn{}, Out: DemoStatusOut{},
}

// DemoStatusIn are the input arguments of DemoStatus.
type DemoStatusIn struct { // MARKER: DemoStatus
	FlowKey string `json:"flowKey,omitzero"`
}

// DemoStatusOut are the output arguments of DemoStatus.
type DemoStatusOut struct { // MARKER: DemoStatus
	Status    string `json:"status,omitzero"`
	Advice    string `json:"advice,omitzero"`
	BlockCard bool   `json:"blockCard,omitzero"`
	Risk      int    `json:"risk,omitzero"`
	ErrorMsg  string `json:"errorMsg,omitzero"`
}
