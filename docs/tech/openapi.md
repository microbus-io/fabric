# OpenAPI

The [OpenAPI](https://www.openapis.org) specification is a formal standard for describing HTTP APIs in a YAML or JSON document. It is the world's most widely used API description standard.

`Microbus` leverages the knowledge it has about the structure of a microservice to automatically generate an OpenAPI document for each of its public web and functional endpoints. A separate OpenAPI document is created for each port of each microservice. Here's an (abbreviated) example of an OpenAPI document generated for the `:443` endpoints of the [calculator microservice](../structure/examples-calculator.md):

```yaml
openapi: 3.0.0
info:
    title: calculator.example
    description: The Calculator microservice performs simple mathematical operations.
    version: "141"
servers:
    - url: http://localhost:8080/
paths:
    /calculator.example:443/arithmetic:
        post:
            summary: Arithmetic(x int, op string, y int) (result int)
            description: Arithmetic perform an arithmetic operation between two integers x and y given an operator op.
            requestBody:
                required: true
                content:
                    application/json:
                        schema:
                            $ref: '#/components/schemas/Arithmetic_in'
            responses:
                "200":
                    description: OK
                    content:
                        application/json:
                            schema:
                                $ref: '#/components/schemas/Arithmetic_out'
components:
    schemas:
        Arithmetic_in:
            type: object
            properties:
                op:
                    type: string
                x:
                    type: integer
                    format: int64
                "y":
                    type: integer
                    format: int64
        Arithmetic_out:
            type: object
            properties:
                result:
                    type: integer
                    format: int64
```

An `openapi.json` endpoint is created for each port of each microservice to serve the OpenAPI document. For example, the OpenAPI endpoint of the `:443` endpoints of the calculator microservice is located at https://localhost:8080/calculator.example/openapi.json. In `Microbus` ports are used to control access to a microservice's endpoints. A separate documents for each port follows that philosophy and exposes only the endpoints on the same port as the request's.

OpenAPI generation can be disabled in `service.yaml` using the `openApi: false` directive for the entire microservice and/or for an individual endpoint. For example:

```yaml
# General
#
# host - The hostname of the microservice
# description - A human-friendly description of the microservice
# integrationTests - Whether or not to generate integration tests (defaults to true)
# openApi - Whether or not to generate an OpenAPI document at openapi.json (defaults to true)
general:
  host: calculator.example
  description: The Calculator microservice performs simple mathematical operations.
  openApi: false
```

The [OpenAPI portal core microservice](../structure/coreservices-openapiportal.md) aggregates the OpenAPI endpoints of all microservices on the bus and renders an HTML page that lists them to a human reader.

[Swagger](https://swagger.io) is a set of popular tools for working with APIs in general and OpenAPI in particular. The [OpenAPI editor](https://editor-next.swagger.io) is an especially useful one that allows editing and exploring OpenAPI documents online.
