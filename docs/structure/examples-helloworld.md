# Package `examples/helloworld`

The `helloworld.example` microservice demonstrates the minimalist classic example.

http://localhost:8080/helloworld.example/hello-world simply prints `Hello, World!`.

The code looks rather daunting but practically all of it was code generated. The manually coded pieces were:

In `service.yaml`:

```yaml
general:
  host: helloworld.example
  description: The HelloWorld microservice demonstrates the minimalist classic example.

webs:
  - signature: HelloWorld()
    description: HelloWorld prints the classic greeting.
```

In `HelloWorld` in `service.go`:

```go
w.Write([]byte("Hello, World!"))
return nil
```

In `TestHelloworld_HelloWorld` in `integration_test.go`:

```go
ctx := Context()
HelloWorld_Get(t, ctx, "").BodyContains("Hello, World!")
```

And finally, the microservice was added to the app in `main` in `main/main.go`.

```go
app.Add(
	helloworld.NewService(),
)
```
