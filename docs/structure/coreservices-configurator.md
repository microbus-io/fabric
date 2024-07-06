# Package `coreservices/configurator`

The configurator is a core microservice of `Microbus` and it must be included with practically all applications. Microservices that define config properties will not start if they cannot reach the configurator. This is why you'll see the configurator included in most self-contained apps, such as in `main/main.go`:

```go
func main() {
	app := application.New()
	app.Add(
		// Configurator should start first
		configurator.NewService(),
	)
	app.Add(
		// ...
	)
	app.Run()
}
```

The configurator is the owner of the values of the config properties. It is responsible for loading those values and disseminating them to the microservices. For this purpose, the configurator looks in the current working directory for the file `config.yaml` which associates a value with a property name and a domain name.

```yaml
domain.name:
  PropertyName: PropertyValue
```

Note that both domain names and property names are case-insensitive.

A property value is applicable to a microservice that (1) defines a property with the same name, and (2) whose hostname equals or is a sub-domain of the domain name. For example, in the following example, the value of `Foo` is applicable only to the `www.example.com` microservice, while the value of `Moo` is applicable to both `www.example.com` and `zzz.example.com`.

```yaml
www.example.com:
  Foo: Bar
example.com:
  Moo: Cow
```

```go
www := connector.New("www.example.com")
www.DefineConfig("Foo")
www.DefineConfig("Moo")

zzz := connector.New("zzz.example.com")
zzz.DefineConfig("Moo")
zzz.DefineConfig("Zoo")
```

Every 20 minutes the configurator broadcasts the command `https://all:888/config-refresh` to instruct all microservices to refresh their config. The microservices will respond by calling the configurator's `https://configurator.core/values` endpoint to fetch the current values. This guarantees that microservices do not fall out of sync with their configuration, at least not for long.

The `/refresh` endpoint can be called manually to force a refresh at any time.
