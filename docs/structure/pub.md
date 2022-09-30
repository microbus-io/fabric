# Package `pub`

The `pub` package is used to enable the functional options pattern in `Connector.Publish(ctx context.Context, options ...pub.Option)`. This pattern is used in Go for expressing optional arguments to a function. For this purpose the package defines the various `Option`s as well as their collector `Request`. `Request` is not used directly but rather applies and collects the list of `Option`s behind the scenes.

For example:

```go
import "github.com/microbus-io/fabric/connector"
import "github.com/microbus-io/fabric/pub"

type Service struct {
	*connector.Connector
}

func (s *Service) Foo(w http.ResponseWriter, r *http.Request) error {
	barReply, err := s.Publish(
		pub.GET("https://another.svc/bar"),
		pub.Body("foo"),
	)
	if err!=nil {
		return err
	}
	w.Write([]("bar"))
	return nil
}
```
