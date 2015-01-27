package kami

import (
	"net/http"
	"strings"

	_ "github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
)

type Middleware func(context.Context, http.ResponseWriter, *http.Request) context.Context

var middleware = make(map[string][]Middleware)

func Use(path string, fn Middleware) {
	chain := middleware[path]
	chain = append(chain, fn)
	middleware[path] = chain
}

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
