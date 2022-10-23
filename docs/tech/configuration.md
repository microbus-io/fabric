# Configuration

It's quite common for microservices to want to define properties whose values can be configured without code changes, often by non-engineers. Connection strings to a database, number of rows to display in a table, a timeout value, are all examples of configuration properties.

In `Microbus`, the microservice owns the definition of its config properties, while [the configurator system microservice owns their values](../structure/services-configurator.md). The former means that microservices are self-descriptive and independent, which is important in a distributed development environment. The latter means that managing configuration values and pushing them to a live system are centrally controlled, which is important because configs can contain secrets and because pushing the wrong config values can destabilize a deployment.

Configuration properties must first be defined by the microservice using `DefineConfig`. In the following example, the property is named `Foo` and given the default value `Bar` and a regexp requirement that the value is comprised of at least one letter and only letters.

```go
con.New("www.example.com")
con.DefineConfig("Foo", cfg.DefaultValue("Bar"), cfg.Validation("str [A-Za-z]+"))
```

The following options are available:
* `cfg.DefaultValue` specifies a default value for the property when one is not provided by the configurator
* `cfg.Validation` uses a pattern to validate values before they are set
	* `str` - Plain text, no validation
	* `str [a-zA-Z0-9]+` - Text with regexp validation
	* `bool` - Must be `true` or `false`
	* `int` - An integer, no validation
	* `int [0,60]` - An integer in range
	* `float` - A decimal number, no validation
	* `float [0.0,1.0)` - A decimal number in range
	* `dur` - A duration such as `7h3m45s500ms300us100ns`
	* `dur (0s,24h]` A duration in range
	* `set Red|Green|Blue` - A limited set of options
	* `url` - A URL
	* `email` - An email address, either `Joe <joe@example.com>` or just `joe@example.com`
	* `json` - A valid JSON string
* `cfg.Secret` indicates that the value of this property is a secret and should not be logged
* `cfg.Description` is intended to explain the purpose of the config property and how it will impact the microservice

Immediately upon startup, the microservice contacts the configurator microservice to ask for the values of its configuration properties. If an override value is available at the configurator, it is set as the new value of the config property; otherwise, the default value of the config property is set instead.

Configs are accessed using the `Config(name string) (value string)` method of the `Connector`. 

```go
foo := con.Config("Foo")
```

Note that configuration property names are case-insensitive.

The microservice keeps listening for a command on the control subscription `:888/config/refresh` and will respond by refetching config values from the configurator upon. The configurator issues this command on a periodic basis (every 20 minutes) to ensure that all microservices always have the latest config. If new values are received, they will be set appropriately and the `OnConfigChanged` callback will be invoked.

```go
con.SetOnConfigChanged(func (ctx context.Context, changed map[string]bool) error {
	if changed["Foo"] {
		con.LogInfo(ctx, "Foo changed", log.String("to", con.Config("Foo")))
	}
	return nil
})
```
