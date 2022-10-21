# Package `cfg`

The `cfg` package is used to enable the options pattern in `Connector.DefineConfig`. This pattern is used in Go for expressing optional arguments. This package defines the various `Option`s as well as their collector `Config` which is not used directly but rather applies and collects the list of `Option`s behind the scenes.

For example:

```go
con.DefineConfig("Database", cfg.Validation("url"), cfg.Secret())
```
