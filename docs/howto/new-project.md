# Bootstrapping a New Project

Follow these steps to create a new project that takes advantage of `Microbus`.

#### Step 1: Choose Package Name

Choose a name for the package to hold your project, for example `github.com/mycompany/mysolution`.

#### Step 2: Create a Matching Directory

Make a directory to hold your projects files. It's recommended to follow the package structure.

```cmd
mkdir github.com/mycompany/mysolution
```

#### Step 3: Init the Go Project

Init the Go project with the name of the package.

```cmd
cd github.com/mycompany/mysolution
go init github.com/mycompany/mysolution
```

#### Step 4: Get `Microbus`

Add `Microbus` to `go.mod` using:

```cmd
go get github.com/microbus-io/fabric
```

#### Step 5: Create `main` Package

Make a `main` subdirectory under your project's root directory.

```cmd
mkdir main
```

Create `main/main.go`:

```go
package main

import (
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/coreservices/configurator"
	"github.com/microbus-io/fabric/coreservices/httpegress"
	"github.com/microbus-io/fabric/coreservices/httpingress"
	"github.com/microbus-io/fabric/coreservices/metrics"
	"github.com/microbus-io/fabric/coreservices/openapiportal"
)

func main() {
	app := application.New()
	app.Add(
		configurator.NewService(),
	)
	app.Add(
		httpegress.NewService(),
		openapiportal.NewService(),
		metrics.NewService(),
	)
	app.Add(
		// Add solution microservices here
	)
	app.Add(
		httpingress.NewService(),
		// smtpingress.NewService(),
	)
	app.Run()
}
```

Create `main/env.yaml` to be able to set the [environment variables](../tech/envars.md) in code:

```yaml
# NATS connection settings
MICROBUS_NATS: nats://127.0.0.1:4222
# MICROBUS_NATS_USER:
# MICROBUS_NATS_PASSWORD:
# MICROBUS_NATS_TOKEN:

# The deployment impacts certain aspects of the framework such as the log format and log verbosity level
#   PROD - production deployments
#   LAB - fully-functional non-production deployments such as dev integration, testing, staging, etc.
#   LOCAL - developing locally
#   TESTING - unit and integration testing
MICROBUS_DEPLOYMENT: LOCAL

# The plane of communication isolates communication among a group of microservices over a NATS cluster
# MICROBUS_PLANE: microbus

# Any non-empty value enables logging of debug-level messages
# MICROBUS_LOG_DEBUG: 1

# The endpoint of the OTLP HTTP collector of OpenTelemetry
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT: http://127.0.0.1:4318
# OTEL_EXPORTER_OTLP_ENDPOINT:
```

Create `main/config.yaml` where you'll store the [configuration](../blocks/configuration.md) of your microservices:

```yaml
http.ingress.core:
#  Ports: 8080
#  TimeBudget: 20s
```

#### Step 6: Visual Studio Code Launcher

If you're using Visual Studio Code, update `.vscode/launch.json` and add a configuration to run `main.go`:

```json
{
    "version": "0.2.0",
    "configurations": [
		{
			"name": "MySolution Main",
			"type": "go",
			"request": "launch",
			"mode": "auto",
			"program": "${workspaceFolder}/main",
			"cwd": "${workspaceFolder}/main"
		}
	]
}
```

#### Step 7: Create Microservices

[Create one microservice](../howto/create-microservice.md) at a time.
