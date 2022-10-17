# Package `sub`

The `sub` package defines the internal `Subscription` struct that facilitates the endpoint subscriptions of the microservice. It transforms the partial path specification in `Connector.Subscribe` to produce a fully-qualified URL.

| Path specification | Fully-qualified URL |
| - | - |
| (empty) | https://www.example.com |
| / | https://www.example.com/ |
| :1080 | https://www.example.com:1080 |
| :1080/ | https://www.example.com:1080/ |
| :1080/path | https://www.example.com:1080/path |
| /path/with/slash | https://www.example.com:443/path/with/slash |
| path/with/no/slash | https://www.example.com:443/path/with/no/slash |
| https://www.example.com/path | https://www.example.com:443/path |
| https://www.example.com:1080/path | https://www.example.com:1080/path |

This package also defines various `Option`s that can be applied to the `Subscription` using the options pattern. This pattern is used in Go for expressing optional arguments. 

For example:

```go
con.Subscribe("/path", handler, sub.NoQueue())
```
