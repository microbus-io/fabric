# Package `services/httpingress`

An HTTP ingress proxy is needed in order to bridge the gap between the browser (or any HTTP client for that matter) and the microservices running on top of NATS because NATS is a closed network that requires a special type of connection. To achieve that, the ingress proxy is listening to HTTP requests on one end, and on the other it connects to the NATS network.

Learn more in the [technical deep dive](../tech/httpingress.md).
