package flightbooking

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"

	"github.com/microbus-io/fabric/coreservices/foreman/foremanapi"
	"github.com/microbus-io/fabric/coreservices/llm/llmapi"
	"github.com/microbus-io/fabric/exampleservices/flightbooking/flightbookingapi"
)

var (
	_ context.Context
	_ http.Request
	_ errors.TracedError
	_ *workflow.Flow
	_ = flightbookingapi.Hostname
)

/*
Service implements the flightbooking.example agent. It runs a durable BookFlight workflow that searches a
route, proposes candidate flights one at a time, parks on a human accept/keep-searching decision, and on
acceptance delegates seat selection to a child LLM workflow before confirming the booking.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove

	// HINT: Add member variables here
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

// routes maps a normalized "origin->destination" key to the flights serving it. Deterministic mock data keeps
// the example self-contained; a production wrapper would call a real GDS through the HTTP egress proxy.
var routes = map[string][]flightbookingapi.Flight{
	"san francisco->london": {
		{Airline: "Oceanic", FlightNo: "OA815", Origin: "San Francisco", Destination: "London", Depart: "18:30", Arrive: "12:55", Stops: 0, PriceUSD: 780, Seats: []string{"3A", "3C", "14A", "14F", "27D", "31B"}},
		{Airline: "Meridian", FlightNo: "ME221", Origin: "San Francisco", Destination: "London", Depart: "09:10", Arrive: "05:20", Stops: 1, PriceUSD: 540, Seats: []string{"7A", "7B", "18C", "22F", "40E"}},
		{Airline: "Polar", FlightNo: "PL9", Origin: "San Francisco", Destination: "London", Depart: "22:45", Arrive: "17:30", Stops: 0, PriceUSD: 910, Seats: []string{"1A", "2A", "9F", "12C"}},
	},
	"new york->paris": {
		{Airline: "Meridian", FlightNo: "ME880", Origin: "New York", Destination: "Paris", Depart: "19:00", Arrive: "08:25", Stops: 0, PriceUSD: 610, Seats: []string{"4A", "4D", "16A", "16F", "33C"}},
		{Airline: "Oceanic", FlightNo: "OA402", Origin: "New York", Destination: "Paris", Depart: "16:40", Arrive: "07:55", Stops: 1, PriceUSD: 445, Seats: []string{"11B", "11C", "25A", "38F"}},
	},
	"tokyo->sydney": {
		{Airline: "Coral", FlightNo: "CR7", Origin: "Tokyo", Destination: "Sydney", Depart: "21:20", Arrive: "08:05", Stops: 0, PriceUSD: 690, Seats: []string{"2C", "5A", "15F", "19D", "28A"}},
	},
}

// lookupRoute returns the flights for a route, matching origin and destination case-insensitively.
func lookupRoute(origin, destination string) []flightbookingapi.Flight {
	key := strings.ToLower(strings.TrimSpace(origin)) + "->" + strings.ToLower(strings.TrimSpace(destination))
	return routes[key]
}

/*
SearchFlights looks up the flights serving a route and seeds the candidate list for the workflow to propose.
*/
func (svc *Service) SearchFlights(ctx context.Context, flow *workflow.Flow, origin string, destination string) (candidates []flightbookingapi.Flight, flightIndex int, err error) { // MARKER: SearchFlights
	return lookupRoute(origin, destination), 0, nil
}

/*
ProposeFlight selects the candidate at the current index, or marks the search exhausted when the list runs out.
*/
func (svc *Service) ProposeFlight(ctx context.Context, flow *workflow.Flow, candidates []flightbookingapi.Flight, flightIndex int) (currentFlight flightbookingapi.Flight, exhausted bool, err error) { // MARKER: ProposeFlight
	if flightIndex < 0 || flightIndex >= len(candidates) {
		return flightbookingapi.Flight{}, true, nil
	}
	return candidates[flightIndex], false, nil
}

/*
AwaitDecision parks the flow on a human accept/keep-searching decision for the proposed flight and advances
to the next candidate when the traveler keeps searching.
*/
func (svc *Service) AwaitDecision(ctx context.Context, flow *workflow.Flow, currentFlight flightbookingapi.Flight, flightIndex int) (accepted bool, flightIndexOut int, err error) { // MARKER: AwaitDecision
	var resume struct {
		Accepted bool `json:"accepted"`
	}
	yield, err := flow.Interrupt(map[string]any{"flight": currentFlight, "index": flightIndex}, &resume)
	if err != nil {
		return false, flightIndex, errors.Trace(err)
	}
	if yield {
		// Parked, waiting for the demo page to Resume with the traveler's decision.
		return false, flightIndex, nil
	}
	if resume.Accepted {
		return true, flightIndex, nil
	}
	// Keep searching: advance to the next candidate and loop back to ProposeFlight.
	flow.Goto("ProposeFlight")
	return false, flightIndex + 1, nil
}

/*
ChooseSeat delegates seat selection for the accepted flight to the ChooseSeatAgent child workflow.
*/
func (svc *Service) ChooseSeat(ctx context.Context, flow *workflow.Flow, seatPreference string, currentFlight flightbookingapi.Flight) (seat string, err error) { // MARKER: ChooseSeat
	seat, yield, err := flightbookingapi.NewSubgraph(flow).ChooseSeatAgent(ctx, seatPreference, currentFlight.Seats)
	if yield {
		return "", nil
	}
	if err != nil {
		// The seat agent needs a real LLM provider. Without one, fall back to the first available seat so the
		// human-in-the-loop booking still completes end-to-end; the LLM only refines this choice.
		svc.LogWarn(ctx, "seat agent unavailable, assigning first available seat", "error", err)
		if len(currentFlight.Seats) > 0 {
			return currentFlight.Seats[0], nil
		}
		return "", nil
	}
	return seat, nil
}

/*
ConfirmBooking finalizes the booking of the accepted flight and chosen seat into a confirmation.
*/
func (svc *Service) ConfirmBooking(ctx context.Context, flow *workflow.Flow, currentFlight flightbookingapi.Flight, seat string) (confirmation string, airline string, flightNo string, err error) { // MARKER: ConfirmBooking
	confirmation = fmt.Sprintf("Booked %s %s from %s to %s departing %s, seat %s, $%d.",
		currentFlight.Airline, currentFlight.FlightNo, currentFlight.Origin, currentFlight.Destination,
		currentFlight.Depart, seat, currentFlight.PriceUSD)
	return confirmation, currentFlight.Airline, currentFlight.FlightNo, nil
}

/*
NoFlights ends the workflow with a not-booked message when no flight is available or every candidate is
declined.
*/
func (svc *Service) NoFlights(ctx context.Context, flow *workflow.Flow) (confirmation string, err error) { // MARKER: NoFlights
	return "No flight was booked. No matching flights remained to propose.", nil
}

/*
BookFlight is the top-level agentic workflow. It searches the route, then loops proposing one candidate flight
at a time and parking on a human accept/keep-searching decision via Interrupt. On acceptance it invokes the
ChooseSeatAgent child workflow as an isolated subgraph to pick a seat, then confirms the booking.
*/
func (svc *Service) BookFlight(ctx context.Context) (graph *workflow.Graph, err error) { // MARKER: BookFlight
	graph = workflow.NewGraph("BookFlight")
	graph.SetEndpoint("SearchFlights", flightbookingapi.SearchFlights.URL())
	graph.SetEndpoint("ProposeFlight", flightbookingapi.ProposeFlight.URL())
	graph.SetEndpoint("AwaitDecision", flightbookingapi.AwaitDecision.URL())
	graph.SetEndpoint("ChooseSeat", flightbookingapi.ChooseSeat.URL())
	graph.SetEndpoint("ConfirmBooking", flightbookingapi.ConfirmBooking.URL())
	graph.SetEndpoint("NoFlights", flightbookingapi.NoFlights.URL())

	graph.AddTransition("SearchFlights", "ProposeFlight")
	// First-match routing: exhausted ends the search, otherwise present the candidate for a decision.
	graph.AddTransitionSwitch("ProposeFlight", "NoFlights", "exhausted")
	graph.AddTransitionSwitch("ProposeFlight", "AwaitDecision", "true")
	// Accept takes the normal edge to seat selection; keep-searching drives flow.Goto back to ProposeFlight.
	graph.AddTransition("AwaitDecision", "ChooseSeat")
	graph.AddTransitionGoto("AwaitDecision", "ProposeFlight")
	graph.AddTransition("ChooseSeat", "ConfirmBooking")
	graph.AddTransition("ConfirmBooking", workflow.END)
	graph.AddTransition("NoFlights", workflow.END)
	return graph, nil
}

// finalAnswer returns the content of the last assistant message in a completed conversation.
func finalAnswer(items []llmapi.Item) (string, error) {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Type() == llmapi.ItemMessage && items[i].Message.Role == "assistant" {
			return items[i].Message.Content, nil
		}
	}
	return "", errors.New("no answer from the LLM")
}

// matchSeat normalizes the model's reply to one of the available seat labels, falling back to the first seat.
func matchSeat(answer string, availableSeats []string) string {
	reply := strings.ToUpper(strings.TrimSpace(answer))
	for _, s := range availableSeats {
		if strings.EqualFold(strings.TrimSpace(s), reply) {
			return s
		}
	}
	// The model may answer in a sentence; pick the first available seat label it mentions.
	for _, s := range availableSeats {
		if strings.Contains(reply, strings.ToUpper(s)) {
			return s
		}
	}
	if len(availableSeats) > 0 {
		return availableSeats[0]
	}
	return ""
}

/*
PickSeat asks the LLM to choose one seat from the available list that best matches the traveler's preference.
*/
func (svc *Service) PickSeat(ctx context.Context, flow *workflow.Flow, seatPreference string, availableSeats []string) (seat string, err error) { // MARKER: PickSeat
	items := []llmapi.Item{
		llmapi.NewMessage("system", "You are a seat-assignment assistant. Choose exactly one seat from the available list that best matches the traveler's preference. Reply with only the seat label and nothing else.").AsItem(),
		llmapi.NewMessage("user", fmt.Sprintf("Available seats: %s\nPreference: %s", strings.Join(availableSeats, ", "), seatPreference)).AsItem(),
	}
	result, _, yield, err := llmapi.NewSubgraph(flow).ChatLoop(ctx, llmapi.ProviderAny, llmapi.ModelDefault, items, nil, nil)
	if yield {
		return "", nil
	}
	if err != nil {
		return "", errors.Trace(err)
	}
	answer, err := finalAnswer(result)
	if err != nil {
		return "", errors.Trace(err)
	}
	return matchSeat(answer, availableSeats), nil
}

/*
ChooseSeatAgent is a child workflow that selects a single seat from a flight's available seats to match a
natural-language preference. It is invoked as an isolated subgraph by the BookFlight workflow's ChooseSeat task.
*/
func (svc *Service) ChooseSeatAgent(ctx context.Context) (graph *workflow.Graph, err error) { // MARKER: ChooseSeatAgent
	graph = workflow.NewGraph("ChooseSeatAgent")
	graph.SetEndpoint("PickSeat", flightbookingapi.PickSeat.URL())
	graph.AddTransition("PickSeat", workflow.END)
	return graph, nil
}

// demoStep holds the data for a single step row in the demo template.
type demoStep struct {
	TaskName string
	Status   string
	Changes  string
	Indent   bool
}

// flattenSteps flattens the step history into a list of demoStep structs, indenting subgraph steps.
func flattenSteps(steps []foremanapi.FlowStep, indent bool) []demoStep {
	var result []demoStep
	for _, s := range steps {
		changes := ""
		if s.Changes.Len() > 0 {
			b, _ := json.Marshal(s.Changes)
			changes = string(b)
		}
		taskName := s.TaskName
		if i := strings.LastIndex(taskName, "/"); i >= 0 {
			taskName = taskName[i+1:]
		}
		result = append(result, demoStep{TaskName: taskName, Status: s.Status, Changes: changes, Indent: indent})
		if len(s.SubHistory) > 0 {
			result = append(result, flattenSteps(s.SubHistory, true)...)
		}
	}
	return result
}

// proposedFlight decodes the flight the flow is currently parked on from its interrupt payload.
func proposedFlight(outcome *workflow.FlowOutcome) (flightbookingapi.Flight, bool) {
	if outcome == nil {
		return flightbookingapi.Flight{}, false
	}
	var f flightbookingapi.Flight
	ok, err := outcome.InterruptPayload.Get("flight", &f)
	if err != nil || !ok {
		return flightbookingapi.Flight{}, false
	}
	return f, true
}

// demoData is the view model rendered by the demo template.
type demoData struct {
	Origin         string
	Destination    string
	SeatPreference string
	Submitted      bool
	Error          string
	Status         string
	FlowKey        string
	Proposed       bool
	Flight         flightbookingapi.Flight
	Booked         bool
	Confirmation   string
	Seat           string
	Steps          []demoStep
	MermaidDiagram string
}

// loadHistory fills the step history and Mermaid diagram for a flow into the view model.
func (svc *Service) loadHistory(ctx context.Context, foremanClient foremanapi.Client, flowKey string, data *demoData) {
	svc.Parallel(ctx,
		func(ctx context.Context) error {
			steps, err := foremanClient.History(ctx, flowKey)
			if err == nil {
				data.Steps = flattenSteps(steps, false)
			}
			return nil
		},
		func(ctx context.Context) error {
			res, err := foremanClient.HistoryMermaid(ctx, "?flowKey="+flowKey+"&format=raw")
			if err == nil {
				defer res.Body.Close()
				b, _ := io.ReadAll(res.Body)
				data.MermaidDiagram = string(b)
			}
			return nil
		},
	)
}

// render decodes a stopped flow's outcome into the view model.
func (svc *Service) render(ctx context.Context, foremanClient foremanapi.Client, flowKey string, outcome *workflow.FlowOutcome, data *demoData) {
	data.FlowKey = flowKey
	if outcome != nil {
		data.Status = outcome.Status
	}
	switch {
	case outcome != nil && outcome.Status == "interrupted":
		if f, ok := proposedFlight(outcome); ok {
			data.Proposed = true
			data.Flight = f
		}
	case outcome != nil && outcome.Status == "completed":
		var out flightbookingapi.BookFlightOut
		if b, err := json.Marshal(outcome.State); err == nil {
			_ = json.Unmarshal(b, &out)
		}
		data.Booked = out.Airline != ""
		data.Confirmation = out.Confirmation
		data.Seat = out.Seat
	case outcome != nil && outcome.Status == "failed":
		data.Error = outcome.Error
	}
	svc.loadHistory(ctx, foremanClient, flowKey, data)
}

/*
Demo serves the human-in-the-loop demo page that drives the BookFlight workflow, presenting each proposed
flight and resuming the parked flow with the traveler's accept or keep-searching decision.
*/
func (svc *Service) Demo(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: Demo
	ctx := r.Context()
	err = r.ParseForm()
	if err != nil {
		return errors.Trace(err, http.StatusBadRequest)
	}

	data := demoData{
		Origin:         r.FormValue("origin"),
		Destination:    r.FormValue("destination"),
		SeatPreference: r.FormValue("seatPreference"),
	}
	if r.Method == "GET" && data.Origin == "" {
		data.Origin = "San Francisco"
		data.Destination = "London"
		data.SeatPreference = "a window seat near the front"
	}

	if r.Method == "POST" {
		data.Submitted = true
		foremanClient := foremanapi.NewClient(svc)
		flowKey := r.FormValue("flowKey")
		var outcome *workflow.FlowOutcome
		if flowKey == "" {
			// Start a new booking flow; it parks on the first proposed flight.
			in := flightbookingapi.BookFlightIn{Origin: data.Origin, Destination: data.Destination, SeatPreference: data.SeatPreference}
			flowKey, err = foremanClient.Create(ctx, flightbookingapi.BookFlight.URL(), in, nil)
			if err != nil {
				data.Error = fmt.Sprintf("%+v", err)
			} else {
				outcome, err = foremanClient.Await(ctx, flowKey)
				if err != nil {
					data.Error = fmt.Sprintf("%+v", err)
				}
			}
		} else {
			// Resume the parked flow with the traveler's accept/keep-searching decision.
			accepted := r.FormValue("decision") == "accept"
			err = foremanClient.Resume(ctx, flowKey, map[string]any{"accepted": accepted})
			if err != nil {
				data.Error = fmt.Sprintf("%+v", err)
			} else {
				outcome, err = foremanClient.Await(ctx, flowKey)
				if err != nil {
					data.Error = fmt.Sprintf("%+v", err)
				}
			}
		}
		if data.Error == "" {
			svc.render(ctx, foremanClient, flowKey, outcome, &data)
		}
	}

	w.Header().Set("Content-Type", "text/html")
	err = svc.WriteResTemplate(w, "demo.html", data)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
