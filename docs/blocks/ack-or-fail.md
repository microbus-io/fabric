# Ack or Fail Fast

When the connector of the downstream microservice receives a request, it sends an ack back to the upstream microservice before forwarding the request to the downstream microservice. The ack is a signal to the connector of the upstream microservice that the request was received and is being processed. The latter then waits for the response to arrive within the full request timeout.

Conversely, if an ack is not received within a short ack timeout (250ms by default), the connector of the upstream microservice fails fast under the assumption that no downstream microservice is available to process the request.

The ack is an HTTP response that is identified by an `Ack` op code and status code `100`:

```http
HTTP/1.1 100 Continue
Connection: close
Microbus-Op-Code: Ack
Microbus-From-Host: downstream.host.name
Microbus-From-Id: m3mcmiftmd
Microbus-Msg-Id: P4zpC2Ea
Microbus-Queue: downstream.host.name
```

### Happy Path

In the happy path, the upstream microservice sends a request to the downstream microservice. The ack from the downstream microservice is received by the upstream microservice within the ack timeout (250ms) so it knows that its request was received by the downstream microservice and therefore waits for the response. The response is received within the request timeout (20s).

<img src="./ack-or-fail-1.drawio.svg">
<p></p>

### Ack Timeout

In this scenario, no microservice matches the destination of the request. The connector of the upstream microservice generates a timeout error after the ack timeout (250ms), significantly faster than the full request timeout (20s).

<img src="./ack-or-fail-2.drawio.svg">
<p></p>

### Request Timeout

In this scenario, the ack is received by the upstream microservice within the ack timeout (250ms) so it knows that its request was received by the downstream microservice and therefore waits for the response. The response however is received after the request timeout (20s) and is discarded because the upstream microservice's connector had already errored out. 

<img src="./ack-or-fail-4.drawio.svg">
<p></p>

### Response After Timeout

In this false negative scenario, the ack is delayed due to latency. The downstream microservice responds within the request timeout (20s) but the response is discarded because the upstream microservice's connector had already timed out.

<img src="./ack-or-fail-3.drawio.svg">
<p></p>
