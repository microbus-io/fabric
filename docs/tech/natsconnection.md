# NATS Connection Settings

NATS is the communication medium of microservices and all microservices must be connected to NATS in order to send and receive messages. By default, microservices attempt to connect to NATS on `nats://127.0.0.1:4222` using a plain (unsecure) TCP connection. The `NATS` [configuration](./configuration.md) property is used to customize this connection URL.

NATS supports [various authentication methods](https://docs.nats.io/using-nats/developer/connecting) for connecting to the NATS cluster. The `Microbus` framework is exposing some of these via environment variables and certificate files.

The `NATSUser` and `NATSPassword` configuration properties, when present, are used to authenticate with simple [username and password](https://docs.nats.io/using-nats/developer/connecting/userpass) credentials.

The `NATSToken` configuration property, when present, is used to authenticate with an [API token](https://docs.nats.io/using-nats/developer/connecting/token#connecting-with-a-token) credential.

NATS needs a public certificate and a private key in order to [secure the connection to NATS with TLS](https://docs.nats.io/using-nats/developer/connecting/tls). `Microbus` looks for the certs in the current working directory. Similar to configuration properties, here too a hierarchical naming convention is used. For the microservice `www.example.com`, certs will be scanned in the following order:

* `www.example.com-cert.pem` and `www.example.com-key.pem`
* `example.com-cert.pem` and `example.com-key.pem`
* `com-cert.pem` and `com-key.pem`
* `all-cert.pem` and `all-key.pem`

A root certificate authority (CA) certificate may be required by NATS to trust other certificates. `Microbus` scans for the CA certificate file in the current working directory in the following order:

* `www.example.com-ca.pem`
* `example.com-ca.pem`
* `com-ca.pem`
* `all-ca.pem`
