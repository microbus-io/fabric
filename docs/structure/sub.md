# Package `sub`

The `sub` package defines the internal `Subscription` struct that facilitates the endpoint subscriptions of the microservice. It transforms the partial path specification in `Connector.Subscribe` to produce a fully-qualified URL.

| Path specification | Fully-qualified URL |
| - | - |
| (empty) | https://example.host |
| / | https://example.host/ |
| :1080 | https://example.host:1080 |
| :1080/ | https://example.host:1080/ |
| :1080/path | https://example.host:1080/path |
| /path/with/slash | https://example.host:443/path/with/slash |
| path/with/no/slash | https://example.host:443/path/with/no/slash |
| /path/{argument}/or/{suffix+} | https://example.host:443/path/{argument}/or/{suffix+} |
| https://another.host/path | https://another.host:443/path |
| https://another.host:1080/path | https://another.host:1080/path |

This package also defines various `Option`s that can be applied to the `Subscription` using the options pattern. This pattern is used in Go for expressing optional arguments. 

For example:

```go
con.Subscribe("GET", "/path", handler, sub.NoQueue())
```
