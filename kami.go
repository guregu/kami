package kami

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"
)

var (
	// Context is the root "god object" from which every request's context will derive
	Context = context.Background()

	// PanicHandler will, if set, be called on panics.
	// You can use kami.Exception(ctx) within the panic handler to get panic details.
	PanicHandler HandlerType
	// LogHandler will, if set, wrap every request and be called at the very end.
	LogHandler func(context.Context, mutil.WriterProxy, *http.Request)
)

var routes = httprouter.New()

func init() {
	// set up the default 404 handler
	NotFound(nil)
}

// Handler returns an http.Handler serving registered routes.
func Handler() http.Handler {
	return routes
}

// Handle registers an arbitrary method handler under the given path.
func Handle(method, path string, handler HandlerType) {
	routes.Handle(method, path, bless(wrap(handler)))
}

// Get registers a GET handler under the given path.
func Get(path string, handler HandlerType) {
	Handle("GET", path, handler)
}

// Post registers a POST handler under the given path.
func Post(path string, handler HandlerType) {
	Handle("POST", path, handler)
}

// Put registers a PUT handler under the given path.
func Put(path string, handler HandlerType) {
	Handle("PUT", path, handler)
}

// Patch registers a PATCH handler under the given path.
func Patch(path string, handler HandlerType) {
	Handle("PATCH", path, handler)
}

// Head registers a HEAD handler under the given path.
func Head(path string, handler HandlerType) {
	Handle("HEAD", path, handler)
}

// Delete registers a DELETE handler under the given path.
func Delete(path string, handler HandlerType) {
	Handle("DELETE", path, handler)
}

// NotFound registers a special handler for unregistered (404) paths.
// If handle is nil, use the default http.NotFound behavior.
func NotFound(handler HandlerType) {
	// set up the default handler if needed
	// we need to bless this so middleware will still run for a 404 request
	if handler == nil {
		handler = HandlerFunc(func(_ context.Context, w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}

	h := bless(wrap(handler))
	routes.NotFound = func(w http.ResponseWriter, r *http.Request) {
		h(w, r, nil)
	}
}

// bless is the meat of kami.
// It wraps a HandleFn into an httprouter compatible request,
// in order to run all the middleware and other special handlers.
func bless(k ContextHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := Context
		if len(params) > 0 {
			ctx = newContextWithParams(Context, params)
		}
		ranLogHandler := false // track this in case the log handler blows up

		writer := w
		var proxy mutil.WriterProxy
		if LogHandler != nil {
			proxy = mutil.WrapWriter(w)
			writer = proxy
		}

		if PanicHandler != nil {
			defer func() {
				if err := recover(); err != nil {
					ctx = newContextWithException(ctx, err)
					wrap(PanicHandler).ServeHTTPContext(ctx, writer, r)

					if LogHandler != nil && !ranLogHandler {
						LogHandler(ctx, proxy, r)
						// should only happen if header hasn't been written
						proxy.WriteHeader(500)
					}
				}
			}()
		}

		ctx, ok := run(ctx, writer, r)
		if ok {
			k.ServeHTTPContext(ctx, writer, r)
		}

		if LogHandler != nil {
			ranLogHandler = true
			LogHandler(ctx, proxy, r)
			// should only happen if header hasn't been written
			proxy.WriteHeader(500)
		}
	}
}

// Reset changes the root Context to context.Background().
// It removes every handler and all middleware.
func Reset() {
	Context = context.Background()
	PanicHandler = nil
	LogHandler = nil
	middleware = make(map[string][]Middleware)
	routes = httprouter.New()
	NotFound(nil)
}
