# Package `log`

The `log` package provides the ability to create log fields. These fields are passed along in the logs.
The package currently makes use of the Zap logger (https://pkg.go.dev/go.uber.org/zap), although
abstracts this away so the scope of use is controlled. Abstracting this also allows the underlying technology
to be replaced whenever necessary.

The connector implements the loggers so that microservices can log at DEBUG, INFO, WARN, and ERROR levels,
and optionally pass along these log fields.

Example:

```go
import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
)

type Service struct {
	*connector.Connector
}

func (s *Service) Foo(w http.ResponseWriter, r *http.Request) error {
	s.LogDebug(r.Context(), "Foo request", log.String("method", r.Method))

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.LogWarn(r.Context(), "Reading body", err, log.Bool("bar", true))
		return errors.Trace(err)
	}
	defer r.Body.Close()

	s.LogInfo(r.Context(), "Successfully read body", log.ByteString("body", body))

	var jsonBody map[string]interface{}
	err = json.Unmarshal(body, &jsonBody)
	if err != nil {
		s.LogError(r.Context(), "Unmarshalling body", err, log.ByteString("body", body))
	}

	return nil
}
```
