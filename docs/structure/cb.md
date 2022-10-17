# Package `cb`

The `cb` package is used to enable the options pattern in `Connector.SetOnStartup`, `Connector.SetOnShutdown` and `Connector.StartTicker`. This pattern is used in Go for expressing optional arguments. This package defines the various `Option`s as well as their collector `Callback` which is not used directly but rather applies and collects the list of `Option`s behind the scenes.

For example:

```go
con.SetOnStartup(startupFunc, cb.TimeBudget(2*time.Minute))
```
