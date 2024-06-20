# RPC vs REST API styles

Two of the most common styles of web API design are RPC over JSON and REST. The choice between them is a matter of preference rather than functional and is akin to the choice between [tabs and spaces](https://www.youtube.com/watch?v=SsoOG6ZeyUI). Luckily, with `Microbus` you don't have to choose: you can have both.

## RPC over JSON

RPC is an abbreviation of remote procedure call and indeed this API style comes from a backend-centric perspective where each operation on the backend is a procedure. Endpoints on the backend are mapped to a URL path that reflects their name. Input arguments are passed as JSON in the body of the HTTP request or as query arguments. Output arguments are returned in the body of the HTTP response. In `Microbus`, RPC is the default API style exactly because it is a consistent and unambiguous way to map the HTTP request to the underlying Go implementation of the endpoint.

Consider the following simple example:

```yaml
functions:
  - signature: Add(x int, y int) (sum int)
    description: Add adds two integers and returns their sum.
```

`Microbus` automatically generates marshaling and unmarshaling code that can process any of the following `GET` or `POST` requests:

In a `GET` request, input arguments are read from the query arguments.

```http
GET /add?x=5&y=6 HTTP/1.1
```

In a `POST` request, the input arguments are read from the request body. Each of the arguments is expected to be a named field in a single JSON object.

```http
POST /add HTTP/1.1
Content-Type: application/json

{"x":5,"y":6}
```

Alternatively, a URL-encoded form data can also be `POST`-ed.

```http
POST /add HTTP/1.1
Content-Type: application/x-www-form-urlencoded

x=5&y=6
```

In all three cases, the response is a JSON object where each of the return values (in this example only one) is a named field:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"sum":11}
```

## RESTful

The REST API style comes from a web-centric perspective. Its philosophy is that everything on the web is a resource that can be identified with a URI and so a web API should reflect that. In REST, the request's path represents the resource and the HTTP method (`GET`, `POST`, `DELETE`, `PUT`) represents the operation done on that resource. This style works best for CRUD APIs but may not translate well when the API has no object that is being worked on. For example, the `Add` example from earlier doesn't translate well to REST.

The following example defines a typical RESTful API for CRUD operations on an `Object`. Notice how it utilizes the [HTTP magic arguments](./httparguments.md) `httpRequestBody`, `httpResponseBody` and `httpStatusCode`.

```yaml
functions:
  - signature: Create(httpRequestBody *Object) (id int, httpStatusCode int)
    description: Create creates an object.
    method: POST
    path: /objects
  - signature: Read(id int) (httpResponseBody *Object)
    description: Read reads an object.
    method: GET
    path: /objects/{id}
  - signature: Update(id int, httpRequestBody *Object)
    description: Update updates an existing object.
    method: PUT
    path: /objects/{id}
  - signature: Delete(id int)
    description: Delete deletes an existing object.
    method: DELETE
    path: /objects/{id}
  - signature: ListAll(sortBy string, limit int, offset int) (httpResponseBody []*Object)
    description: ListAll returns all objects.
    method: GET
    path: /objects
```

`Create` expects a `POST` request with the object to be created in the body of the request. It returns the assigned `id` along with HTTP status code `201`.

```http
POST /objects HTTP/1.1
Content-Type: application/json

{"foo":"bar","count":5,"etc":"..."}
```

```http
HTTP/1.1 201 Created
Content-Type: application/json

{"id":1}
```

`Read` expects a `GET` request with the ID of the object in the path. It returns the matching object.

```http
GET /objects/1 HTTP/1.1
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"foo":"bar","count":5,"etc":"..."}
```

`Update` expects a `PUT` request with the ID of the object in the path and the object to be created in the body of the request. It returns nothing.

```http
PUT /objects/1 HTTP/1.1
Content-Type: application/json

{"foo":"bar","count":6,"etc":"..."}
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{}
```

`Delete` expects a `DELETE` request with the ID of the object in the path. It returns nothing.

```http
DELETE /objects/1 HTTP/1.1
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{}
```

`ListAll` expects a `GET` request with sort order and cursor information as query arguments. It returns a list of objects.

```http
GET /objects?sortBy=count&limit=3&offset=0 HTTP/1.1
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

[
    {"foo":"bar","count":6,"etc":"..."},
    {"foo":"baz","count":8,"etc":"..."},
    {"foo":"bam","count":9,"etc":"..."}
]
```

## Best of Both Worlds

Even if you have a preference as to one of these API styles, users of your API are likely to have a different preference. Luckily, implementing both styles is rather simple: create the RPC-styled endpoint first, then create the RESTful endpoint and call the RPC-styled endpoint under the hood.

Here's how that will look like for the `Read` endpoint discussed earlier:

```yaml
functions:
  - signature: Read(id int) (obj *Object)
    description: Read an object by ID.
  - signature: ReadREST(id int) (httpResponseBody *Object)
    description: Read an object by ID.
    method: GET
    path: /objects/{id}
```

The implementation of the RPC-styled endpoint may then look something like this:

```go
func (svc *Service) Read(ctx context.Context, id int) (obj *Object, err error) {
    row, err := svc.sqlDatabase.QueryRowContext(ctx, "SELECT foo, count, etc FROM objects WHERE id=?", id)
    if err != nil {
        return nil, errors.Trace(err)
    }
    obj := &Object{}
    err = row.Scan(&obj.Foo, &obj.Count, &obj.Etc)
    if err != nil {
        if errors.Is(sql.ErrNoRows) {
            return Object{}, errors.Newcf(http.StatusNotFound, "object %v not found", id)
        }
        return nil, errors.Trace(err)
    }
    return obj, nil
}
```

The RESTful endpoint can then simply delegate the call to the above:

```go
func (svc *Service) ReadREST(ctx context.Context, id int) (httpResponseBody *Object, err error) {
    return svc.Read(ctx, id)
}
```
