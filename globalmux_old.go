// +build !go1.7

package kami

import (
	"net/http"

	"github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"
)

var (
	// Context is the root "god object" from which every request's context will derive.
	Context = context.Background()

	// PanicHandler will, if set, be called on panics.
	// You can use kami.Exception(ctx) within the panic handler to get panic details.
	PanicHandler HandlerType
	// LogHandler will, if set, wrap every request and be called at the very end.
	LogHandler func(context.Context, mutil.WriterProxy, *http.Request)
)

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
	routes.NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		h(w, r, nil)
	}
}

// MethodNotAllowed registers a special handler for automatically responding
// to invalid method requests (405).
func MethodNotAllowed(handler HandlerType) {
	if handler == nil {
		handler = HandlerFunc(func(_ context.Context, w http.ResponseWriter, r *http.Request) {
			http.Error(w,
				http.StatusText(http.StatusMethodNotAllowed),
				http.StatusMethodNotAllowed,
			)
		})
	}

	h := bless(wrap(handler))
	routes.MethodNotAllowedHandler = func(w http.ResponseWriter, r *http.Request, methods map[string]httptreemux.HandlerFunc) {
		if !enable405 {
			routes.NotFoundHandler(w, r)
			return
		}
		h(w, r, nil)
	}
}

// bless creates a new kamified handler using the global mux and middleware.
func bless(h ContextHandler) httptreemux.HandlerFunc {
	k := kami{
		handler:      h,
		base:         &Context,
		middleware:   defaultMW,
		panicHandler: &PanicHandler,
		logHandler:   &LogHandler,
	}
	return k.handle
}

// Reset changes the root Context to context.Background().
// It removes every handler and all middleware.
func Reset() {
	Context = context.Background()
	PanicHandler = nil
	LogHandler = nil
	defaultMW = newWares()
	routes = newRouter()
	NotFound(nil)
	MethodNotAllowed(nil)
}
