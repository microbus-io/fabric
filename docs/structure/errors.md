# Package `errors`

The `errors` package is an enhancement of Go's standard `errors` package. It augments the standard `error`s to capture and print stack traces. For this purpose, it overrides the standard `errors.New` method and provides a new `errors.Newf` (in lieu of `fmt.Error`).

```go
import "github.com/microbus-io/errors"

err := errors.New("my error")
// err is augmented with the stack trace of this line
err = errors.Newf("error in process '%s'", processName)
// err is augmented with the stack trace of this line
```

Note how it seamlessly replaces the standard `import "errors"` with `import "github.com/microbus-io/errors"`. That is made possible because `github.com/microbus-io/errors` redefines all the constructs in `errors`.

If a standard `error` was created by an unaware function, `errors.Trace` is used to augment it with the stack trace

```go
import "github.com/microbus-io/errors"

body, err := io.ReadAll("non/existent.file") // err is a standard Go error
err = errors.Trace(err) // err is now augmented with the stack trace of this line
```

HTTP status codes can be attached to errors using `errors.Newc` or by converting the error to the underlying `TracedError` struct. The status code is returned to upstream clients.

```go
notFound := errors.Newc(http.StatusNotFound, "nothing to see here")

body, err := io.ReadAll("non/existent.file") // err is a standard Go error
errors.Convert(err).StatusCode = http.StatusNotFound
```

Both `errors.New` and `errors.Trace` support augmenting errors with optional annotations. Annotations can be added per stack location and do not alter the original error message.

Here is a complete example of an error bubbling from `c` to `b` to `a`:

```go
import "fmt"
import "github.com/microbus-io/errors"

func main() {
	a()
}

func a() {
	err := b()
	err = errors.Trace(err, "annotation by a") // Line 10
	fmt.Printf("%v", err)
	fmt.Print("\n-----\n")
	fmt.Printf("%+v", err)
}

func b() error {
	err := c()
	return errors.Trace(err, "annotation by b") // Line 18
}

func c() error {
	return errors.New("bad situation", "annotation by c") // Line 22
}
```

The `fmt` verb `%v` is used to print the error message while the `%+v` verb can be used to also print the stack trace. Calling `a()` will therefore output something like this:

```
bad situation
-----
bad situation

main.c
	/src/main/main.go:22
	annotation by c
main.b
	/src/main/main.go:18
	annotation by b
main.a
	/src/main/main.go:10
	annotation by a
```
