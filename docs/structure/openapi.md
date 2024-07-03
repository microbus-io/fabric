# Package `openapi`

All `Microbus` microservices [produce OpenAPI documents](../blocks/openapi.md) describing their endpoints. The `openapi` package is an internal package that supports the generation of these documents by:

* Modeling the OpenAPI document using Go structs to facilitate marshaling it as YAML
* Translating Go primitives to the corresponding OpenAPI types
* Traversing the dependency tree of complex types (structs) using reflection and translating them into the corresponding OpenAPI components
