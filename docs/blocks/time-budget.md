# Time Budget and Call Stack Depth

For any network call it is best practice to set a timeout and error out if a response is not received in time. Timeouts prevents the client from infinitely waiting for a remote that may never respond, for example when a network failure prevents the client from reaching the remote, or when the remote is offline.

## Point-to-Point Timeouts

The common practice is to set a timeout on a point-to-point basis, between each client microservice and the remote microservice that it calls. Although easy to implement, this is an anti-pattern that results in improper system behavior.

In this first example, microservice `X` is setting a shorter timeout than microservice `Y`, resulting in microservice `Z` performing work that is no longer needed.

<img src="./time-budget-1.drawio.svg">
<p></p>

In the second example, all microservices set a 1 minute timeout. Nevertheless, microservices `W` and `X` time out while microservice `Z` is still working. There is no way to tell microservice `Z` to stop.

<img src="./time-budget-2.drawio.svg">
<p></p>

In both cases the issue stems from the fact that upstream services do not have visibility into what's happening downstream.

## Time Budget

A time budget introduces a deadline for the entire transaction DAG to complete. The deadline is set at the root and propagated downstream. Every microservice therefore knows when the transaction will time out and so all microservices can abort at the same time.

The typical way to implement a time budget is to pass along a timestamp by which the transaction must end. However, one must consider that microservices may run on different hardware whose clocks are not fully in-sync. To deal with potential clock skew, the deadline is passed as a duration that gets depleted, hence the terminology "budget".

In the following example, microservices `Y` will have completed its work at the 45 sec mark while microservices `Z`, `X` and `W` will have all timed out at the 60 sec mark.

<img src="./time-budget-3.drawio.svg">
<p></p>

If you look closely at the code you'll notice that the budget is decreased somewhat with each hop. This compensates for the network latency.

The time budget is set at the root of the transaction. For incoming HTTP requests, the root is the [HTTP ingress proxy](httpingress.md) which by default sets the time budget to 20 secs. This may seem quite short but the philosophy is that most users expect a response within 2 secs or less and they will hit the refresh button long before 20 secs pass. If something takes long to process, it should probably be handled asynchronously.

It is possible to [configure the time budget](../structure/coreservices-httpingress.md) set by the ingress proxy and it is possible to have multiple proxies with different budgets, e.g. one for user-facing requests and one for internal requests. In addition, the HTTP ingress proxy respects the `Request-Timeout` header and will set the time budget to match.

## Call Stack Depth

In addition to the time budget, the depth of the call stack is also propagated downstream, incremented by 1 on each network hop. The framework will error out if the call stack depth reached a level of 64 nested calls, which most likely indicates an infinite loop.
