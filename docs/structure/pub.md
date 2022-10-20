# Package `pub`

The `pub` package is used to enable the options pattern in `Connector.Publish`. This pattern is used in Go for expressing optional arguments. This package defines the various `Option`s as well as their collector `Request` which is not used directly but rather applies and collects the list of `Option`s behind the scenes.

For example:

```go
con.Publish(
	ctx,
	pub.GET("https://another.svc/bar"),
	pub.Body("foo"),
)
```
