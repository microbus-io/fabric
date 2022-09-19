# Quick Start

## Install and Run NATS

From the root folder of this project:

```
go get github.com/nats-io/nats-server
go build github.com/nats-io/nats-server
./nats-server -D -V
```

It's recommended to start the NATS server in a separate terminal window to better be able to see the action. 

## Run the Examples

To run the examples:

```
cd examples/main
go run main.go
```

This sets the working directory to `examples/main` and makes sure that the `examples/main/env.yaml` file is located.

If you're using Visual Studio Code, you may alternatively open or focus on `examples/main/main.go` and press `F5`.

Try the following URLs in your browser:

* http://localhost:8080/calculator.example/arithmetic?x=5&op=*&y=8
* http://localhost:8080/calculator.example/square?x=5
* http://localhost:8080/echo.example/echo
* http://localhost:8080/echo.example/who
* http://localhost:8080/helloworld.example/hello?name=Gopher

Feel free to play with the query argumnets.
