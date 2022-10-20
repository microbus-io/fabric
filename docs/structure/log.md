# Package `log`

Logging is performed through the `Connector`'s `LogDebug`, `LogInfo`, `LogWarn` or `LogError` methods.
The `log` package defines type-safe log fields that can be attached to log messages.

Example:

```go
con := connector.New("logger.example")
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
