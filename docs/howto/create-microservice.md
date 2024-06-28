# Creating a Microservice

#### Step 1: Create a Directory

Create a new directory for the new microservice. If you expect the solution to have a large number of microservice, you might want to create a nested structure.

```cmd
mkdir mydomain/myservice
```

Create `mydomain/myservice/doc.go`.

```go
//go:generate go run github.com/microbus-io/fabric/codegen

package myservice
```

#### Step 2: Initialized the Code Generator

From within the directory, run `go generate` to create an empty `service.yaml` template.

```cmd
cd myservice
go generate
```

#### Step 3: Declare the Functionality

Fill in the [characteristics of the microservice](../tech/service-yaml.md) in `service.yaml`, and `go generate` to generate the skeleton code for the new microservice and its client stubs.

#### Step 4: Implement the Business Logic

Implement the functionality of the microservice in `service.go` and test it in `integration_test.go`.

Run `go generate` a final time to update the version number of the microservice.

#### Step 5: Add to the Application

Add the new service to the application in `main.go`:

```go
app.Include(
    // Add solution microservices here
    myservice.NewService(),
)
```

Repeat this step for each new microservice.
