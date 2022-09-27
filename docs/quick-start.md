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

Fetch the code:

```
mkdir github.com/microbus-io
cd github.com/microbus-io
git clone https://github.com/microbus-io/fabric
```

Run the examples:

```
cd examples/main
go run main.go
```

It is important to set the working directory to `examples/main` so that the `examples/main/env.yaml` file is located.

If you're using Visual Studio Code, simply press `F5`. The `.vscode/launch.json` file includes a launch configuration for running `examples/main`.

Try the following URLs in your browser:

* http://localhost:8080/calculator.example/arithmetic?x=5&op=*&y=8
* http://localhost:8080/calculator.example/square?x=5
* http://localhost:8080/calculator.example/square?x=not-a-number
* http://localhost:8080/echo.example/echo
* http://localhost:8080/echo.example/who
* http://localhost:8080/helloworld.example/hello?name=Gopher

Feel free to experiment with different values for the query arguments.
