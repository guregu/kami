package kami

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"
)

// HandleFn is a kami-compatible handler function.
type HandleFn func(context.Context, http.ResponseWriter, *http.Request)

var (
	Context      = context.Background()                                  // the root "god object" of every request
	PanicHandler HandleFn                                                // if exists, will be called on panics
	LogHandler   func(context.Context, mutil.WriterProxy, *http.Request) // if exists, will run after each request
)

var routes = httprouter.New()

// Handler returns an http.Handler serving registered routes.
func Handler() http.Handler {
	return routes
}

// Handle registers an arbitrary method handler under the given path.
func Handle(method, path string, handle HandleFn) {
	routes.Handle(method, path, wrap(handle))
}

// Get registers a GET handler under the given path.
func Get(path string, handle HandleFn) {
	Handle("GET", path, handle)
}

// Post registers a POST handler under the given path.
func Post(path string, handle HandleFn) {
	Handle("POST", path, handle)
}

// Put registers a PUT handler under the given path.
func Put(path string, handle HandleFn) {
	Handle("PUT", path, handle)
}

// Patch registers a PATCH handler under the given path.
func Patch(path string, handle HandleFn) {
	Handle("PATCH", path, handle)
}

// Head registers a HEAD handler under the given path.
func Head(path string, handle HandleFn) {
	Handle("HEAD", path, handle)
}

// wrap is the meat of kami.
// It wraps a httprouter compatible request to run all the middleware, etc.
func wrap(k HandleFn) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := newContextWithParams(Context, params)
		ranLogHandler := false

		writer := w
		var wrapped mutil.WriterProxy
		if LogHandler != nil {
			wrapped = mutil.WrapWriter(w)
			writer = wrapped
		}

		if PanicHandler != nil {
			defer func() {
				if err := recover(); err != nil {
					ctx = newContextWithException(ctx, err)
					PanicHandler(ctx, writer, r)

					if LogHandler != nil && !ranLogHandler {
						LogHandler(ctx, wrapped, r)
						// should only happen if header hasn't been written
						wrapped.WriteHeader(500)
					}
				}
			}()
		}

		ctx, ok := run(ctx, writer, r)
		if ok {
			k(ctx, writer, r)
		}

		if LogHandler != nil {
			ranLogHandler = true
			LogHandler(ctx, wrapped, r)
			// should only happen if header hasn't been written
			wrapped.WriteHeader(500)
		}
	}
}
