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
c := connector.New("www.example.com")
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
