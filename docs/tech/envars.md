# Environment Variables

`Microbus` microservices use environment variables for four main purposes:

* Values of user-defined configuration properties of microservices
* Customizing the [NATS connection settings](./natsconnection.md)
* Determining the deployment environment (PROD, LAB, LOCAL, UNITTEST)
* Altering the plane of communication

## Microservice Configuration

It's quite common for microservices to define configuration properties whose values can be set without code changes, possibly by non-engineers. Connection strings to a database, number of rows to display in a table, a timeout value, are all examples of configuration properties.

In the code, configs are fetched using the `Config(name string) (value string)` method of the `Connector`. Behind the scenes, this method is looking for environment variables that match the name of the config and the host name of the microservice. In the following example

```
c := NewConnector()
c.SetHostName("www.example.com")
c.Startup()
foo := c.Config("Bar")
```

The value of `foo` will be fetched from the following environment variables, whichever is located first, in order.

* `MICROBUS_WWWEXAMPLECOM_BAR`
* `MICROBUS_EXAMPLECOM_BAR`
* `MICROBUS_COM_BAR`
* `MICROBUS_BAR`

The hierarchical approach allows for the sharing of config values across microservices. A database connection string is an example of a config that makes sense to share.

## NATS Connection Settings

All microservices must connect to NATS in order to run. These environment variables are used to customize that connection.

* `NATS_URL` - The location of the NATS cluster
* `NATS_USER` - Credentials
* `NATS_PASSWORD` - Credentials
* `NATS_TOKEN` - Credentials

Learn more about [NATS connection settings](./natsconnection.md).

## Deployment Environment

`Microbus` recognizes four deployment environments:

* `PROD` for production deployments
* `LAB` for fully-functional non-production deployments such as dev integration, testing, staging, etc.
* `LOCAL` for when developing locally
* `UNITTEST` for when running applications inside unit tests

The deployment environment impacts certain aspects of the framework such as the log format and log level.

The deployment environment is set according to the value of the `MICROBUS_DEPLOYMENT` environment variable. If it is empty, `PROD` is assumed, unless connecting to NATS on `localhost:4222`, in which case `LOCAL` is assumed.

## Plane of Communication

The plane of communication is a unique prefix set for all communications sent or received over NATS.
It is used to isolate communication among a group of microservices over a NATS cluster
that is shared with other microservices.

If not explicitly set via the `SetPlane(plane string)` method of the `Connector`, the value is pulled from the `MICROBUS_PLANE` environment variable. The plane must include only alphanumeric characters and is case-sensitive.

This is an advanced feature and in most cases there is no need to customize the plane of communications.
