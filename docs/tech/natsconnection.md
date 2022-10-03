# NATS Connection Settings

NATS is the communication medium of microservices and all microservices must be connected to NATS in order to send and receive messages. By default, microservices attempt to connect to NATS on `nats://127.0.0.1:4222` using a plain (unsecure) TCP connection. The `NATS_URL` environment variable is used to customize this connection URL.

NATS supports [various authentication methods](https://docs.nats.io/using-nats/developer/connecting) for connecting to the NATS cluster. The `Microbus` framework is exposing some of these via environment variables and certificate files.

The `NATS_USER` and `NATS_PASSWORD` environment variables, when present, are used to authenticate with simple [username and password](https://docs.nats.io/using-nats/developer/connecting/userpass) credentials.

The `NATS_TOKEN` environment variable, when present, is used to authenticate with an [API token](https://docs.nats.io/using-nats/developer/connecting/token#connecting-with-a-token) credential.

If both a `cert.pem` and `key.pem` are present in the current working directory, they will be used as the public certificate and private key (respectively) to [secure the connection to NATS with TLS](https://docs.nats.io/using-nats/developer/connecting/tls).

If a file named `ca.pem` is present in the current working directory, it will be used as the root certificate of the certificate authority. A root certificate may be required for NATS to trust other certificates.

Note that when included in an `Application`, all microservices share the same NATS connection settings. There is currently no way to configure microservices separately when they run in the same executable.
