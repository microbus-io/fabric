# Package `utils`

Package `utils` includes various independent utility classes and functions.

`BodyReader` implemented the `io.Reader` and `io.Closer` and is used to contain the body of a request or response. The access it provides to the underlying `[]byte` array is used for memory optimization purposes.

`ResponseRecorder` implements the `http.ResponseWriter` interface and is used as the underlying struct passed in to the request handlers in the `w *http.ResponseWriter` argument. The `ResponseRecorder` uses a `BodyReader` to contain the body of the generated response. Contrary to the `httptest.ResponseRecorder`, the `utils.ResponseRecorder` allows for multiple `Write` operations.

`CatchPanic` is a utility function that [converts panics into standard errors](../tech/errorcapture.md). It is used by the `Connector` to wrap callbacks to user code.
