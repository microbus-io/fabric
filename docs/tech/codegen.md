# Code Generation

Automatic code generation is `Microbus`'s most powerful tool. It facilitates rapid development (RAD) of microservices and significantly increases developer productivity. Although it's possible to create a microservice by working directly with the `Connector`, the abstraction added by the code generator makes things simpler by taking care of much of the repetitive boilerplate code.

## Bootstrapping

Code generation starts by introducing the `//go:generate` directive into any source file in the directory of the microservice. The recommendation is to add it to a `doc.go` file:

```go
//go:generate go run github.com/microbus-io/fabric/codegen

package myservice
```

The next step is to create a `service.yaml` file which will be used to specify the functionality of the microservice. If the directory contains only `doc.go` or if `service.yaml` is empty, running `go generate` inside the directory will automatically populate `service.yaml`.

## Service.yaml

In `service.yaml`, developers are able to define the various pieces of the microservice in a declarative fashion. Code generation picks up these definitions to generate boilerplate code, leaving it up to the developer to implement the business logic.

`service.yaml` include several sections:

### General

The `general` section of `service.yaml` defines the `host` name of the microservice and its human-friendly `description`. The host name is required. It will be how the microservice is addressed inside the `Microbus` system. A hierarchical naming scheme for host names such as `myservice.mydomain.myproduct` can help avoid conflicts.

```yaml
# General
#
# host - The host name of the microservice
# description - A human-friendly description of the microservice
general:
  host: email.communication.xyz
  description: The email service delivers emails to recipients.
```

### Configs

The `configs` section is used to define the [configuration](./configuration.md) properties of the microservices. Config properties get their values in runtime from the [configurator](../structure/services-configurator.md) system microservice. 

```yaml
# Config properties
#
# signature - Func() (val Type)
# description - Documentation
# default - A default value (defaults to empty)
# validation - A validation pattern
#   str [a-zA-Z0-9]+
#   bool
#   int [0,60]
#   float [0.0,1.0)
#   dur (0s,24h]
#   set Red|Green|Blue
#   url
#   email
#   json
# callback - "true" to handle the change event (defaults to "false")
# secret - "true" to indicate a secret (defaults to "false")
configs:
  - signature:
    description:
    default:
    validation:
    callback:
    secret:
```

The `signature` is required. It defines the name and type of the property. The name must start with an uppercase letter. Types are limited to `string`, `bool`, `int`, `float` or `Duration`.

`validation` is enforced before accepting a new value for the config property. A validation comprises of a type and an optional regexp (for strings) or range (for numeric types).

If `callback` is set to `true`, a callback function will be generated and called when the value changes in runtime.

```go
// OnChangedFoo is triggered when the value of the Foo config property changes.
func (svc *Service) OnChangedFoo(ctx context.Context) (err error) {
    return // todo
}
```

### Functions

`functions` define a web endpoint that is made to appear like a function. Input arguments are pulled from either the JSON body of the request or from the query arguments. Output arguments are written as JSON to the body of the response.

```yaml
# Functions
#
# signature - Func(name Type, name Type) (name Type, name Type, httpStatusCode int)
# description - Documentation
# path - The subscription path
#   (empty) - The function name in kebab-case
#   /path
#   /directory/
#   :123/path
#   :123/... - Ellipsis denotes the function name in kebab-case
#   https://example.com:123/path
#   ^ - Empty path
# queue - The subscription queue
#   default - Load balanced (default)
#   none - Pervasive
functions:
  - signature:
    description:
    path:
    queue:
```

The `signature` defines the function name (which much start with an uppercase letter) and the input and output arguments. The special output argument `httpStatusCode` can be used to set the HTTP status code of the response.

The code generated functional request handler will look similar to this:

```go
/*
FuncHandler is an example of a functional handler.
*/
func (svc *Service) FuncHandler(ctx context.Context, id string) (ok bool, httpStatusCode int, err error) {
    return
}
```

Along with the host name of the service, the `path` defines the URL to this endpoint. It defaults to the function name in `kebab-case`.

`queue` defines whether a request is routed to one of the replicas of the microservice (load-balanced) or to all (pervasive).

### Types

Any complex non-primitive types used in functions must be declared in the `types` section. Primitive types are `int`, `float`, `byte`, `bool`, `string`, `Time` and `Duration`. Maps (dictionaries) and arrays are allowed. Types that are owned by this microservice are defined locally to this microservice. Types that are owned by other microservices but are used by this microservices must be imported by pointing to the location of their package.

Complex types may contain other complex types, in which case those nested types also must be declared.

```yaml
# Types
#
# name - All non-primitive types used in functions must be accounted for
# description - Documentation
# define - Define a new type with the specified fields (name: type)
# import - A path to another microservice that defines the type
types:
  - name:
    description:
    define:
      fieldName: Type
  - name:
    description:
    import:
```

### Web Handlers

The `webs` sections defines raw web handlers.

```yaml
# Web handlers
#
# signature - Func()
# description - Documentation
# path - The subscription path
#   (empty) - The function name in kebab-case
#   /path
#   /directory/
#   :123/path
#   :123/... - Ellipsis denotes the function name in kebab-case
#   https://example.com:123/path
#   ^ - Empty path
# queue - The subscription queue
#   default - Load balanced (default)
#   none - Pervasive
webs:
  - signature:
    description:
    path:
    queue:
```

The `signature` may not include any arguments. The handler receives the typical `http.ResponseWriter`, and `*http.Request` and is expected to extract input from there directly.

The code generated web handler will look similar to this:

```go
/*
WebHandler is an example of a web handler.
*/
func (svc *Service) WebHandler(w http.ResponseWriter, r *http.Request) (err error) {
    return nil // todo
}
```

### Tickers

Tickers are means to invoke a function on a periodic basis. The `signature` and `interval` fields are required.

```yaml
# Tickers
#
# signature - Func()
# description - Documentation
# interval - Duration between iterations (e.g. 15m)
# timeBudget - Duration to complete an iteration
tickers:
  - signature:
    description:
    interval:
    timeBudget:
```

The code generated ticker handler will look similar to this:

```go
/*
TickerHandler is an example of a ticker handler.
*/
func (svc *Service) TickerHandler(ctx context.Context) (err error) {
    return nil // todo
}
```

### Clients

In addition to the server-side, the code generator also creates clients to facilitate calling the microservice. A unicast `Client` and a multicast `MulticastClient` are placed in a separate API package to reduce the chance of cyclical dependencies between upstream and downstream microservices.

### Embedded Resources

A `resources` directory is automatically created with a `//go:embed` directive to allow microservices to bundle resource files along with the executable.

### Versioning

The code generator tool calculates a hash of the source code of the microservice which helps it detect changes. When a change is detected, the tool increments the version number of the microservice, storing it in `version-gen.go`. The version number is used to differentiate between different builds of the microservice.

## Structure of Generated Code

The code generator creates quite a few files and sub-directories in the directory of the microservice. Files that include `-gen` in their name are fully code generated and should not be edited.

The `{service}api` directory (and package) defines the `Client` and `MulticastClient` of the microservice and the complex types that they use. Together these represent the public-facing API of the microservice to upstream microservices. The name of the directory is derived from that of the microservice in order to make it easily distinguishable in code completion tools.

The `intermediate` directory (and package) defines the `Intermediate` which is used as the base of the microservice via anonymous inclusion. The `Intermediate` in turn extends the [`Connector`](../structure/connector.md).

The `resources` directory is a place to put static resources to be embedded into the executable of the microservice. Templates, images, scripts, etc. are some examples of what can potentially be embedded.

`service-gen.go` primarily includes the function to create a `NewService`. It may also alias the config initializers of the `Intermediate` and its `With` function, if appropriate.

`service.go` is where developers are expected to introduce the business logic of the microservice. `service.go` implements `Service`, which extends `Intermediate` as mentioned earlier. Most of the tools that a microservice needs are available through the receiver `(svc *Service)` which points to the `Intermediate` and by extension the `Connector`. It include the methods of the `Connector` as well as type-specific methods defined in the `Intermediate`.

```go
type Intermediate struct {
	*connector.Connector
}

type Service struct {
	*Intermediate
}

func (svc *Service) DoSomething(ctx context.Context) (err error) {
    // svc points to the Intermediate and the Connector
}
```

In addition to the standard `OnStartup` and `OnShutdown` callbacks, the code generator creates an empty function in `service.go` for each and every web handler, functional handler, ticker or config change callback defined in `service.yaml` as described earlier.

```go
// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return // todo
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return // todo
}
```

`version-gen.go` holds the SHA256 of the source code and the auto-incremented version number. `version-gen_test.go` makes sure it is up to date. If the test fails, running `go generate` will bring the version up to date.
