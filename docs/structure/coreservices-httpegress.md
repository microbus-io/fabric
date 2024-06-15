# Package `coreservices/httpegress`

The HTTP egress proxy is a core microservice that relays HTTP requests to external non-`Microbus` URLs. It is a thin wrapper over the standard `net/http` client but provides the following benefits:

* Requests are easily mockable in tests
* The [time budget](../blocks/time-budget.md) is correctly taken into account
* Requests are traced in Jaeger
* Requests are metered in Prometheus and Grafana

To make a request via the egress proxy, use `Get`, `Post` or `Do` methods of the client.
To set a timeout shorter than the time budget of the current context, use `context.WithTimeout`.
For example:

```go
req, _ := http.NewRequest("DELETE", "https://example.com/ex/5", nil)
req.Header.Set("Authentication", "Bearer " + token)
shortCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
resp, err := httpegressapi.NewClient(svc).Do(shortCtx, req)
cancel()
if err != nil {
	return err
}
```

To mock the egress microservice, create `Mock` of it and handle the request manually:

```go
	mock := NewMock()
	mock.MockMakeRequest = func(w http.ResponseWriter, r *http.Request) (err error) {
		req, _ := http.ReadRequest(bufio.NewReader(r.Body))
		if req.Method == "DELETE" && req.URL.String() == "https://example.com/ex/5" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"deleted":true}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return nil
	}
```

Note that the single endpoint of the HTTP egress microservice `MakeRequest` is listening on internal `Microbus` port `:444` rather than `:443`. That is because port `:443` is open by default to the outside via the HTTP ingress proxy.
