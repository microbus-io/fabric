# Package `httpx`

Package `httpx` includes various HTTP utilities.

`BodyReader` implemented the `io.Reader` and `io.Closer` and is used to contain the body of a request or response. The access it provides to the underlying `[]byte` array is used for memory optimization purposes.

`ResponseRecorder` implements the `http.ResponseWriter` interface and is used as the underlying struct passed in to the request handlers in the `w *http.ResponseWriter` argument. The `ResponseRecorder` uses a `BodyReader` to contain the body of the generated response. Contrary to the `httptest.ResponseRecorder`, the `utils.ResponseRecorder` allows for multiple `Write` operations.

`ParseRequestData` parses the body and query arguments of an incoming request and populates a data object that represents its input arguments. This type of parsing is used in the generated code of the microservice to process functional requests.

The `frag` package implements means to break large HTTP requests and responses into fragments that can then be reassembled. Fragmentation is required because NATS sets a limit (1MB by default) to the size for messages that can be transferred on the bus.

`FragRequest` and its counterpart `DefragRequest` break and reassemble (respectively) large `http.Request`s. `FragResponse` and its counterpart `DefragResponse` break and reassemble (respectively) large `http.Response`s. Fragmentation is required because NATS sets a limit (1MB by default) to the size for messages that can travel on the bus. During fragmentation, fragments are added the control header `Microbus-Fragment` that specifies the index of the fragment out of the total number of fragments. This information is later used to reassemble the fragments on the destination.

Here's an example of an HTTP request that was fragmented into 3 fragments of up to 128 bytes each. The client starts by sending the first fragment. Notice the header `Microbus-Fragment: 1/3` indicating this is the first of three fragments.

```
POST /too-big HTTP/1.1
Host: server.example
User-Agent: Go-http-client/1.1
Content-Length: 128
Microbus-Call-Depth: 1
Microbus-Fragment: 1/3
Microbus-From-Host: client.example
Microbus-From-Id: m3mcmiftmd
Microbus-Msg-Id: P4zpC2Ea
Microbus-Op-Code: Req
Microbus-Time-Budget: 19999

9kHYrhgFdztxSZ00eafjfHoirvROe53j8ooZA14z0CxMV9cMHbjnKeVVHxarmvlyhGqbtiOTGsYfE7eLPImNQgYRKYG01npWZBfqlVbkqw2zxWznetDzD0q5fOr4HKOn
```

Upon receipt of the first fragment, the server responds with a `100 Continue` ack message.

```
HTTP/1.1 100 Continue
Connection: close
Microbus-Op-Code: Ack
Microbus-From-Host: server.example
Microbus-From-Id: m3mcmiftmd
Microbus-Msg-Id: P4zpC2Ea
Microbus-Queue: server.example
```

The client then sends the remaining fragments.

```
POST /too-big HTTP/1.1
Host: server.example
User-Agent: Go-http-client/1.1
Content-Length: 128
Microbus-Call-Depth: 1
Microbus-Fragment: 2/3
Microbus-From-Host: client.example
Microbus-From-Id: m3mcmiftmd
Microbus-Msg-Id: P4zpC2Ea
Microbus-Op-Code: Req
Microbus-Time-Budget: 19999

IBtVOBMQPjaBTEdwXTeCij9ZY61OOidkYTnwgUk98tC7mZzAgsDTH2pRxKTav0lD34MYJS0haYgWUr0brT1RENDCoffYIzKQYDcAsp73O7X1HD9VjGv0C3parRDPCCEz
```

and

```
POST /too-big HTTP/1.1
Host: server.example
User-Agent: Go-http-client/1.1
Content-Length: 16
Microbus-Call-Depth: 1
Microbus-Fragment: 3/3
Microbus-From-Host: client.example
Microbus-From-Id: m3mcmiftmd
Microbus-Msg-Id: P4zpC2Ea
Microbus-Op-Code: Req
Microbus-Time-Budget: 19999

7SLujUrm4W99YLUp
```
