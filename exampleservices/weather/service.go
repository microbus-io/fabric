package weather

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"strings"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"

	"github.com/microbus-io/fabric/coreservices/llm/llmapi"
	"github.com/microbus-io/fabric/exampleservices/weather/weatherapi"
)

var (
	_ context.Context
	_ http.Request
	_ errors.TracedError
	_ *workflow.Flow
	_ = weatherapi.Hostname
)

/*
Service implements the weather.example agent, which answers natural-language weather questions.
It geocodes a location to coordinates and fetches that location's forecast, exposing both as LLM tools
and letting the model chain them to answer a question.
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

// coordinates of a handful of well-known cities, so common questions geocode to real locations
var knownCities = map[string]struct{ lat, lng float64 }{
	"san francisco":  {37.77, -122.42},
	"new york":       {40.71, -74.01},
	"london":         {51.51, -0.13},
	"paris":          {48.85, 2.35},
	"berlin":         {52.52, 13.40},
	"tokyo":          {35.68, 139.69},
	"sydney":         {-33.87, 151.21},
	"cairo":          {30.04, 31.24},
	"rio de janeiro": {-22.91, -43.17},
	"reykjavik":      {64.15, -21.94},
}

/*
LatLng geocodes a location name into its latitude and longitude coordinates.
*/
func (svc *Service) LatLng(ctx context.Context, location string) (lat float64, lng float64, err error) { // MARKER: LatLng
	key := strings.ToLower(strings.TrimSpace(location))
	// Trim a trailing country/region qualifier, e.g. "London, UK" -> "london".
	if comma := strings.IndexByte(key, ','); comma >= 0 {
		key = strings.TrimSpace(key[:comma])
	}
	if c, ok := knownCities[key]; ok {
		return c.lat, c.lng, nil
	}
	// Any other place geocodes to a deterministic pseudo-coordinate, keeping the demo self-contained.
	h := fnv.New64a()
	h.Write([]byte(key))
	sum := h.Sum64()
	lat = math.Round((float64(sum%13000)/100.0-60.0)*100) / 100 // -60.00 .. 70.00
	lng = math.Round((float64((sum>>20)%36000)/100.0-180.0)*100) / 100
	return lat, lng, nil
}

/*
Forecast returns the current weather forecast for a latitude/longitude coordinate.
*/
func (svc *Service) Forecast(ctx context.Context, lat float64, lng float64) (summary string, temperatureC float64, precipitationChance int, err error) { // MARKER: Forecast
	h := fnv.New64a()
	fmt.Fprintf(h, "%.2f,%.2f", lat, lng)
	sum := h.Sum64()
	conditions := []string{
		"Sunny with light winds",
		"Partly cloudy",
		"Overcast",
		"Light rain",
		"Heavy rain showers",
		"Clear and cold",
		"Warm and humid",
		"Foggy in the morning, clearing later",
	}
	summary = conditions[sum%uint64(len(conditions))]
	// Warmer near the equator, colder toward the poles, plus a deterministic wobble.
	base := 30.0 - math.Abs(lat)*0.5
	temperatureC = math.Round((base+float64(sum%1000)/100.0-5.0)*10) / 10
	precipitationChance = int(sum % 101)
	return summary, temperatureC, precipitationChance, nil
}

/*
AskAgent answers a natural-language weather question. It runs the LLM tool-calling loop as a durable workflow, letting the model geocode a location with LatLng and fetch its forecast with Forecast before composing a reply.
*/
func (svc *Service) AskAgent(ctx context.Context) (graph *workflow.Graph, err error) { // MARKER: AskAgent
	graph = workflow.NewGraph("AskAgent")
	graph.SetURL("Answer", weatherapi.Answer.URL())
	graph.AddTransition("Answer", workflow.End)
	return graph, nil
}

// weatherAgentPrompt builds the conversation and tool list shared by the Answer task and the Ask function.
func weatherAgentPrompt(question string) (items []llmapi.Item, tools []string) {
	items = []llmapi.Item{
		llmapi.NewMessage("system", "You are a helpful weather assistant. Use the lat-lng tool to geocode a location, then the forecast tool to get its conditions, and answer the question conversationally.").AsItem(),
		llmapi.NewMessage("user", question).AsItem(),
	}
	tools = []string{
		weatherapi.LatLng.URL(),
		weatherapi.Forecast.URL(),
	}
	return items, tools
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

/*
Answer runs the LLM tool-calling loop for a weather question, chaining the LatLng and Forecast tools, and returns the agent's reply.
*/
func (svc *Service) Answer(ctx context.Context, flow *workflow.Flow, question string) (answer string, err error) { // MARKER: Answer
	items, tools := weatherAgentPrompt(question)
	// ChatLoop is llm.core's multi-step chat workflow, run here as a durable subgraph
	result, _, yield, err := llmapi.NewSubgraph(flow).ChatLoop(ctx, llmapi.ProviderAny, llmapi.ModelDefault, items, tools, nil)
	if err != nil {
		return "", errors.Trace(err)
	}
	if yield {
		return "", nil
	}
	return finalAnswer(result)
}

/*
Ask runs the weather agent synchronously for a natural-language question and returns its answer. It executes
the same tool-calling loop as the AskAgent workflow, but in-process via a single llm.core Chat call rather
than as a durable workflow, giving the tour one browser-clickable endpoint.
*/
func (svc *Service) Ask(ctx context.Context, q string) (answer string, err error) { // MARKER: Ask
	items, tools := weatherAgentPrompt(q)
	result, _, _, err := llmapi.NewClient(svc).Chat(ctx, llmapi.ProviderAny, llmapi.ModelDefault, items, tools, nil)
	if err != nil {
		return "", errors.Trace(err)
	}
	return finalAnswer(result)
}
