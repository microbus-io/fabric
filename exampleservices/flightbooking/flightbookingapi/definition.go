package flightbookingapi

import "github.com/microbus-io/fabric/define"

var _ = define.None

// HINT: This file is the single source of truth for the microservice's API. After editing it, run
// cmd/genservice on the microservice's directory (the parent of this api package) to regenerate client.go,
// intermediate.go, mock.go, mock_test.go, and manifest.yaml. Do not hand-edit those generated files.

// Hostname is the default hostname of the microservice.
const Hostname = "flightbooking.example"

// Name is the decorative PascalCase name of the microservice.
const Name = "FlightBooking"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 2

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `FlightBooking is an example agentic workflow that searches flights, pauses for a human accept/keep-searching decision via Interrupt/Resume, and delegates seat selection to a child LLM workflow.`

// Demo serves the human-in-the-loop demo page that drives the BookFlight workflow, presenting each proposed
// flight and resuming the parked flow with the traveler's accept or keep-searching decision.
var Demo = define.Web{ // MARKER: Demo
	Host: Hostname, Method: "ANY", Route: "/demo",
}

// SearchFlights looks up the flights serving a route and seeds the candidate list for the workflow to propose.
var SearchFlights = define.Task{ // MARKER: SearchFlights
	Host: Hostname, Method: "POST", Route: ":428/search-flights",
	In: SearchFlightsIn{}, Out: SearchFlightsOut{},
}

// SearchFlightsIn are the input arguments of SearchFlights.
type SearchFlightsIn struct { // MARKER: SearchFlights
	Origin      string `json:"origin,omitzero"`
	Destination string `json:"destination,omitzero"`
}

// SearchFlightsOut are the output arguments of SearchFlights.
type SearchFlightsOut struct { // MARKER: SearchFlights
	Candidates  []Flight `json:"candidates,omitzero"`
	FlightIndex int      `json:"flightIndex,omitzero"`
}

// ProposeFlight selects the candidate at the current index, or marks the search exhausted when the list runs out.
var ProposeFlight = define.Task{ // MARKER: ProposeFlight
	Host: Hostname, Method: "POST", Route: ":428/propose-flight",
	In: ProposeFlightIn{}, Out: ProposeFlightOut{},
}

// ProposeFlightIn are the input arguments of ProposeFlight.
type ProposeFlightIn struct { // MARKER: ProposeFlight
	Candidates  []Flight `json:"candidates,omitzero"`
	FlightIndex int      `json:"flightIndex,omitzero"`
}

// ProposeFlightOut are the output arguments of ProposeFlight.
type ProposeFlightOut struct { // MARKER: ProposeFlight
	CurrentFlight Flight `json:"currentFlight,omitzero"`
	Exhausted     bool   `json:"exhausted,omitzero"`
}

// AwaitDecision parks the flow on a human accept/keep-searching decision for the proposed flight and advances
// to the next candidate when the traveler keeps searching.
var AwaitDecision = define.Task{ // MARKER: AwaitDecision
	Host: Hostname, Method: "POST", Route: ":428/await-decision",
	In: AwaitDecisionIn{}, Out: AwaitDecisionOut{},
}

// AwaitDecisionIn are the input arguments of AwaitDecision.
type AwaitDecisionIn struct { // MARKER: AwaitDecision
	CurrentFlight Flight `json:"currentFlight,omitzero"`
	FlightIndex   int    `json:"flightIndex,omitzero"`
}

// AwaitDecisionOut are the output arguments of AwaitDecision.
type AwaitDecisionOut struct { // MARKER: AwaitDecision
	Accepted       bool `json:"accepted,omitzero"`
	FlightIndexOut int  `json:"flightIndex,omitzero"`
}

// ChooseSeat delegates seat selection for the accepted flight to the ChooseSeatAgent child workflow.
var ChooseSeat = define.Task{ // MARKER: ChooseSeat
	Host: Hostname, Method: "POST", Route: ":428/choose-seat",
	In: ChooseSeatIn{}, Out: ChooseSeatOut{},
}

// ChooseSeatIn are the input arguments of ChooseSeat.
type ChooseSeatIn struct { // MARKER: ChooseSeat
	SeatPreference string `json:"seatPreference,omitzero"`
	CurrentFlight  Flight `json:"currentFlight,omitzero"`
}

// ChooseSeatOut are the output arguments of ChooseSeat.
type ChooseSeatOut struct { // MARKER: ChooseSeat
	Seat string `json:"seat,omitzero"`
}

// ConfirmBooking finalizes the booking of the accepted flight and chosen seat into a confirmation.
var ConfirmBooking = define.Task{ // MARKER: ConfirmBooking
	Host: Hostname, Method: "POST", Route: ":428/confirm-booking",
	In: ConfirmBookingIn{}, Out: ConfirmBookingOut{},
}

// ConfirmBookingIn are the input arguments of ConfirmBooking.
type ConfirmBookingIn struct { // MARKER: ConfirmBooking
	CurrentFlight Flight `json:"currentFlight,omitzero"`
	Seat          string `json:"seat,omitzero"`
}

// ConfirmBookingOut are the output arguments of ConfirmBooking.
type ConfirmBookingOut struct { // MARKER: ConfirmBooking
	Confirmation string `json:"confirmation,omitzero"`
	Airline      string `json:"airline,omitzero"`
	FlightNo     string `json:"flightNo,omitzero"`
}

// NoFlights ends the workflow with a not-booked message when no flight is available or every candidate is
// declined.
var NoFlights = define.Task{ // MARKER: NoFlights
	Host: Hostname, Method: "POST", Route: ":428/no-flights",
	In: NoFlightsIn{}, Out: NoFlightsOut{},
}

// NoFlightsIn are the input arguments of NoFlights.
type NoFlightsIn struct { // MARKER: NoFlights
}

// NoFlightsOut are the output arguments of NoFlights.
type NoFlightsOut struct { // MARKER: NoFlights
	Confirmation string `json:"confirmation,omitzero"`
}

// PickSeat asks the LLM to choose one seat from the available list that best matches the traveler's preference.
var PickSeat = define.Task{ // MARKER: PickSeat
	Host: Hostname, Method: "POST", Route: ":428/pick-seat",
	In: PickSeatIn{}, Out: PickSeatOut{},
}

// PickSeatIn are the input arguments of PickSeat.
type PickSeatIn struct { // MARKER: PickSeat
	SeatPreference string   `json:"seatPreference,omitzero"`
	AvailableSeats []string `json:"availableSeats,omitzero"`
}

// PickSeatOut are the output arguments of PickSeat.
type PickSeatOut struct { // MARKER: PickSeat
	Seat string `json:"seat,omitzero"`
}

/*
BookFlight is the top-level agentic workflow. It searches the route, then loops proposing one candidate flight
at a time and parking on a human accept/keep-searching decision via Interrupt. On acceptance it invokes the
ChooseSeatAgent child workflow as an isolated subgraph to pick a seat, then confirms the booking.
*/
var BookFlight = define.Workflow{ // MARKER: BookFlight
	Host: Hostname, Method: "GET", Route: ":428/book-flight",
	In: BookFlightIn{}, Out: BookFlightOut{},
}

// BookFlightIn are the input arguments of BookFlight.
type BookFlightIn struct { // MARKER: BookFlight
	Origin         string `json:"origin,omitzero" jsonschema_description:"Origin is the departure city, e.g. San Francisco"`
	Destination    string `json:"destination,omitzero" jsonschema_description:"Destination is the arrival city, e.g. London"`
	SeatPreference string `json:"seatPreference,omitzero" jsonschema_description:"SeatPreference is a natural-language seat request, e.g. a window seat near the front"`
}

// BookFlightOut are the output arguments of BookFlight.
type BookFlightOut struct { // MARKER: BookFlight
	Confirmation string `json:"confirmation,omitzero" jsonschema_description:"Confirmation is the human-readable booking confirmation, or a not-booked message"`
	Airline      string `json:"airline,omitzero" jsonschema_description:"Airline is the booked flight's airline"`
	FlightNo     string `json:"flightNo,omitzero" jsonschema_description:"FlightNo is the booked flight number"`
	Seat         string `json:"seat,omitzero" jsonschema_description:"Seat is the assigned seat label"`
}

/*
ChooseSeatAgent is a child workflow that selects a single seat from a flight's available seats to match a
natural-language preference. It is invoked as an isolated subgraph by the BookFlight workflow's ChooseSeat task.
*/
var ChooseSeatAgent = define.Workflow{ // MARKER: ChooseSeatAgent
	Host: Hostname, Method: "GET", Route: ":428/choose-seat-agent",
	In: ChooseSeatAgentIn{}, Out: ChooseSeatAgentOut{},
}

// ChooseSeatAgentIn are the input arguments of ChooseSeatAgent.
type ChooseSeatAgentIn struct { // MARKER: ChooseSeatAgent
	SeatPreference string   `json:"seatPreference,omitzero" jsonschema_description:"SeatPreference is a natural-language seat request, e.g. an aisle seat with legroom"`
	AvailableSeats []string `json:"availableSeats,omitzero" jsonschema_description:"AvailableSeats are the seat labels the traveler may choose from, e.g. 1A, 14C, 22F"`
}

// ChooseSeatAgentOut are the output arguments of ChooseSeatAgent.
type ChooseSeatAgentOut struct { // MARKER: ChooseSeatAgent
	Seat string `json:"seat,omitzero" jsonschema_description:"Seat is the chosen seat label"`
}
