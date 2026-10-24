package weatherapi

import "github.com/microbus-io/fabric/define"

var _ = define.None

// HINT: This file is the single source of truth for the microservice's API. After editing it, run
// cmd/genservice on the microservice's directory (the parent of this api package) to regenerate client.go,
// intermediate.go, mock.go, mock_test.go, and manifest.yaml. Do not hand-edit those generated files.

// Hostname is the default hostname of the microservice.
const Hostname = "weather.example"

// Name is the decorative PascalCase name of the microservice.
const Name = "Weather"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 4

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `Weather is an LLM agent that answers natural-language weather questions by geocoding a location and fetching its forecast through sequential tool-calling.`

// LatLng geocodes a location name into its latitude and longitude coordinates.
var LatLng = define.Function{ // MARKER: LatLng
	Host: Hostname, Method: "ANY", Route: "/lat-lng",
	In: LatLngIn{}, Out: LatLngOut{},
}

// LatLngIn are the input arguments of LatLng.
type LatLngIn struct { // MARKER: LatLng
	Location string `json:"location,omitzero" jsonschema_description:"Location is a place name, e.g. San Francisco or London, UK"`
}

// LatLngOut are the output arguments of LatLng.
type LatLngOut struct { // MARKER: LatLng
	Lat float64 `json:"lat,omitzero" jsonschema_description:"Lat is the latitude in decimal degrees"`
	Lng float64 `json:"lng,omitzero" jsonschema_description:"Lng is the longitude in decimal degrees"`
}

// Forecast returns the current weather forecast for a latitude/longitude coordinate.
var Forecast = define.Function{ // MARKER: Forecast
	Host: Hostname, Method: "ANY", Route: "/forecast",
	In: ForecastIn{}, Out: ForecastOut{},
}

// ForecastIn are the input arguments of Forecast.
type ForecastIn struct { // MARKER: Forecast
	Lat float64 `json:"lat,omitzero" jsonschema_description:"Lat is the latitude in decimal degrees"`
	Lng float64 `json:"lng,omitzero" jsonschema_description:"Lng is the longitude in decimal degrees"`
}

// ForecastOut are the output arguments of Forecast.
type ForecastOut struct { // MARKER: Forecast
	Summary             string  `json:"summary,omitzero" jsonschema_description:"Summary is a short human-readable description of the conditions, e.g. Sunny with light winds"`
	TemperatureC        float64 `json:"temperatureC,omitzero" jsonschema_description:"TemperatureC is the temperature in degrees Celsius"`
	PrecipitationChance int     `json:"precipitationChance,omitzero" jsonschema_description:"PrecipitationChance is the chance of precipitation as a percentage from 0 to 100"`
}

// Answer runs the LLM tool-calling loop for a weather question, chaining the LatLng and Forecast tools, and returns the agent's reply.
var Answer = define.Task{ // MARKER: Answer
	Host: Hostname, Method: "POST", Route: ":428/answer",
	In: AnswerIn{}, Out: AnswerOut{},
}

// AnswerIn are the input arguments of Answer.
type AnswerIn struct { // MARKER: Answer
	Question string `json:"question,omitzero"`
}

// AnswerOut are the output arguments of Answer.
type AnswerOut struct { // MARKER: Answer
	Answer string `json:"answer,omitzero"`
}

// AskAgent answers a natural-language weather question. It runs the LLM tool-calling loop as a durable workflow, letting the model geocode a location with LatLng and fetch its forecast with Forecast before composing a reply.
var AskAgent = define.Workflow{ // MARKER: AskAgent
	Host: Hostname, Method: "GET", Route: ":428/ask-agent",
	In: AskAgentIn{}, Out: AskAgentOut{},
}

// AskAgentIn are the input arguments of AskAgent.
type AskAgentIn struct { // MARKER: AskAgent
	Question string `json:"question,omitzero" jsonschema_description:"Question is a natural-language weather question, e.g. What should I wear in Paris today?"`
}

// AskAgentOut are the output arguments of AskAgent.
type AskAgentOut struct { // MARKER: AskAgent
	Answer string `json:"answer,omitzero" jsonschema_description:"Answer is the agent's natural-language reply"`
}

/*
Ask runs the weather agent synchronously for a natural-language question and returns its answer. It executes
the same tool-calling loop as the AskAgent workflow, but in-process via a single llm.core Chat call rather
than as a durable workflow, giving the tour one browser-clickable endpoint.
*/
var Ask = define.Function{ // MARKER: Ask
	Host: Hostname, Method: "ANY", Route: "/ask",
	In: AskIn{}, Out: AskOut{},
}

// AskIn are the input arguments of Ask.
type AskIn struct { // MARKER: Ask
	Q string `json:"q,omitzero" jsonschema_description:"Q is a natural-language weather question, e.g. What should I wear in Paris today?"`
}

// AskOut are the output arguments of Ask.
type AskOut struct { // MARKER: Ask
	Answer string `json:"answer,omitzero" jsonschema_description:"Answer is the agent's natural-language reply"`
}
