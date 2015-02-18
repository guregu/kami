package kami

import (
	"net/http"

	"golang.org/x/net/context"
)

// Middleware is a function that takes the current request context and returns a new request context.
// You can use middleware to build your context before your handler handles a request.
// As a special case, middleware that returns nil will halt middleware and handler execution (LogHandler will still run).
type Middleware func(context.Context, http.ResponseWriter, *http.Request) context.Context

var middleware = make(map[string][]Middleware)

// Use registers middleware to run for the given path.
// Middleware with be executed hierarchically, starting with the least specific path.
// Middleware will be executed in order of registration.
// Adding middleware is not threadsafe.
func Use(path string, fn Middleware) {
	chain := middleware[path]
	chain = append(chain, fn)
	middleware[path] = chain
}

// run runs the middleware chain for a particular request.
// run returns false if it should stop early.
func run(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, bool) {
	for i, c := range r.URL.Path {
		if c == '/' || i == len(r.URL.Path)-1 {
			wares, ok := middleware[r.URL.Path[:i+1]]
			if !ok {
				continue
			}
			for _, mw := range wares {
				// return nil middleware to stop
				result := mw(ctx, w, r)
				if result == nil {
					return ctx, false
				}
				ctx = result
			}
		}
	}
	return ctx, true
}
