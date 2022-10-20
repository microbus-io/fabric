# Package `application`

An `Application` is a collection of microservices that run in a single process and share the same lifecycle. The ability to run multiple microservices in a single executable is one of the big benefits of the `Microbus` approach. It is much easier to develop and debug complex systems when all microservices can be run in a single debuggable process with minimal memory requirements.
 
Here's a simple example of an `Application` hosting two microservices: a "Hello, World!" microservice and the HTTP ingress microservice.

```go
package main

import (
	"net/http"
	"os"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/services/httpingress"
)

func main() {
	hello := connector.New("helloworld")
	hello.Subscribe("/", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("Hello, World!"))
		return nil
	})
	app := application.New(
		httpingress.NewService(),
		hello,
	)
	err := app.Run()
	if err != nil {
		os.Exit(-1)
	}
}
```
