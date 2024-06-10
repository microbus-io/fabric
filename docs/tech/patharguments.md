# Path Arguments

## Fixed Path

In the typical case, endpoints of a microservice have fixed URLs at which they are reachable. Consider the following `service.yaml` specification.

```yaml
general:
  host: calculator.example

functions:
  - signature: Add(x int, y int) (sum int)
    description: Add adds two integers and returns their sum.
```

The `Add` endpoint is reachable at the internal `Microbus` URL of `https://calculator.example/add` or the external URL of `https://localhost:8080/calculator.example/add` assuming that the ingress proxy is listening at `localhost:8080`. The arguments `x` and `y` of the function are unmarshaled from the request query argument or from the body of the request.

```http
GET /add?x=5&y=5 HTTP/1.1
Host: calculator.example
```

```http
POST /add?x=5&y=5 HTTP/1.1
Host: calculator.example

{"x":5,"y":5}
```

## Variable Path

A fixed path is consistent with the [RPC over JSON](./rpcvsrest.md) style of API but is insufficient for implementing a [RESTful](./rpcvsrest.md) style of API where it is common to expect input arguments in the path of the request. This is where path arguments come into play.

Consider the following `service.yaml` specification that defines a path argument `{id}` and a corresponding function argument `id int`:

```yaml
general:
  host: articles.example

functions:
  - signature: Load(id int) (article *Article)
    description: Load looks for an article by its ID.
    path: //article/{id}
    method: GET
```

The `Load` endpoint is now reachable at the internal `Microbus` wildcard URL of `https://article/{id}` where `{id}` is expected to be an `int`. The argument `id` of the function is unmarshaled from the second part of the path because it shares the same name as the path argument.

```http
GET /1 HTTP/1.1
Host: article
```

## Greediness

A typical path argument only captures the data in one part of the path, i.e. between two slashes in the path or after the last slash. Multiple arguments may be defined in the path. 

```yaml
functions:
  - signature: LoadComment(articleID int, commentID int) (comment *Comment)
    description: LoadComment looks for a comment of an article by its ID.
    path: //article/{articleID}/comment/{commentID}
    method: GET
```

A greedy path argument on the other hand captures the remainder of the path and can span multiple slashes. A greedy path argument must be the last element in the path specification. Greedy path arguments are denoted using a plus in their definition, e.g. `{greedy+}`.

```yaml
functions:
  - signature: LoadFile(filePath string) (data []byte)
    description: LoadFile looks for a file by its path.
    path: //file/{filePath+}
    method: GET
```

## Unnamed Arguments

Path arguments that are left unnamed are automatically given the names `path1`, `path2` etc. in order of their appearance. In the following example, the three unnamed path arguments are named `path1`, `path2` and `path3`. It is recommended to name path arguments and avoid this ambiguity.

```yaml
functions:
  - signature: UnnamedPathArguments(path1 int, path2 int, path3 string) (ok bool)
    description: UnnamedPathArguments demonstrates unnamed arguments.
    path: /foo/{}/bar/{}/greedy/{+}
    method: GET
```

## Conflicts

Path arguments are a form of wildcard subscription and if not crafted carefully, they may overlap. In the following case, requests to `/hello/world` will alternate between the two handlers and will result in unpredictable behavior.

```yaml
functions:
  - signature: CatchAll(suffix string) (ok bool)
    description: CatchAll catches all requests.
    path: /{suffix+}
    method: GET
  - signature: Hello(name string) (n int)
    description: Hello's subscription is a subset of CatchAll's.
    path: /hello/{name}
    method: GET
```

## Web Handlers

Path arguments work also for web handlers but they must be parsed manually from the request's path. Consider the following example of a web handler:

```yaml
web:
  - signature: AvatarImage()
    description: AvatarImage serves the avatar image of the user.
    path: /avatar/{uid}/{size}/{name+}
    method: GET
```

```go
func (svc *Service) AvatarImage(w http.ResponseWriter, r *http.Request) (err error) {
    // Path arguments must be manually extracted from the path
    parts := strings.Split(r.URL.Path, "/") // ["", "avatar", "{uid}", "{size}", "{name}", "..."]
    uid = parts[2]
    size = parts[3]
    name = strings.Join(parts[4:], "/")

    return serveImage(uid, size)
}
```

## Pervasive Routing

Path arguments are not recommended for pervasive (non-load balanced) endpoints, events or sinks. Using path arguments in these cases will result in significantly slow response times because they interfere with an optimization that relies on a fixed URL pattern.

```yaml
functions:
  - signature: NotAGoodIdea(id int) (ok bool)
    description: NotAGoodIdea mixes a path argument with pervasive routing.
    path: /data/{id}
    queue: none
```
