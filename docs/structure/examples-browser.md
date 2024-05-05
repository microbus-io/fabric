# Package `examples/browser`

The `browser.example` microservice implement a single endpoints, `/browse` that renders an HTML page with a form that takes in a URL and fetches the source code of the page. This example demonstrates how to use the [HTTP egress core microservice](./coreservices-httpegress.md) as well as how to mock it in tests.

Visit http://localhost:8080/browser.example/browse to play around with the simple UI.
