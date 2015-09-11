## kami [![GoDoc](https://godoc.org/github.com/guregu/kami?status.svg)](https://godoc.org/github.com/guregu/kami) [![Coverage](http://gocover.io/_badge/github.com/guregu/kami?0)](http://gocover.io/github.com/guregu/kami)
`import "github.com/guregu/kami"` [or](http://gopkg.in) `import "gopkg.in/guregu/kami.v1"`

kami (神) is a tiny web framework using [x/net/context](https://blog.golang.org/context) for request context and [HttpRouter](https://github.com/julienschmidt/httprouter) for routing. It includes a simple system for running hierarchical middleware before and after requests, in addition to log and panic hooks. Graceful restart via einhorn is also supported.

kami is designed to be used as central registration point for your routes, middleware, and context "god object". You are encouraged to use the global functions, but kami supports multiple muxes with `kami.New()`. 

You are free to mount `kami.Handler()` wherever you wish, but a helpful `kami.Serve()` function is provided.

Here is a [presentation about the birth of kami](http://go-talks.appspot.com/github.com/guregu/slides/kami/kami.slide), explaining some of the design choices. 

### Example

[Skip :fast_forward:](#usage)

A contrived example using kami and x/net/context to localize greetings.

```go
// Our webserver
package main

import (
	"fmt"
	"net/http"

	"github.com/guregu/kami"
	"golang.org/x/net/context"

	"github.com/my-github/greeting" // see package greeting below
)

func greet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	hello := greeting.FromContext(ctx)
	name := kami.Param(ctx, "name")
	fmt.Fprintf(w, "%s, %s!", hello, name)
}

func main() {
	ctx := context.Background()
	ctx = greeting.WithContext(ctx, "Hello") // set default greeting
	kami.Context = ctx                       // set our "god context", the base context for all requests

	kami.Use("/hello/", greeting.Guess) // use this middleware for paths under /hello/
	kami.Get("/hello/:name", greet)     // add a GET handler with a parameter in the URL
	kami.Serve()                        // gracefully serve with support for einhorn and systemd
}

}
```

```go
// Package greeting stores greeting settings in context.
package greeting

import (
	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/text/language"
)

// For more information about context and why we're doing this,
// see https://blog.golang.org/context
type ctxkey int

var key ctxkey = 0

var greetings = map[language.Tag]string{
	language.AmericanEnglish: "Yo",
	language.Japanese:        "こんにちは",
}

// Guess is kami middleware that examines Accept-Language and sets
// the greeting to a better one if possible.
func Guess(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	if tag, _, err := language.ParseAcceptLanguage(r.Header.Get("Accept-Language")); err == nil {
		for _, t := range tag {
			if g, ok := greetings[t]; ok {
				ctx = WithContext(ctx, g)
				return ctx
			}
		}
	}
	return ctx
}

// WithContext returns a new context with the given greeting.
func WithContext(ctx context.Context, greeting string) context.Context {
	return context.WithValue(ctx, key, greeting)
}

// FromContext retrieves the greeting from this context,
// or returns an empty string if missing.
func FromContext(ctx context.Context) string {
	hello, _ := ctx.Value(key).(string)
	return hello
}
```

### Usage

* Set up routes using `kami.Get("/path", handler)`, `kami.Post(...)`, etc. You can use [named parameters](https://github.com/julienschmidt/httprouter#named-parameters) or [wildcards](https://github.com/julienschmidt/httprouter#catch-all-parameters) in URLs like `/hello/:name/edit` or `/files/*path`, and access them using the context kami gives you: `kami.Param(ctx, "name")`. The following kinds of handlers are accepted:
  * types that implement `kami.ContextHandler`
  * `func(context.Context, http.ResponseWriter, *http.Request)`
  * types that implement `http.Handler`
  * `func(http.ResponseWriter, *http.Request)`
* All contexts that kami uses are descended from `kami.Context`: this is the "god object" and the namesake of this project. By default, this is `context.Background()`, but feel free to replace it with a pre-initialized context suitable for your application.
* Builds targeting Google App Engine will automatically wrap the "god object" Context with App Engine's per-request Context.
* Add middleware with `kami.Use("/path", kami.Middleware)`. Middleware runs before requests and can stop them early. More on middleware below.
* Add afterware with `kami.After("/path", kami.Afterware)`. Afterware runs after requests.
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
		w.WriteHeader(http.StatusForbidden)
		// ... render 503 Forbidden page
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

#### Afterware

```go
type Afterware func(context.Context, mutil.WriterProxy, *http.Request) context.Context
```

```go
func init() {
	kami.After("/", cleanup)
}
```

Running after the request handler, afterware is useful for cleaning up. Afterware is like a mirror image of middleware. Afterware also runs hierarchically, but in the reverse order of middleware. Wildcards are evaluated **before** hierarchical afterware.

For example, a request for `/hello/greg` will run afterware registered under the following paths:

1. `/hello/greg`
2. `/hello/`
3. `/`

This gives afterware under specific paths the ability to use resources that may be closed by `/`. 

Unlike middleware, afterware returning **nil** will not stop the remaining afterware from being evaluated. 

`kami.After("/path", afterware)` supports many different types of functions, see the docs for `kami.AfterwareType` for more details. 

### Independent stacks with `*kami.Mux`

kami was originally designed to be the "glue" between multiple packages in a complex web application. The global functions and `kami.Context` are an easy way for your packages to work together. However, if you would like to use kami as an embedded server within another app, serve two separate kami stacks on different ports, or otherwise would like to have an non-global version of kami, `kami.New()` may come in handy.

Calling `kami.New()` returns a fresh `*kami.Mux`, a completely independent kami stack. Changes to `kami.Context`, paths registered with `kami.Get()` et al, and global middleware registered with `kami.Use()` will not affect a `*kami.Mux`. 

Instead, with `mux := kami.New()` you can change `mux.Context`, call `mux.Use()`, `mux.Get()`, `mux.NotFound()`, etc. 

`*kami.Mux` implements `http.Handler`, so you may use it however you'd like!

```go
// package admin is an admin panel web server plugin
package admin

import (
	"net/http"
	"github.com/guregu/kami"
)

// automatically mount our secret admin stuff
func init() {
	mux := kami.New()
	mux.Context = adminContext
	mux.Use("/", authorize)
	mux.Get("/admin/memstats", memoryStats)
	mux.Post("/admin/die", shutdown)
	//  ...
	http.Handle("/admin/", mux)
}
```

### License

MIT

### Acknowledgements

* [HttpRouter](https://github.com/julienschmidt/httprouter): router
* [Goji](https://github.com/zenazn/goji): graceful, WriterProxy