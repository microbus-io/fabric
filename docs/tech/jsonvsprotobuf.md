# JSON vs Protobuf

`Microbus` uses JSON over HTTP as the default protocol with which input arguments are sent to functional endpoints, and conversely, output arguments are returned from functional endpoints. Albeit not as efficient as [Protobuf](https://protobuf.io), JSON was chosen for several reasons:

* JSON is fully compatible with JavaScript clients, such as React applications running in the browser, making each endpoint easily exposed as a public API
* JSON is the basis for the two common web API styles: [RPC over JSON and REST](../tech/rpcvsrest.md)
* JSON is human-readable and more easily debuggable, contributing to engineering velocity
* Like Protobuf, JSON is extensible
* JSON is only about 2x slower than Protobuf, which in most cases is negligible compared to the network latency
