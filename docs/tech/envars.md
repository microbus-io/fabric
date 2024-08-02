# Environment Variables

The `Microbus` framework uses environment variables for various purposes:

* Initializing the connection to NATS
* Identifying the deployment environment (`PROD`, `LAB`, `LOCAL`, `TESTING`)
* Designating a plane of communication
* Enabling output of debug-level messages
* Configuring the URL to the OpenTelemetry collector endpoint
* Designating a geographic locality

Environment variables may also be set by placing an `env.yaml` file in the working directory of the executable running the microservice. The bundled example application includes such a file at `main/env.yaml`.

## NATS Connection

Before connecting to NATS, a microservice can't communicate with other microservices and therefore it can't reach the configurator microservice to fetch the values of its config properties. Connecting to NATS therefore must precede configuration which means that initializing the NATS connection itself can't be done using the standard configuration pattern. Instead, the [NATS connection is initialized using environment variables](../tech/nats-connection.md): `MICROBUS_NATS`, `MICROBUS_NATS_USER`, `MICROBUS_NATS_PASSWORD` and `MICROBUS_NATS_TOKEN`.

## Deployment

The `MICROBUS_DEPLOYMENT` environment variable determines the [deployment environment](../tech/deployments.md) of the microservice: `PROD`, `LAB`, `LOCAL` or `TESTING`. If not specified, `PROD` is assumed, unless connecting to `nats://localhost:4222` or `nats://127.0.0.1:4222` in which case `LOCAL` is assumed.

## Plane of Communication

The plane of communication is a unique prefix set for all communications sent or received over NATS.
It is used to isolate communication among a group of microservices over a NATS cluster
that is shared with other microservices.

If not explicitly set via the `SetPlane` method of the `Connector`, the value is pulled from the `MICROBUS_PLANE` environment variable. The plane must include only alphanumeric characters and is case-sensitive.

Applications created with `application.NewTesting` set a random plane to eliminate the chance of collision when tests are executed in parallel in the same NATS cluster, e.g. using `go test ./... -p=8`.

This is an advanced feature and in most cases there is no need to customize the plane of communications.

## Locality

A geographic locality, when provided, is used by `Microbus` to optimize routing of unicast communications. A microservice making a unicast request will prioritize microservices that most resemble its own locality. For example, if the upstream microservice is located in `az1.dc2.west.us` and the downstream microservice is located in both `az2.dc2.west.us` and `az1.dc1.east.us`, the request will be directed to the former because they share the longer common suffix `dc2.west.us`.

Locality-aware routing works when both upstream and downstream microservices designate a locality, typically by means of the `MICROBUS_LOCALITY` environment variable. The pattern is similar to that of a standard hostname, with the most specific location first.

The special value `AWS` can be used when deployed on AWS to determine the availability zone automatically by making a request to the meta-data service at `http://169.254.169.254/latest/meta-data/placement/availability-zone`. The availability zone will then be used as the basis for the locality of the microservice. For example, availability zone `us-east-1b` is transformed to locality `b.1.east.us`.

Similarly, the special value `GCP` can be used when deployed on GCP to determine the availability zone from `http://metadata.google.internal/computeMetadata/v1/instance/zone`. For example, availability zone `us-east1-b` is transformed to locality `b.1.east.us`

Caution: the microservice will fail to start if the availability zone cannot be determined.

## Logging

Setting the `MICROBUS_LOG_DEBUG` environment variable to any non-empty value is required for microservices to [log](../blocks/logging.md) debug-level messages.
 
## Distributed Tracing

`Microbus` is using OpenTelemetry to send tracing spans to Jaeger via the OTLP HTTP collector. In order to enable tracing, the endpoint of the collector must be set in the `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` or `OTEL_EXPORTER_OTLP_ENDPOINT` environment variables.
