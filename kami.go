package kami

import (
	"net/http"

	"github.com/dimfeld/httptreemux"
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

var (
	routes    = newRouter()
	enable405 = true
)

func init() {
	// set up the default 404/405 handlers
	NotFound(nil)
	MethodNotAllowed(nil)
}

func newRouter() *httptreemux.TreeMux {
	r := httptreemux.New()
	r.PathSource = httptreemux.URLPath
	r.RedirectBehavior = httptreemux.Redirect307
	r.RedirectMethodBehavior = map[string]httptreemux.RedirectBehavior{
		"GET": httptreemux.Redirect301,
	}
	return r
}

// Handler returns an http.Handler serving registered routes.
func Handler() http.Handler {
	return routes
}

// Handle registers an arbitrary method handler under the given path.
func Handle(method, path string, handler HandlerType) {
	routes.Handle(method, path, defaultBless(wrap(handler)))
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

// Head registers a OPTIONS handler under the given path.
func Options(path string, handler HandlerType) {
	Handle("OPTIONS", path, handler)
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

	h := defaultBless(wrap(handler))
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

	h := defaultBless(wrap(handler))
	routes.MethodNotAllowedHandler = func(w http.ResponseWriter, r *http.Request, methods map[string]httptreemux.HandlerFunc) {
		if !enable405 {
			routes.NotFoundHandler(w, r)
			return
		}
		h(w, r, nil)
	}
}

// EnableMethodNotAllowed enables or disables automatic Method Not Allowed handling.
// Note that this is enabled by default.
func EnableMethodNotAllowed(enabled bool) {
	enable405 = enabled
}

func defaultBless(k ContextHandler) httptreemux.HandlerFunc {
	return bless(k, &Context, defaultMW, &PanicHandler, &LogHandler)
}

// bless is the meat of kami.
// It wraps a ContextHandler into an httprouter compatible request,
// in order to run all the middleware and other special handlers.
func bless(h ContextHandler, base *context.Context, mw *wares, panicHandler *HandlerType, logHandler *func(context.Context, mutil.WriterProxy, *http.Request)) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		ctx := defaultContext(*base, r)
		if len(params) > 0 {
			ctx = newContextWithParams(ctx, params)
		}
		ranLogHandler := false // track this in case the log handler blows up

		var proxy mutil.WriterProxy
		if *logHandler != nil || mw.needsWrapper() {
			proxy = mutil.WrapWriter(w)
			w = proxy
		}

		if *panicHandler != nil {
			defer func() {
				if err := recover(); err != nil {
					ctx = newContextWithException(ctx, err)
					wrap(*panicHandler).ServeHTTPContext(ctx, w, r)

					if *logHandler != nil && !ranLogHandler {
						(*logHandler)(ctx, proxy, r)
						// should only happen if header hasn't been written
						proxy.WriteHeader(http.StatusInternalServerError)
					}
				}
			}()
		}

		ctx, ok := mw.run(ctx, w, r)
		if ok {
			h.ServeHTTPContext(ctx, w, r)
		}
		if proxy != nil {
			ctx = mw.after(ctx, proxy, r)
		}

		if *logHandler != nil {
			ranLogHandler = true
			(*logHandler)(ctx, proxy, r)
			// should only happen if header hasn't been written
			proxy.WriteHeader(http.StatusInternalServerError)
		}
	}
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
