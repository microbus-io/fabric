# Package `application`

An `Application` is a collection of microservices that run in a single process and share the same lifecycle. The ability to run multiple microservices in a single executable is one of the big benefits of the `Microbus` approach. It is much easier to develop, debug and [test](../blocks/integration-testing.md) a system when all its microservices can be run in a single debuggable process with minimal memory requirements.
 
Here's a simple example of an `Application` hosting two microservices: a "Hello, World!" microservice and the HTTP ingress microservice.

```go
package main

import (
	"net/http"
	"os"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/coreservices/httpingress"
)

func main() {
	hello := connector.New("helloworld")
	hello.Subscribe("GET", "/", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("Hello, World!"))
		return nil
	})
	app := application.New(
		httpingress.NewService(),
		hello,
	)
	err := app.Run()
	if err != nil {
		os.Exit(1)
	}
}
```

Microservices are added to an `Application` either during creation in `application.New` or later via the `Include` method. In either case, the microservices are not automatically started. A call to `Startup` starts up all included microservices that are not already started. Conversely, a call to `Shutdown` shuts down all included microservices that are not already shut down.

The `Run` method starts up all microservices, waits for an interrupt and then shuts down all microservices. `Interrupt` allows to programmatically interrupt a running `Application`.

The methods `Services`, `ServicesByHost` (plural) and `ServiceByHost` (singular) allow searching for microservices included in the app.

Microservices can be `Join`ed to the `Application` without being included in it. The lifecycle of a joined microservice is not managed by the `Application` and it must be explicitly started up and shutdown. Joined microservices can fully communicate with other microservices included with or joined to the app.
