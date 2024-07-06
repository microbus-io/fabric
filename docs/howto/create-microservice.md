# Creating a Microservice

#### Step 1: Create a Directory

Create a new directory for the new microservice. If you expect the solution to have a large number of microservice, you might want to create a nested structure.

```cmd
mkdir mydomain/myservice
```

#### Step 2: Initialized the Code Generator

Create `mydomain/myservice/doc.go` with the `go:generate` instruction that will run the code generator.

```go
//go:generate go run github.com/microbus-io/fabric/codegen

package myservice
```

#### Step 3: Generate `service.yaml`

From within the directory, run `go generate` to create an empty `service.yaml` template.

```cmd
cd mydomain/myservice
go generate
```

#### Step 4: Declare the Functionality

Fill in the [characteristics of the microservice](../tech/service-yaml.md) in `service.yaml`, and `go generate` to generate the skeleton code for the new microservice and its client stubs.

#### Step 5: Implement the Business Logic

Implement the functionality of the microservice in `service.go` and [test](../blocks/integration-testing.md) it in `integration_test.go`.

Run `go generate` a final time to update the version number of the microservice.

#### Step 6: Add to the Application

Add the new microservice to the application in `main.go`:

```go
app.Add(
    // Add solution microservices here
    myservice.NewService(),
)
```

Repeat this step for each new microservice.
