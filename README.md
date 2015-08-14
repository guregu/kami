## kami [![GoDoc](https://godoc.org/github.com/guregu/kami?status.svg)](https://godoc.org/github.com/guregu/kami) [![Coverage](http://gocover.io/_badge/github.com/guregu/kami?0)](http://gocover.io/github.com/guregu/kami)
`import "github.com/guregu/kami"`

kami (神) is a tiny web framework using [x/net/context](https://blog.golang.org/context) for request context and [HttpRouter](https://github.com/julienschmidt/httprouter) for routing. It includes a simple system for running hierarchical middleware before requests, in addition to log and panic hooks. Graceful restart via einhorn is also supported.

kami is designed to be used as central registration point for your routes, middleware, and context "god object", so kami has no concept of multiple muxes. 

You are free to mount `kami.Handler()` wherever you wish, but a helpful `kami.Serve()` function is provided.

Here is a [presentation about the birth of kami](http://go-talks.appspot.com/github.com/guregu/slides/kami/kami.slide), explaining some of the design choices. 

### Example

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/guregu/kami"
	"golang.org/x/net/context"
)

func hello(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", kami.Param(ctx, "name"))
}

func main() {
	kami.Get("/hello/:name", hello)
	kami.Serve()
}
```

### Usage

* Set up routes using `kami.Get("path", handler)`, `kami.Post(...)`, etc. You can use [named parameters](https://github.com/julienschmidt/httprouter#named-parameters) or [wildcards](https://github.com/julienschmidt/httprouter#catch-all-parameters) in URLs like `/hello/:name/edit` or `/files/*path`, and access them using the context kami gives you: `kami.Param(ctx, "name")`. The following kinds of handlers are accepted:
  * types that implement `kami.ContextHandler`
  * `func(context.Context, http.ResponseWriter, *http.Request)`
  * types that implement `http.Handler`
  * `func(http.ResponseWriter, *http.Request)`
* All contexts that kami uses are descended from `kami.Context`: this is the "god object" and the namesake of this project. By default, this is `context.Background()`, but feel free to replace it with a pre-initialized context suitable for your application.
* Add middleware with `kami.Use("path", kami.Middleware)`. More on middleware below.
* You can provide a panic handler by setting `kami.PanicHandler`. When the panic handler is called, you can access the panic error with `kami.Exception(ctx)`. 
* You can also provide a `kami.LogHandler` that will wrap every request. `kami.LogHandler` has a different function signature, taking a WriterProxy that has access to the response status code, etc.
* Use `kami.Serve()` to gracefully serve your application, or mount `kami.Handler()` somewhere convenient. 

### Middleware
```go
type Middleware func(context.Context, http.ResponseWriter, *http.Request) context.Context
```
Middleware differs from a HandlerType in that it returns a new context. You can take advantage of this to build your context by registering middleware at the approriate paths. As a special case, you may return **nil** to halt execution of the middleware chain.

Middleware is hierarchical. For example, a request for `/hello/greg` will run middleware registered under the following paths, in order:

1. `/`
2. `/hello/`
3. `/hello/greg`

Within a path, middleware is run in the order of registration.

```go
func init() {
	kami.Use("/", Login)
	kami.Use("/private/", LoginRequired)
}

// Login returns a new context with the appropiate user object inside
func Login(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	if u, err := user.GetByToken(ctx, r.FormValue("auth_token")); err == nil {
		ctx = user.NewContext(ctx, u)
	}
	return ctx
}

// LoginRequired stops the request if we don't have a user object
func LoginRequired(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	if _, ok := user.FromContext(ctx); !ok {
		return nil
	}
	return ctx
}	
```

#### Named parameters, wildcards, and middleware

Named parameters and wildcards in middleware are supported now. Middleware registered under a path with a wildcard will run **after** all hierarchical middleware. 

```go
kami.Use("/user/:id/edit", CheckAdminPermissions)
```

#### Vanilla net/http middleware

kami can use vanilla http middleware as well. `kami.Use` accepts functions in the form of `func(next http.Handler) http.Handler`. Be advised that kami will run such middleware in sequence, not in a chain. This means that standard loggers and panic handlers won't work as you expect. You should use `kami.LogHandler` and `kami.PanicHandler` instead.

The following example uses [goji/httpauth](https://github.com/goji/httpauth) to add HTTP Basic Authentication to paths under `/secret/`.

```go
import (
	"github.com/goji/httpauth"
	"github.com/guregu/kami"
)

func main() {
	kami.Use("/secret/", httpauth.SimpleBasicAuth("username", "password"))
	kami.Get("/secret/message", secretMessageHandler)
	kami.Serve()
}
```

### License

MIT

### Acknowledgements

* [HttpRouter](https://github.com/julienschmidt/httprouter): router
* [Goji](https://github.com/zenazn/goji): graceful, WriterProxy