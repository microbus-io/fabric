# Package `coreservices/openapiportal`

All `Microbus` microservices [produce OpenAPI documents](../blocks/openapi.md) describing their endpoints.
The OpenAPI portal core microservice renders an HTML portal page that lists all microservices that have an OpenAPI endpoint on the same port as the request. For example, the portal page at the internal `Microbus` address of `https://openapi:443` finds all microservices with open endpoints on port `:443`.
