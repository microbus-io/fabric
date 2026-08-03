---
name: review-architecture
description: Performs an architectural review of a microservice-based system built on the Microbus framework. Covers only cross-cutting, cross-microservice concerns - service boundaries, the dependency graph, coupling, cross-service consistency, data ownership, workflow composition, edge security, and system operations. Anything judgeable inside a single microservice directory belongs to the review-microservice skill and is out of scope here. Produces a structured report with findings and recommendations.
---

**CRITICAL**: This review covers only cross-cutting, cross-microservice concerns - the relationships between
microservices and properties of the system as a whole. Anything that can be fully judged from within a single
microservice directory (framework compliance, per-endpoint security, code quality, test coverage, per-service
documentation, data-access performance, and a graph's own internal correctness) is out of scope here. Those checks
belong to the `review-microservice` skill. Do not duplicate its findings - when a per-service concern surfaces,
note it belongs to a per-service review and move on.

**CRITICAL**: Perform this review sequentially in the current context. Do NOT launch subagents to parallelize the
steps unless the user explicitly asks to run the review concurrently or asks for a faster review.

## Workflow

Copy this checklist and track your progress:

```
Architectural review:
- [ ] Step 1: Build the system map
- [ ] Step 2: Service boundaries and data ownership
- [ ] Step 3: Dependencies and coupling
- [ ] Step 4: Cross-service API consistency
- [ ] Step 5: Cross-service data consistency
- [ ] Step 6: Workflow composition
- [ ] Step 7: Edge and system security
- [ ] Step 8: System operations
- [ ] Step 9: Documentation and observability
- [ ] Step 10: Produce the report
```

#### Step 1: Build the System Map

Read `main/main.go` to identify all microservices included in the application and their startup group ordering. Read the `manifest.yaml` and `CLAUDE.md` of each microservice to understand its features, downstream dependencies, design rationale, and general properties. Read `config.yaml` for runtime configuration context. If `main/topology.mmd` exists, read it to visualize the dependency graph.

Produce a written inventory: for each microservice, note its hostname, purpose (from `description`), downstream dependencies, outbound events, inbound event sinks, configs, `db` and `cloud` properties, and startup group. This inventory is the foundation for all subsequent steps. It is also the map used to scope a change-driven review to the microservices that were touched and their immediate graph neighbors.

#### Step 2: Service Boundaries and Data Ownership

Evaluate how responsibility and data are divided across microservices. Every check here needs a view of two or more microservices at once - it cannot be made from inside one directory.

- **Separation of concerns** - does each microservice own a single, well-defined domain? Flag microservices whose description suggests multiple unrelated responsibilities.
- **Right-sized microservices** - are they too granular (nano-services adding unnecessary network hops) or too broad (a distributed monolith with many unrelated endpoints)?
- **Cohesion across the system** - is related functionality grouped together, or scattered across microservices? Are there endpoints that would be better placed in a *different* microservice than the one hosting them?
- **Encapsulated persistence** - does any microservice access another's database, share tables via foreign keys or JOINs, or make assumptions about another's schema? Grep for SQL statements and check import paths to verify.
- **Encapsulated external APIs** - is each external API (third-party web service, cloud provider, etc.) wrapped in its own dedicated microservice, as indicated by the `cloud` property in the manifest? Flag an upstream microservice that reaches an external API directly (grep for `http.NewRequest` or `http.Get`) instead of going through a dedicated wrapping microservice and the HTTP egress proxy. The per-microservice grep for direct HTTP is a `review-microservice` concern; here the question is whether the external dependency deserves its own encapsulating microservice.
- **Hostname conventions** - do hostnames follow the framework's dotted lowercase convention and stay unique across the app? Are core services on `.core` and application services on an appropriate domain?
- **Data ownership clarity** - does each piece of data have exactly one authoritative owner (one microservice that owns the writes)? Flag data whose write authority is split or ambiguous across microservices.

#### Step 3: Dependencies and Coupling

Analyze the dependency graph from the `downstream` sections of each `manifest.yaml`, and verify against actual code:

- **No circular dependencies** - is the dependency graph a DAG? Trace the graph from the inventory built in Step 1. Flag any cycles, including indirect cycles (A depends on B depends on C depends on A).
- **Loose coupling** - do microservices interact only through their `*api` client stubs? Grep import statements in each microservice for imports of other microservices' packages. Flag any import of a sibling microservice's internal package (anything other than its `*api` package).
- **Fan-out** - does any single request cascade through many microservices? Trace call chains from ingress-facing endpoints. The framework enforces a max call depth of 64, but even chains deeper than 4-5 levels warrant scrutiny for latency and fragility.
- **Dependency direction** - do higher-level, domain-specific microservices depend on lower-level, generic ones - not the reverse? Flag infrastructure or utility microservices that depend on domain microservices.
- **Event pairing** - does every inbound event sink reference a valid outbound event from the source microservice? Check that the `source` in each `inboundEvents` entry matches an actual `outboundEvents` entry in the source microservice's manifest.
- **Appropriate use of events** - are events used for loose coupling where appropriate (e.g., notifications, cache invalidation, cross-domain reactions), rather than forcing direct downstream dependencies? Conversely, are events misused where a direct call would be simpler and more reliable (e.g., request/response flows shoehorned into events)?
- **Manifest accuracy** - does the `downstream` section of each manifest match the actual `*api` imports in the code? Flag missing or stale entries in either direction.

#### Step 4: Cross-Service API Consistency

Examine the `functions`, `webs`, and `outboundEvents` sections across all manifests together. This step is only about consistency *between* microservices; per-endpoint correctness (route kebab-case, port choice, method usage, signature shape, magic HTTP args, pagination on a given endpoint) is a `review-microservice` concern and is not repeated here.

- **Consistent contracts** - are naming, HTTP methods, routes, and error-handling patterns uniform across microservices? Flag inconsistencies in conventions, e.g. one microservice uses `Load` while another uses `Get` for the same semantic operation, or one returns `404` for absence while another returns an empty result.
- **Consistent error semantics** - are HTTP status codes used the same way across microservices? Is `404` always "not found" system-wide? Are 4xx-vs-5xx conventions applied uniformly?
- **Chatty cross-service APIs** - are there 1+N call patterns where one microservice loops over a singular operation on another instead of using a bulk endpoint? Read the `service.go` of upstream microservices to see how they call downstream clients, and flag opportunities for bulk operations that would collapse a cross-service loop.

#### Step 5: Cross-Service Data Consistency

Evaluate consistency strategies for data that flows between microservices. Per-microservice cache mechanics (`DistribCache` init, invalidation, stampede protection) and a microservice's own migration-file safety are `review-microservice` concerns; here the focus is the cross-service data flow.

- **Consistency strategy** - is there an explicit choice between strong and eventual consistency for each cross-service data flow? Flag flows where the consistency model is ambiguous or implicit.
- **Cross-service data access** - when a microservice needs data owned by another, is the strategy appropriate? The three strategies are: on-demand querying via client stubs (strong consistency, higher latency), caching via `DistribCache` with event-driven invalidation (eventual consistency, volatile storage), or denormalization into local tables (eventual consistency, durable storage). Flag mismatches between the chosen strategy and the consistency requirement.
- **No distributed transactions** - are there implicit distributed transactions where multiple microservices are mutated in sequence without compensation or saga logic? Flag operations that partially succeed across microservices with no rollback path.
- **Event ordering** - do event sinks assume events arrive in a particular order? In a distributed system, events can arrive out of order or be delivered more than once. Flag handlers that depend on sequencing (e.g., "created" before "updated") without explicit ordering logic or idempotency guards.

#### Step 6: Workflow Composition

Skip this step if no microservice has `tasks` or `workflows` in its `manifest.yaml`. A single graph's internal correctness (fan-in reducers, its own subgraph state hygiene, LLM tool-exposure shape, task idempotency, long-running-task handling, and per-microservice foreman call-site anti-patterns) is a `review-microservice` concern. This step covers only what needs a system-wide view.

- **`main.go` wiring** - if any workflow calls an LLM, verify `foreman.core`, `llm.core`, and an LLM provider (e.g. `claudellm.core`) are all added to `main/main.go`. Even a non-LLM workflow needs `foreman.core`. The workflow compiles without them but fails at runtime with an `ack timeout` on whichever endpoint is missing.
- **Launcher-vs-provider separation** - a `foremanapi.Run`/`Create` launch site is cleanest in a microservice separate from the workflow's *provider* (the microservice hosting the tasks). Evaluate where launches live across the system: a microservice that both provides a workflow and exposes a pass-through `Run<Workflow>`/`<Workflow>Status` endpoint on it is coupling the provider to the foreman - the code that triggers a workflow should call the foreman directly. The same microservice launching its own workflow is acceptable when it genuinely owns the *triggering event* (its own ticker, an inbound event sink, or a UI/webhook action). Flag concentration of launch logic in the wrong tier.
- **Cross-service subgraph boundaries** - when a workflow in one microservice invokes another microservice's workflow via `flow.Subgraph(otherflowapi.OtherFlow.URL(), input)`, check that state field names line up at the boundary, or that a small adapter task (`Before<Node>` / `After<Node>`, using `flow.Del` / `flow.Clear`) reshapes state across it. A cross-service subgraph where the parent stores `userQuestion` but the callee reads `prompt`, with no adapter, is a boundary smell that only a two-microservice view reveals.

#### Step 7: Edge and System Security

Review the authentication and authorization posture of the system as a whole. Per-endpoint `requiredClaims` correctness, stale `iss` predicates, unmarked secrets, and input validation are `review-microservice` concerns; this step covers the edge and the trust-root tier.

- **Authentication at the edge** - is the HTTP ingress proxy configured with authorization middleware that validates bearer tokens and exchanges them for access tokens? Check the HTTP ingress setup in `main/main.go` and its configuration.
- **Trust-root tier isolation** - identify every endpoint on `:666` (token minters, shell exec, privileged writes) across the system. Is the set small and explicitly trusted? A `:666` endpoint's only authorization layer is the NATS `:666` `PUB` ACL, so flag any *new* or surprising `:666` capability, and never expect one to be gated by `requiredClaims`. Confirm internal ports `:666` and `:888` are not exposed through the ingress and that `AllowedInternalPorts` (if set) lists only intentionally reachable internal ports.
- **System-wide authorization coverage** - using the inventory, scan for externally reachable endpoints (`:443`, or any port in `AllowedInternalPorts`) that perform sensitive operations or wield a stored secret yet carry no `requiredClaims`. Flag confused-deputy risks where an ungated public endpoint spends an operator credential on any caller's behalf. Defer the per-endpoint claims-expression audit to `review-microservice`.

#### Step 8: System Operations

Evaluate deployment and composition properties that depend on the whole application:

- **Startup group ordering** - in `main/main.go`, are microservices added in the correct startup group order, each starting after its dependencies? Verify against the dependency graph from Step 3.
- **Application composition** - are all microservices referenced by the dependency graph actually added to `main/main.go`? Is the HTTP egress proxy present if any microservice makes web requests? Are `foreman.core`/`llm.core`/providers present when workflows need them (cross-check with Step 6)?
- **Version tracking** - when a microservice's `*api/definition.go` has changed, is its `Version` const incremented? Stale versions across the app hamper rollout coordination.

Per-microservice operational concerns - resource lifecycle symmetry, ticker intervals, embedded-resource placement, config defaults and validation, and test coverage - are `review-microservice` concerns and are not audited here.

#### Step 9: Documentation and Observability

Assess whether the *system* is understandable and observable as a whole. Per-microservice `CLAUDE.md`/godoc quality, dead metrics, non-framework logging, error tracing, and histogram buckets are `review-microservice` concerns.

- **System documentation** - from the inventory, flag microservices with missing, empty, or boilerplate-only `CLAUDE.md`, and note whether the manifests collectively give a coherent picture of the system. Deep per-file documentation audits belong to the per-service review.
- **Manifest description completeness** - do the `description` fields across manifests give a reader enough to model the system without reading code? Flag microservices whose exposed features are described in placeholder terms.
- **Cross-service trace continuity** - is `ctx` propagated across microservice boundaries so distributed traces stay connected? Flag any call site that starts a new `context.Background()` before a downstream call. Within-microservice context propagation is a per-service concern.

#### Step 10: Produce the Report

Compile findings into a structured report. For each category (Steps 2-9), list:

- **Findings** - specific observations, both positive and negative, with file paths and line numbers where applicable
- **Recommendations** - actionable suggestions for improvement, ordered by severity

Use severity levels:
- **Critical** - architectural issues that could cause system-wide failures, data loss, or security vulnerabilities
- **Warning** - design concerns that may lead to problems at scale or during evolution
- **Info** - minor improvements or deviations from best practices

Omit categories where no issues were found, but mention them in the summary as areas that passed review.

Format the report as follows:

```markdown
# Architectural Review

## Summary
Brief overview of the system, number of microservices reviewed, and high-level assessment.
Note categories that passed review without issues.

## Service Boundaries and Data Ownership
### Findings
- ...
### Recommendations
- [severity] ...

## Dependencies and Coupling
### Findings
- ...
### Recommendations
- [severity] ...

(repeat for each category that has findings)

## Conclusion
Overall assessment and prioritized action items.
```

Present the report to the user.
