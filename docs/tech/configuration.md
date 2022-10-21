# Configuration

It's quite common for microservices to want to define properties whose values can be configured without code changes, often by non-engineers. Connection strings to a database, number of rows to display in a table, a timeout value, are all examples of configuration properties.

In `Microbus`, the microservice owns the definition of its config properties, while [the configurator system microservice owns their values](../structure/services-configurator.md). The former means that microservices are self-descriptive and independent, which is important in a distributed development environment. The latter means that managing configuration values and pushing them to a live system are centrally controlled, which is important because configs can contain secrets and because pushing the wrong config values can destabilize a deployment.

Configuration properties must first be defined by the microservice using `DefineConfig`. In the following example, the property is named `Foo` and given the default value `Bar` and a regexp requirement that the value is comprised of at least one letter and only letters.

```go
con.New("www.example.com")
con.DefineConfig("Foo", cfg.DefaultValue("Bar"), cfg.Validation("str [A-Za-z]+"))
```

Immediately upon startup, the microservice contacts the configurator microservice to ask for the values of its configuration properties. If an override value is available at the configurator, it is set as the new value of the config property; otherwise, the default value of the config property is set instead.

Configs are accessed using the `Config(name string) (value string)` method of the `Connector`. 

```go
foo := con.Config("Foo")
```

Note that configuration property names are case-insensitive.

The microservice keeps listening for a command on the control subscription `:888/config/refresh` and will respond by refetching config values from the configurator upon. The configurator issues this command on a periodic basis to ensure that all microservices always have the latest config. If new values are received, they will be set appropriately and the `OnConfigChanged` callback will be invoked.

```go
con.SetOnConfigChanged(func (ctx context.Context, changed map[string]bool) error {
	if changed["Foo"] {
		con.LogInfo(ctx, "Foo changed", log.String("to", con.Config("Foo")))
	}
	return nil
})
```
