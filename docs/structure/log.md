# Package `log`

The `log` package provides the ability to create log fields. These fields are passed along in the logs.
The package currently makes use of the Zap logger (https://pkg.go.dev/go.uber.org/zap), although
abstracts this away so the scope of use is controlled. Abstracting this also allows the underlying technology
to be replaced whenever necessary.

The connector implements the loggers so that microservices can log at DEBUG, INFO, WARN, and ERROR levels,
and optionally pass along the log fields from the log package.

Example:

```go
con := NewConnector()
con.SetHostName("logger.example")
con.Subscribe("/foo", func(w http.ResponseWriter, r *http.Request) error {
	// Log level debug
	con.LogDebug(r.Context(), "Foo request", log.String("method", r.Method))

	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Log level warn
		con.LogWarn(r.Context(), "Reading body", log.Error(err), log.Bool("bar", true))
		return errors.Trace(err)
	}
	defer r.Body.Close()

	// Log level info
	con.LogInfo(r.Context(), "Successfully read body", log.ByteString("body", body))

	var jsonBody map[string]interface{}
	err = json.Unmarshal(body, &jsonBody)
	if err != nil {
		// Log level error
		con.LogError(r.Context(), "Unmarshalling body", log.Error(err), log.ByteString("body", body))
	}
	return nil
})
```
