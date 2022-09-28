# Package `sub`

The `sub` package defines the internal `Subscription` struct that facilitates the endpoint subscriptions of the microservice. It transforms the partial path specification in `Connector.Subscribe(path string, handler sub.HTTPHandler, options ...sub.Option)` to produce a fully-qualified URL. In future releases, it will also enable the functional options pattern via various `Option`s.

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
