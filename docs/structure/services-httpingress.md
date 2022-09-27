# Package `services/httpingress`

An HTTP ingress proxy is needed in order to bridge the gap between the browser (or any HTTP client for that matter) and the microservices running on top of NATS because NATS is a closed network that requires a special type of connection. To achieve that, the ingress proxy is listening to HTTP requests on one end, and on the other it connects to the NATS network.

By default, the HTTP ingress proxy listens on port `:8080`. The port can be changed using the `Port` configuration property by either setting the environment variable `MICROBUS_HTTPINGRESSSYS_PORT` or by adding a section to the `env.yaml` file.

```
http.ingress.proxy:
  Port: 9090
```

Learn more in the [technical deep dive](../tech/httpingress.md).