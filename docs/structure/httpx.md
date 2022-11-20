# Package `httpx`

Package `httpx` includes various HTTP utilities.

`BodyReader` implemented the `io.Reader` and `io.Closer` and is used to contain the body of a request or response. The access it provides to the underlying `[]byte` array is used for memory optimization purposes.

`ResponseRecorder` implements the `http.ResponseWriter` interface and is used as the underlying struct passed in to the request handlers in the `w *http.ResponseWriter` argument. The `ResponseRecorder` uses a `BodyReader` to contain the body of the generated response. Contrary to the `httptest.ResponseRecorder`, the `utils.ResponseRecorder` allows for multiple `Write` operations.

`ParseRequestData` parses the body and query arguments of an incoming request and populates a data object that represents its input arguments. This type of parsing is used in the generated code of the microservice to process functional requests.
