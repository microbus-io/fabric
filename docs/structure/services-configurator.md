# Package `services/configurator`

## Introduction

The configurator is a core microservice of `Microbus` and it must be included with practically all applications. Microservices that define config properties will not start if they cannot reach the configurator. This is why you'll see it listed first most apps, such as in `examples/main/main.go`:

```go
func main() {
	app := application.New(
		configurator.NewService(),
		httpingress.NewService(),
		hello.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		calculator.NewService(),
	)
	app.Run()
}
```

## Loading Values

The configurator is the owner of the values of the config properties. It is responsible for loading those values and disseminating them to the microservices. For this purpose, the configurator looks for two files in the current working directory.

If present, the file `config.yaml` is used to specify property values explicitly. In the file, values are associated with a property name and a domain name.

```yaml
domain.name:
  PropertyName: PropertyValue
```

Note that both domain names and property names are case-insensitive.

A property value is applicable to a microservice that define a property with the same name, and whose host name equals or is a sub-domain of the domain name. For example, in the following example, the value of `Foo` is applicable only to the `www.example.com` microservice, while the value of `Moo` is applicable to both `www.example.com` and `zzz.example.com`.

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

If present, the file `configimport.yaml` is used to import `config.yaml`s from a URL. The file includes a list of directives such as these:

```yaml
- from: http://www.example.com/config.yaml
  import: example.com, nowhere.com
```

This tells the configurator to download the file at `http://www.example.com/config.yaml` and import the property values included therein, but only those whose domain name equals to or is a sub-domain of `example.com` or `nowhere.com`. This simple ACL mechanism allows for the delegation of ownership of configuration values in a distributed team.

## Refreshing

The configurator monitors these files every 5 minutes and issues a command `https://all:888/config/refresh` to tell all microservices to refresh their config if it detects a change in the local `config.yaml` or in any of the remote ones. It issues the refresh command every 20 minutes whether or not a change is detected in order to guarantees that microservices do not fall out of sync with their configuration. In addition, the `/refresh` endpoint can be called manually to force a refresh at any time.
