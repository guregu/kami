package kami

import (
	"net/http"
	"strings"

	_ "github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
)

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
	paths := strings.SplitAfter(r.URL.Path, "/")

	for i, _ := range paths {
		route := strings.Join(paths[0:i+1], "")
		mws, ok := middleware[route]
		if !ok {
			continue
		}
		for _, mw := range mws {
			// return nil middleware to stop
			result := mw(ctx, w, r)
			if result == nil {
				return ctx, false
			}
			ctx = result
		}
	}
	return ctx, true
}
