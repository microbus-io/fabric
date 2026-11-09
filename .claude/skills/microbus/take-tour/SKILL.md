---
name: take-tour
description: Runs the agent-guided tour of Microbus using examples. Use when the user asks to take the tour.
---

**CRITICAL**: Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

## Workflow

Copy this checklist and track your progress:

```
Take the agent-guided tour:
- [ ] Step 1: Download examples
- [ ] Step 2: Extend main app
- [ ] Step 3: Docker Compose
- [ ] Step 4: Run the example app
- [ ] Step 5: Example menu (OpenAPI portal + examples; loops until the user ends the tour)
- [ ] Step 6: Telemetry
- [ ] Step 7: Stop example app
- [ ] Step 8: Stop Docker containers
- [ ] Step 9: What's next
```

#### Step 1: Download Examples

Download the latest examples from Github.

```shell
git clone --depth 1 https://github.com/microbus-io/fabric temp-clone
rm -rf exampleservices
cp -r temp-clone/exampleservices .
rm -rf temp-clone  
```

The example files reference `github.com/microbus-io/fabric/exampleservices/...` internally. Replace all those references with the local module path.

#### Step 2: Extend Main App

Edit `main/main.go` and add the following block to the main `app` after the block of the core microservices.

```go
app.Add(
	// Example microservices
	helloworld.NewService(),
	hello.NewService(),
	messaging.NewService(),
	messaging.NewService(),
	messaging.NewService(),
	calculator.NewService(),
	eventsource.NewService(),
	eventsink.NewService(),
	yellowpages.NewService(),
	browser.NewService(),
	petstore.NewService(),
	login.NewService(),
	creditflow.NewService(),
	chatbox.NewService(),
	weather.NewService(),
	flightbooking.NewService(),
	// LLM core and providers - required by the chatbox, weather, and flight-booking demos
	llm.NewService(),
	claudellm.NewService(),
	chatgptllm.NewService(),
	geminillm.NewService(),
)
```

Add the appropriate imports too, again, replace all references to `github.com/microbus-io/fabric/exampleservices/...` with the local module path.

```go
import (
	"github.com/microbus-io/fabric/exampleservices/browser"
	"github.com/microbus-io/fabric/exampleservices/petstore"
	"github.com/microbus-io/fabric/exampleservices/calculator"
	"github.com/microbus-io/fabric/exampleservices/chatbox"
	"github.com/microbus-io/fabric/exampleservices/creditflow"
	"github.com/microbus-io/fabric/exampleservices/yellowpages"
	"github.com/microbus-io/fabric/exampleservices/eventsink"
	"github.com/microbus-io/fabric/exampleservices/eventsource"
	"github.com/microbus-io/fabric/exampleservices/flightbooking"
	"github.com/microbus-io/fabric/exampleservices/hello"
	"github.com/microbus-io/fabric/exampleservices/helloworld"
	"github.com/microbus-io/fabric/exampleservices/login"
	"github.com/microbus-io/fabric/exampleservices/messaging"
	"github.com/microbus-io/fabric/exampleservices/weather"
	"github.com/microbus-io/fabric/coreservices/httpingress/middleware"
	"github.com/microbus-io/fabric/coreservices/llm"
	"github.com/microbus-io/fabric/coreservices/claudellm"
	"github.com/microbus-io/fabric/coreservices/chatgptllm"
	"github.com/microbus-io/fabric/coreservices/geminillm"
)
```

Add the `LoginExample401Redirect` middleware to the HTTP ingress proxy using its `Init` method.

```go
httpingress.NewService().Init(func(svc *httpingress.Service) (err error) {
	svc.Middleware().Append("LoginExample401Redirect",
		middleware.OnRoute(
			func(path string) bool {
				return strings.HasPrefix(path, "/"+login.Hostname+"/")
			},
			middleware.ErrorPageRedirect(http.StatusUnauthorized, "/"+login.Hostname+"/login"),
		),
	)
	return nil
}),
```

#### Step 3: Docker Compose

Ask the user if they'd like to install NATS and the LGTM stack using Docker Compose.

- [NATS](https://nats.io) is not required for the tour, but it is worthwhile learning
- The [Grafana LGTM](https://grafana.com/blog/2024/03/13/an-opentelemetry-backend-in-a-docker-image-introducing-grafana/otel-lgtm/) stack is also optional and only required for one step of the tour

If the answer is yes, do so using the following command.

```shell
docker compose -f setup/microbus.yaml -p microbus up -d
```

**CRITICAL**: Stop the Docker containers when the user ends the tour or exits the session.

```shell
docker compose -f setup/microbus.yaml -p microbus down
```

#### Step 4: Run the Example App

Change into the `main` directory and run the example app in the backgroun.

```shell
cd main
go run main.go
```

**CRITICAL**: Interrupt or kill the example app when the user ends the tour or exists the session.

#### Step 5: Example Menu

This is the hub of the tour. There are many things to explore and the user should drive, not click "next" a dozen times. Present the numbered menu below (keep the one-line descriptions) and let the user choose by number. If the user is unsure where to start, suggest 1 (the OpenAPI Portal) for a bird's-eye view of the whole running app before drilling into individual microservices.

When the user picks an item, go to its subsection below, present it, and offer the deeper "Explore more" dive if they want it. Then return here and present the menu again. Repeat until the user chooses "End the tour" (option 0), then proceed to Step 6.

```
1.  OpenAPI Portal - The whole running app from the outside: every service's endpoints in one interactive surface.
2.  Calculator     - Typed functional endpoints (RPCs), input validation, and custom types.
3.  Hello          - Config properties, tickers, embedded resources, and inter-service calls.
4.  Messaging      - Unicast, multicast, and direct addressing, plus the distributed cache.
5.  Browser        - Outbound HTTP requests through the egress proxy.
6.  Petstore       - A third-party REST API delegated one-for-one from its OpenAPI document.
7.  Yellow Pages   - A SQL-backed microservice with JSON and REST endpoints and a web UI.
8.  Login          - JWT authentication and role-based access control.
9.  Credit Flow    - Agentic workflows: fan-out/in, forEach, goto loops, subgraphs, reducers.
10. Chatbox        - LLM tool-calling, with a simulated provider and optional real providers.
11. Embedder       - A Go microservice using a real Python library for its compute.
12. Weather        - An LLM agent built as a durable workflow, chaining two tools to answer a question.
13. Flight Booking - An agentic workflow with a real human-in-the-loop pause (Interrupt/Resume) and a subgraph.
0.  End the tour.
```

##### 1. OpenAPI Portal

The `openapi.core` portal aggregates the OpenAPI documents of every running service into a single interactive surface, so the user can take in the whole running app from the outside before drilling into any one microservice. A good first pick.

Explain the portal to the user and present the following links for them to experiment with in their browser:

- http://localhost:8080/openapi - Swagger-UI-style HTML browser; lists every running service and lets the user expand any one to see its endpoints, parameter tables, and sample payloads
- http://localhost:8080/openapi?hostname=calculator.example - the same browser scoped to a single service
- http://localhost:8080/openapi.json - the JSON aggregate document covering every running service, suitable for piping to other tools
- http://localhost:8080/openapi.json?hostname=calculator.example - the JSON document for a single service

Mention that the portal is generated automatically from each service's live subscriptions - there's no hand-maintained spec file to drift. Tasks, inbound events, and outbound events are filtered out at the source so the portal shows only what's externally callable (functions, web handlers, workflow graphs).

**Explore more** (offer, optional): walk the user through the structure of one service's entry (tags, paths, parameters, request/response schemas) and how the JSON output would feed an LLM tool-calling client.

Return to the Example Menu.

##### 2. Calculator

The `calculator.example` microservice demonstrates basic functional endpoints (RPCs) that perform mathematical operations. It shows how to define typed request/response functions, handle input validation errors, and work with custom types like points.

Explain the microservice to the user and present the following links for them to experiment with in their browser:

- http://localhost:8080/calculator.example/arithmetic?x=5&op=*&y=8 - computes 5 * 8
- http://localhost:8080/calculator.example/square?x=5 - computes 5 squared
- http://localhost:8080/calculator.example/square?x=not-a-number - demonstrates input validation error handling
- http://localhost:8080/calculator.example/distance?p1.x=0&p1.y=0&p2.x=3&p2.y=4 - calculates distance between two points using a custom Point type

**Explore more** (offer, optional): prepare and show an overview of the microservice's features; the user may ask to see the full implementation code of any feature (show the full code, not just signatures).

Return to the Example Menu.

##### 3. Hello

The `hello.example` microservice demonstrates a variety of framework capabilities including configuration properties, tickers, embedded static resources, and inter-service communication. It calls into `calculator.example` to show how microservices interact with each other, and multicasts a ping to discover all running microservices.

Explain the microservice to the user and present the following links for them to experiment with in their browser:

- http://localhost:8080/hello.example/echo - echoes back the raw HTTP request in wire format
- http://localhost:8080/hello.example/ping - multicasts a ping to discover all running microservices
- http://localhost:8080/hello.example/hello?name=Bella - greets the user using a configurable greeting
- http://localhost:8080/hello.example/calculator - renders a calculator UI that calls the calculator microservice
- http://localhost:8080/hello.example/bus.png - serves an embedded static image resource

**Explore more** (offer, optional): prepare and show an overview of the microservice's features; the user may ask to see the full implementation code of any feature (show the full code, not just signatures).

Return to the Example Menu.

##### 4. Messaging

The `messaging.example` microservice demonstrates service-to-service communication patterns - unicast (load-balanced), multicast (all peers respond), and direct addressing - as well as the distributed cache for sharing state across instances.

Explain the microservice to the user and present the following links for them to experiment with in their browser:

- http://localhost:8080/messaging.example/home - demonstrates unicast, multicast, and direct addressing patterns
- http://localhost:8080/messaging.example/cache-store?key=foo&value=bar - stores a key-value pair in the distributed cache
- http://localhost:8080/messaging.example/cache-load?key=foo - retrieves a value from the distributed cache by key

**Explore more** (offer, optional): prepare and show an overview of the microservice's features; the user may ask to see the full implementation code of any feature (show the full code, not just signatures).

Return to the Example Menu.

##### 5. Browser

The `browser.example` microservice demonstrates how to make outbound HTTP requests using the HTTP egress proxy. It provides a simple web UI with an address bar that fetches and displays the HTML source of any URL.

Explain the microservice to the user and present the following link for them to experiment with in their browser:

- http://localhost:8080/browser.example/browse?url=example.com - fetches and displays the HTML source of example.com

**Explore more** (offer, optional): prepare and show an overview of the microservice's features; the user may ask to see the full implementation code of any feature (show the full code, not just signatures).

Return to the Example Menu.

##### 6. Petstore

The `petstore.example` microservice demonstrates the recommended pattern for integrating a third-party REST API: a one-for-one *delegating* microservice generated from the API's OpenAPI document by the `import-openapi-microservice` skill. Where Browser showed raw egress, Petstore shows the typed pattern - each upstream operation becomes a Microbus endpoint that forwards through the HTTP egress proxy to the real public [Swagger Petstore](https://petstore3.swagger.io) API. Only a curated three-endpoint subset of the upstream API was imported (`AddPet`, `GetPetById`, `UploadFile`).

This step requires outbound internet access. The read endpoint is public, so no credential or configuration is needed.

Explain the microservice to the user and present the following link for them to experiment with in their browser:

- http://localhost:8080/petstore.example/pet/1 - fetches pet #1 from the live Swagger Petstore through the egress proxy (try other small integer IDs if #1 is absent)

`AddPet` (POST) and `UploadFile` (POST, binary upload) are not browser-clickable; mention they exist and that a write API would normally be gated with `requiredClaims` rather than left open like these public reads.

**Explore more** (offer, optional): point out that `service.go` is almost entirely the shared `makeFunctionRequest`/`makeWebRequest` helpers plus one-line handlers, and that `openapispecs.json` is the durable record of the upstream API. Offer to show the full implementation code of any feature.

Return to the Example Menu.

##### 7. Yellow Pages

The `yellowpages.example` microservice demonstrates a SQL-backed microservice built on the sequel package. It persists `Person` records with fields for first name, last name, email, and birthday, and exposes the same create, load, store, delete and list operations twice: as JSON endpoints for other microservices and as REST endpoints on `/persons`, with a web UI for trying the REST API. If a SQL database is not configured, it falls back to a local SQLite file.

Explain the microservice to the user and present the following links for them to experiment with in their browser:

- http://localhost:8080/yellowpages.example/demo?method=POST&path=/persons&body=%7B%22firstName%22:%22Harry%22,%22lastName%22:%22Potter%22,%22email%22:%22hp@hogwarts.edu%22,%22birthday%22:%221980-07-31T00:00:00Z%22%7D - opens the web UI pre-filled to create a person record (push submit)
- http://localhost:8080/yellowpages.example/demo?method=GET&path=/persons - opens the web UI pre-filled to list all persons (push submit)
- http://localhost:8080/yellowpages.example/demo?method=DELETE&path=/persons/{key} - opens the web UI pre-filled to delete a person by key (replace {key} with the actual key, then push submit)

**Explore more** (offer, optional): prepare and show an overview of the microservice's features; the user may ask to see the full implementation code of any feature (show the full code, not just signatures).

Return to the Example Menu.

##### 8. Login

The `login.example` microservice demonstrates authentication and role-based access control using JWT tokens. It includes a login form, protected pages with different role requirements, and cookie-based session management. Try logging in with `admin@example.com`, `manager@example.com`, or `user@example.com` (any password works).

Explain the microservice to the user and present the following link for them to experiment with in their browser:

- http://localhost:8080/login.example/welcome

**Explore more** (offer, optional): prepare and show an overview of the microservice's features; the user may ask to see the full implementation code of any feature (show the full code, not just signatures).

Return to the Example Menu.

##### 9. Credit Flow

The `creditflow.example` microservice demonstrates agentic workflows - multi-step processes orchestrated by the Foreman core service. It implements a credit approval workflow with 11 tasks spanning credit verification, employment verification, and identity verification. The workflow showcases advanced patterns: fan-out/fan-in parallelism, forEach iteration over employers, conditional transitions, goto loops for borderline credit scores, subgraphs for identity verification, reducers for merging parallel results, time budgets, retries, and sleep signals.

Explain the microservice to the user and present the following links for them to experiment with in their browser:

- http://localhost:8080/creditflow.example/demo - the demo page with a form to submit applicant data and run the workflow
- http://localhost:8080/creditflow.example/demo?name=Alice&ssn=123-45-6789&address=123+Main+St&phone=555-123-4567&employers=Acme+Corp&score=750 - pre-filled happy path (approved)
- http://localhost:8080/creditflow.example/demo?name=Bob&ssn=987-65-4321&address=456+Oak+Ave&phone=555-987-6543&employers=Globex&score=400 - pre-filled bad credit (rejected)
- http://localhost:8080/creditflow.example/demo?name=Diana&ssn=444-55-6666&address=321+Pine+Ln&phone=555-444-3333&employers=Umbrella+Corp&score=560 - pre-filled borderline score (demonstrates goto loop)
- http://localhost:8080/creditflow.example/demo?name=Eve&ssn=555-66-7777&address=654+Maple+Ct&phone=555-555-5555&employers=Acme+Corp,Globex,Initech&score=750 - pre-filled multiple employers (demonstrates forEach fan-out)

**Explore more** (offer, optional): prepare and show an overview of the microservice's features; the user may ask to see the full implementation code of any feature (show the full code, not just signatures).

Return to the Example Menu.

##### 10. Chatbox

The `chatbox.example` microservice demonstrates the LLM integration capabilities of Microbus. It implements a *simulated* LLM provider that pattern-matches math questions and uses the Calculator microservice as a tool - all without requiring a real LLM API key. The demo showcases the full tool-calling flow: the LLM service resolves tool schemas from OpenAPI, the simulated provider requests a tool call, the LLM service executes it against the Calculator, and the result is fed back to produce the final answer.

The demo page's provider dropdown has two options: **Simulated** (the no-key chatbox) and **Real**. The Real option passes `provider="any"` with the `fast` tier alias, and `llm.core` auto-selects whichever real provider (Claude, ChatGPT, or Gemini) has its `APIKey` configured - so a **single key of any brand** makes it work, with no code change. This is the provider-portability path: the example asks for a capability tier, not a specific vendor model.

**Offer to set up a real provider (optional).** Ask the user whether they want to try a real LLM, and if so, to paste one API key from any of the three providers. Detect which provider it belongs to from the key prefix and write it to that provider's block in the git-ignored `config.local.yaml`:

- `sk-ant-...` -> `claude.llm.core`
- `AIza...` (or `AI...`) -> `gemini.llm.core`
- `sk-...` (not `sk-ant-`) -> `chatgpt.llm.core`

```yaml
# example: a Claude key was provided
claude.llm.core:
  APIKey: sk-ant-...
```

If the prefix is ambiguous, ask the user which provider the key is for. Only one provider needs a key for the "Any configured provider" options to work.

Skipping this step is fine - the simulated option remains fully functional without any API keys. Mention that real-provider calls are billable.

**Restart the example app after adding the key.** Microbus reads `config.local.yaml` only at startup (the configurator loads it in `OnStartup`; it does not watch the file), so a key added while the app is running does not take effect until restart. Restart the app before choosing the Real option, or the request resolves to no configured provider and errors.

Explain the microservice to the user and present the following link for them to experiment with in their browser:

- http://localhost:8080/chatbox.example/demo - the interactive chat demo page

Suggest the user try questions like "What is 6 times 7?", "How much is 100 divided by 4?", or "Calculate 15 plus 28". Also try "Hello!" to see how it handles unrecognized questions. The dropdown lets them swap between the simulated chatbox and the Real option (which needs a key set above and an app restart).

**Explore more** (offer, optional): prepare and show an overview of the microservice's features; the user may ask to see the full implementation code of any feature (show the full code, not just signatures).

Return to the Example Menu.

##### 11. Embedder

The `embedder.example` microservice demonstrates how a Go microservice can use a real Python library (here, `sentence-transformers`) for its core compute. It loads the `all-MiniLM-L6-v2` model in an in-process Python virtual environment via the [`github.com/microbus-io/pyvenv`](https://github.com/microbus-io/pyvenv) module and exposes typed Go endpoints that delegate to it via `svc.venv.CallAndAwait`.

The microservice is already running as part of the example app. The Python venv is *not* started at OnStartup; the user explicitly triggers it from the demo page so the multi-second cost of `pip install sentence-transformers` and the model download is obvious rather than hidden. Skip this step if the user does not have `python3` on `$PATH` or does not want to download ~80MB on first run.

Present the demo link:

- http://localhost:8080/embedder.example/demo - interactive UI; click "Initialize Python VM" to start the venv (20–60 seconds on first run, instant on subsequent runs since the venv is cached on disk at a temp directory pyvenv creates). Once the status flips to "ready", the page exposes Embed and Similarity action cards.

Once the venv is ready, these direct REST calls also work:

- http://localhost:8080/embedder.example/similarity?a=cat&b=feline - cosine similarity between two strings
- http://localhost:8080/embedder.example/embed?text=hello - returns the 384-dimensional embedding

Explain the lifecycle to the user:

- The microservice's `OnStartup` constructs the `*pyvenv.Venv` but does not start it. The bus accepts the microservice for control-plane traffic immediately; the Python subprocess is not spawned until the user clicks Initialize.
- `Embed` and `Similarity` are subscribed with `sub.Manual()` and `sub.Tag("python")` so they stay off-bus until pyvenv's `LivenessCallback` fires `StateReady`. The microservice's `onVenvLiveness` handler activates the python-tagged subs at that point. Before that, calls to `/embed` or `/similarity` get a clean 404 ack-timeout.
- `Demo`, `DemoInit`, and `DemoStatus` are *not* manual and are reachable immediately. The page long-polls `DemoStatus` to surface tailed `pip install` and Python stdout/stderr while the venv warms up.
- If the Python subprocess dies unexpectedly after going Ready, the `LivenessCallback` fires `StateDied`; the microservice deactivates Python subs and schedules a fresh `Start` in the background. Recovery is fast since the on-disk venv is reused.
- `OnShutdown` calls `svc.venv.Close(ctx)` to kill the subprocess and clean up the on-disk venv.

**Explore more** (offer, optional): walk through `service.go`, `python.go`, and `service.py` (at the microservice's root, alongside the Go files) to show how the Go side delegates to Python via `svc.venv.CallAndAwait` and how the manual-subscription pattern is wired.

Return to the Example Menu.

##### 12. Weather

The `weather.example` microservice is the suite's canonical answer to "how do I build an agent?" Its headline form is a *workflow*: `AskAgent` is a single-node graph whose one task runs `llm.core`'s `ChatLoop` as a subgraph, exposing the microservice's own `LatLng` and `Forecast` endpoints as the model's tools. Ask it a question and the model geocodes the location with one tool, fetches conditions with the other, and composes a natural-language answer - the full tool-calling loop, which as a workflow the foreman drives so each step is durable and independently budgeted. The tour's clickable `/ask` runs that same loop synchronously (see below) so a browser gets an immediate reply.

Like the Chatbox "Real" option, this example needs a real LLM provider because the simulated provider does not do real tool-calling. If the user set up a provider key during the Chatbox step (10) and restarted the app, it already works here. If not, point them back to the Chatbox step's "set up a real provider" instructions (one key of any brand, written to `config.local.yaml`, then restart). Real-provider calls are billable.

Explain the microservice to the user and present the following links for them to experiment with in their browser:

- http://localhost:8080/weather.example/ask?q=What+should+I+wear+in+Paris+today%3F - runs the agent end-to-end and returns its reply
- http://localhost:8080/weather.example/ask?q=Is+it+raining+in+Tokyo+right+now%3F - another question, a different city
- http://localhost:8080/weather.example/lat-lng?location=London - the geocoding tool on its own (mock coordinates)
- http://localhost:8080/weather.example/forecast?lat=51.51&lng=-0.13 - the forecast tool on its own (mock conditions)

`ask` runs the same tool-calling loop synchronously via `llm.core`'s `Chat` (not the durable workflow), so the tour has one clickable URL that returns immediately. It does not touch the foreman - a workflow's own microservice never depends on the execution engine; launching `AskAgent` durably would be the job of whichever microservice owns the triggering event. The fact that this agent runs fine synchronously is itself the point: it is short enough not to strictly need a workflow, but the example models the workflow form for when an agent grows past a single request budget. `LatLng` and `Forecast` return deterministic mock data, so the example needs only an LLM key, not a third-party weather account.

**Explore more** (offer, optional): show the one-node graph in `service.go` (`AskAgent` wiring `Answer` to run `ChatLoop` as a subgraph) and the rendered `ASKAGENT.mmd` diagram, and contrast the workflow form with Chatbox's synchronous `Chat` - the same tool-calling loop, but durable, resumable, and observable step-by-step. The user may ask to see the full implementation code of any feature.

Return to the Example Menu.

##### 13. Flight Booking

The `flightbooking.example` microservice is the suite's showcase for the parts of the framework a synchronous LLM call cannot express: a durable graph, subgraph composition, and a **real human-in-the-loop pause** via `flow.Interrupt` / `foreman.Resume`. Where Credit Flow (9) is human-in-the-loop *by branching* and Weather (12) is a single-node agent, this workflow actually *parks* mid-run and waits for the traveler's decision. The `BookFlight` workflow searches a route, proposes one candidate flight at a time, and parks on an accept/keep-searching decision. Accepting a flight runs a separate `ChooseSeatAgent` child workflow as an isolated subgraph to pick a seat; keep-searching drives a goto loop back to the next candidate; an exhausted list ends with a not-booked message.

Because Interrupt/Resume is stateful and multi-round, the stop is a `/demo` web page rather than a clickable functional URL: a single stateless request cannot round-trip a parked flow. The page starts the flow, presents each proposed flight, and its **Accept** / **Keep searching** buttons resume the parked flow, carrying the flow key in a hidden field so nothing has to be copy-pasted.

Unlike Weather, this example is usable **without** a real LLM provider: only the final seat selection uses the LLM, and it degrades gracefully to the first available seat when no provider is configured, so the human-in-the-loop booking still completes end-to-end. If the user set up a provider key during the Chatbox step (10) and restarted the app, the seat is instead chosen by the model to match the natural-language preference. Real-provider calls are billable.

Explain the microservice to the user and present the following link for them to experiment with in their browser:

- http://localhost:8080/flightbooking.example/demo - the human-in-the-loop booking page (pre-filled San Francisco to London); push Search Flights, then Accept or Keep searching on each proposed flight

Suggest the user try Keep searching a couple of times to watch the goto loop propose the next candidate, then Accept to see the seat-selection subgraph and the final booking. The Execution History and Flow Diagram on the page show each task step, the `interrupted` park, and the nested subgraph.

**Explore more** (offer, optional): show the `BookFlight` graph in `service.go` (the `AwaitDecision` task's `flow.Interrupt` park and the `flow.Goto` keep-searching loop), the `ChooseSeatAgent` subgraph invoked via the typed `NewSubgraph` client, and the rendered `BOOKFLIGHT.mmd` diagram. Contrast the real Interrupt/Resume here with Credit Flow's branch-based approval. The user may ask to see the full implementation code of any feature.

Return to the Example Menu.

#### Step 6: Telemetry

Reached when the user ends the tour from the Example Menu. Skip this step if the user elected not to install the LGTM stack with Docker.

Explain to the user that they can view the telemetry collected by Grafana at http://localhost:3000. Metrics and traces are visualized in [dashboards](http://localhost:3000/dashboards) and can also be viewed in the [metrics drill-down app](http://localhost:3000/a/grafana-metricsdrilldown-app) and the [traces drill-down app](http://localhost:3000/a/grafana-exploretraces-app).

Proceed to the next step.

#### Step 7: Stop Example App

Interrupt or kill the example app that was spun up earlier.

#### Step 8: Stop Docker Containers

Skip this step if the user elected not to install NATS and LGTM with Docker in step 3.

Stop the Docker containers that were started earlier.

```shell
docker compose -f setup/microbus.yaml -p microbus down
```

#### Step 9: What's Next

Tell the user this concludes the tour and suggest to them that they try creating their own microservice by prompting the agent. Offer a few example prompts to get them started:

1. "Create a microservice that converts between Fahrenheit and Celsius"
2. "Create a microservice that shortens URLs and redirects short links back to the original"
3. "Create a microservice that returns a random quote from a list stored in a resource file"
4. "Create a microservice that accepts a Markdown body and returns rendered HTML"
