# Configuration

`Microbus` microservices are configured using environment variables and/or an `env.yaml` file.

* Values of user-defined configuration properties of microservices
* Customizing the [NATS connection settings](./natsconnection.md)
* Identifying the deployment environment (`PROD`, `LAB`, `LOCAL`)
* Designating a plane of communication

## User-Defined Properties

It's quite common for microservices to define configuration properties whose values can be set without code changes, possibly by non-engineers. Connection strings to a database, number of rows to display in a table, a timeout value, are all examples of configuration properties.

In the code, configs are fetched using the `Config(name string) (value string)` method of the `Connector`. Behind the scenes, this method is looking for environment variables that match the name of the config and the host name of the microservice. 

```go
c := NewConnector()
c.SetHostName("www.example.com")
foo := c.Config("Foo")
```

In this example, the value of `Foo` is fetched from the following environment variables, whichever is located first.
The hierarchical approach allows for the sharing of config values among microservices. A database connection string is an example of a config that makes sense to share among most if not all microservices.

* `MICROBUS_WWWEXAMPLECOM_FOO`
* `MICROBUS_EXAMPLECOM_FOO`
* `MICROBUS_COM_FOO`
* `MICROBUS_ALL_FOO`

If not defined by an environment variable, the value of `Foo` is fetched from an `env.yaml` file that must be located in the current working directory. Here too a hierarchical structure is leveraged to enable sharing configs by looking for the value under `www.example.com`, `example.com`, `com` and `all` sections, whichever is located first. 

```yaml
# env.yaml

www.example.com:
  Foo: Bar

example.com:
  Foo: Baz

com:
  Foo: Bag

all:
  Foo: Bam
```

Note: Host names, configuration property names and environment variable names are all case-insensitive.

## NATS Configuration Properties

All microservices must connect to NATS in order to run. These configuration properties are used to [customize the connection to NATS](./natsconnection.md).

* `NATS` - The location of the NATS cluster
* `NATSUser` - Credentials
* `NATSPassword` - Credentials
* `NATSToken` - Credentials

These are standard configuration properties and their values can be set by either environment variables or an `env.yaml` file as described earlier.

## Deployment

`Microbus` recognizes three deployment environments:

* `PROD` for production deployments
* `LAB` for fully-functional non-production deployments such as dev integration, testing, staging, etc.
* `LOCAL` for when developing locally or when running applications inside a testing application

The deployment environment impacts certain aspects of the framework such as the log format and log verbosity level.

The deployment environment is set according to the value of the `Deployment` configuration property. If not specified, `PROD` is assumed, unless connecting to NATS on `localhost` in which case `LOCAL` is assumed.

## Plane of Communication

The plane of communication is a unique prefix set for all communications sent or received over NATS.
It is used to isolate communication among a group of microservices over a NATS cluster
that is shared with other microservices.

If not explicitly set via the `SetPlane` method of the `Connector`, the value is pulled from the `Plane` configuration property. The plane must include only alphanumeric characters and is case-sensitive.

Applications created with `application.NewTesting` set a random plane to eliminate the chance of collision when tests are executed in parallel in the same NATS cluster, e.g. using `go test ./... -p=8`.

This is an advanced feature and in most cases there is no need to customize the plane of communications.
