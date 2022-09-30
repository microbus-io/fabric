# HTTP Ingress Proxy

Think of NATS as a closed garden that requires a special key to access. In order to send and receive messages over NATS, it's necessary to use the NATS libraries to connect to NATS. This is basically what the `Connector` is facilitating for service-to-service calls.

Practically all systems will require interaction from a source that is outside the NATS bus. The most common scenario is perhaps a request generated from a web browser to a public API endpoint. In this case, something needs to bridge the gap between the incoming real HTTP request and the HTTP messages that travel over the bus. This is exactly the role of the HTTP ingress proxy.


<img src="httpingress-1.svg" width="420">

On one end, the HTTP ingress proxy listens on port `:8080` for real HTTP requests; on the other end it is connected to NATS. The ingress proxy converts real requests into requests on the bus; and on the flip side, converts responses from the bus to real responses. Because the bus messages in `Microbus` are formatted themselves as HTTP messages, this conversion is trivial, with two caveats:
* The proxy filters out `Microbus-*` control headers from coming in or leaking out
* The first segment of the path of the real HTTP request is treated as the host name of the microservice on the bus. So for example, `http://localhost:8080/echo.example/echo` is translated to the bus address `https://echo.example/echo` which is then mapped to the NATS subject `443.example.echo.|.echo`. Port 443 is assumed by default when a port is not explicitly specified
