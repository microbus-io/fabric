# Connectivity Liveness Test

In `Microbus`, each microservice uses a single bi-directional [persistent connection](../blocks/multiplexed.md) for both incoming and outgoing messages. Consequently, that connection is the single source of truth of its liveness. If the connection drops, the microservice can no longer send messages to other microservices. In addition, once the messaging bus detects the failed connection, it deletes the microservice's subscriptions and no longer delivers any messages to it. Messages will be delivered to the microservice's replicas instead.

In rare situations, if the messaging bus doesn't realize that a connection dropped, it might still attempt to deliver messages to the disconnected microservice. These messages will be lost and the upstream microservice will receive an [ack timeout](../blocks/ack-or-fail.md). The situation will rectify itself when the messaging bus realizes that the downstream microservice is not responding to pings, or when the microservice reconnects.

Microservices are responsible for establishing and maintaining the connection to the messaging bus. When a connection drops, they do their best to reconnect.
