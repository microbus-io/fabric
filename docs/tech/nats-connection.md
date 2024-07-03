# NATS Connection Settings

NATS is the communication medium of microservices and all microservices must first and foremost be connected to NATS in order to send and receive messages. By default, microservices attempt to connect to NATS on `nats://127.0.0.1:4222` using a plain (unsecure) TCP connection. The `MICROBUS_NATS` [environment variable](../tech/envars.md) is used to customize this connection URL.

NATS supports [various authentication methods](https://docs.nats.io/using-nats/developer/connecting) for connecting to the NATS cluster. The `Microbus` framework is exposing some of these via [environment variables](../tech/envars.md) and certificate files.

The `MICROBUS_NATS_USER` and `MICROBUS_NATS_PASSWORD` [environment variables](../tech/envars.md), when present, are used to authenticate with simple [username and password](https://docs.nats.io/using-nats/developer/connecting/userpass) credentials.

The `MICROBUS_NATS_TOKEN` [environment variable](../tech/envars.md), when present, is used to authenticate with an [API token](https://docs.nats.io/using-nats/developer/connecting/token#connecting-with-a-token) credential.

NATS needs a public certificate and a private key in order to [secure the connection to NATS with TLS](https://docs.nats.io/using-nats/developer/connecting/tls). `Microbus` looks for the certs in the current working directory under the names `cert.pem` and `key.pem`.

A root certificate authority (CA) certificate may be required by NATS to trust other certificates. `Microbus` looks for the CA certificate file in the current working directory under the name `ca.pem`.
