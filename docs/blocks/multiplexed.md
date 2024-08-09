# Persistent Multiplexed Connections

A persistent multiplexed connection is a TCP connection that is kept open and allows bi-directional streaming exchange of requests and responses at any time. Unlike HTTP/1.1, there is no restriction to have only one request and one response for the lifetime of the connection. A multiplexed connection can transport multiple requests and responses at the same time, interwoven on the timeline, and out of order. HTTP/2 and gRPC are a multiplexed connection.

Three concurrent HTTP/1.1 requests utilize three TCP connections:

<img src="multiplexed-1.drawio.svg">
<p></p>

A multiplexed connection on the other hand can serve the three requests on a single TCP connection:

<img src="multiplexed-2.drawio.svg">
<p></p>

The benefits of a persistent multiplexed connection are:
* A single multiplexed connection is more memory efficient when compared to multiple HTTP/1.1 connections open concurrently
* There is almost no risk of running out of the approx 50,000 ephemeral ports needed to maintain a TCP connection, allowing a practically-unlimited number of concurrent requests
* The overhead of establishing the connection, especially if it is a secure connection, is incurred only once
* A connection that is persistent reduces churn in the network routing table, lending to a more stable networking topology
