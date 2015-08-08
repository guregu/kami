package kami

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"github.com/go-kami/tree"
)

// Middleware is a function that takes the current request context and returns a new request context.
// You can use middleware to build your context before your handler handles a request.
// As a special case, middleware that returns nil will halt middleware and handler execution (LogHandler will still run).
type Middleware func(context.Context, http.ResponseWriter, *http.Request) context.Context

// MiddlewareType represents types that kami can convert to Middleware.
// kami will try its best to convert standard, non-context middleware.
// See the Use function for important information about how kami middleware is run.
// The following concrete types are accepted:
// 	- Middleware
// 	- func(context.Context, http.ResponseWriter, *http.Request) context.Context
// 	- func(http.Handler) http.Handler               [* see Use docs]
// 	- func(http.ContextHandler) http.ContextHandler [* see Use docs]
type MiddlewareType interface{}

type middlewares struct {
	hierarchy map[string][]Middleware
	wildcards *tree.Node
}

func newMiddlewares() *middlewares {
	return &middlewares{
		hierarchy: make(map[string][]Middleware),
		wildcards: new(tree.Node),
	}
}

// Use registers middleware to run for the given path.
// See the global Use function's documents for information on how middleware works.
func (m *middlewares) Use(path string, mw MiddlewareType) {
	if containsWildcard(path) {
		m.wildcards.AddRoute(path, convert(mw))
	} else {
		fn := convert(mw)
		chain := m.hierarchy[path]
		chain = append(chain, fn)
		m.hierarchy[path] = chain
	}
}

var defaultMW = newMiddlewares()

// Use registers middleware to run for the given path.
// Middleware with be executed hierarchically, starting with the least specific path.
// Middleware will be executed in order of registration.
// You may use wildcards in the path. Wildcard middleware will be run last,
// after all hierarchical middleware has run.
//
// Adding middleware is not threadsafe.
//
// WARNING: kami middleware is run in sequence, but standard middleware is chained;
// middleware that expects its code to run after the next handler, such as
// standard loggers and panic handlers, will not work as expected.
// Use kami.LogHandler and kami.PanicHandler instead.
// Standard middleware that does not call the next handler to stop the request is supported.
func Use(path string, mw MiddlewareType) {
	defaultMW.Use(path, mw)
}

// run runs the middleware chain for a particular request.
// run returns false if it should stop early.
func (m middlewares) run(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, bool) {
	// hierarchical middlewares
	for i, c := range r.URL.Path {
		if c == '/' || i == len(r.URL.Path)-1 {
			wares, ok := m.hierarchy[r.URL.Path[:i+1]]
			if !ok {
				continue
			}
			for _, mw := range wares {
				// return nil context to stop
				result := mw(ctx, w, r)
				if result == nil {
					return ctx, false
				}
				ctx = result
			}
		}
	}
	// wildcard middlewares
	if wild, params, _ := m.wildcards.GetValue(r.URL.Path); wild != nil {
		if mw, ok := wild.(Middleware); ok {
			ctx = mergeParams(ctx, params)
			result := mw(ctx, w, r)
			if result == nil {
				return ctx, false
			}
			ctx = result
		}
	}
	return ctx, true
}

// convert turns standard http middleware into kami Middleware if needed.
func convert(mw MiddlewareType) Middleware {
	switch x := mw.(type) {
	case Middleware:
		return x
	case func(context.Context, http.ResponseWriter, *http.Request) context.Context:
		return Middleware(x)
	case func(ContextHandler) ContextHandler:
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			var dh dummyHandler
			x(&dh).ServeHTTPContext(ctx, w, r)
			if !dh {
				return nil
			}
			return ctx
		}
	case func(http.Handler) http.Handler:
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			var dh dummyHandler
			x(&dh).ServeHTTP(w, r)
			if !dh {
				return nil
			}
			return ctx
		}
	}
	panic(fmt.Errorf("unsupported MiddlewareType: %T", mw))
}

// dummyHandler is used to keep track of whether the next middleware was called or not.
type dummyHandler bool

func (dh *dummyHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
	*dh = true
}

func (dh *dummyHandler) ServeHTTPContext(_ context.Context, _ http.ResponseWriter, _ *http.Request) {
	*dh = true
}

func containsWildcard(path string) bool {
	return strings.Contains(path, "/:") || strings.Contains(path, "/*")
}
