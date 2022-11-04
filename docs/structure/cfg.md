# Package `cfg`

The `cfg` package is used to enable the options pattern in `Connector.DefineConfig`. This pattern is used in Go for expressing optional arguments. This package defines the various `Option`s as well as their collector `Config` which is not used directly but rather applies and collects the list of `Option`s behind the scenes.

For example:

```go
con.DefineConfig("Database", cfg.Validation("url"), cfg.Secret())
```

The following options are supported:

* `cfg.DefaultValue` specifies a default value for the property when one is not provided by the configurator
* `cfg.Validation` uses a pattern to validate values before they are set
	* `str` - Plain text, no validation
	* `str ^[a-zA-Z0-9]+$` - Text with regexp validation
	* `bool` - Must be `true` or `false`
	* `int` - An integer, no validation
	* `int [0,60]` - An integer in range
	* `float` - A decimal number, no validation
	* `float [0.0,1.0)` - A decimal number in range
	* `dur` - A duration such as `7h3m45s500ms300us100ns`
	* `dur (0s,24h]` - A duration in range
	* `set Red|Green|Blue` - A set of explicit options separated by `|`
	* `url` - A URL
	* `email` - An email address, either `Joe <joe@example.com>` or just `joe@example.com`
	* `json` - A valid JSON string
* `cfg.Secret` indicates that the value of this property is a secret and should not be logged
* `cfg.Description` is intended to explain the purpose of the config property and how it will impact the microservice
