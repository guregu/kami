package kami

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"
)

// Mux is an independent kami router and middleware stack. Manipulating it is not threadsafe.
type Mux struct {
	// Context is the root "god object" for this mux,
	// from which every request's context will derive.
	Context context.Context
	// PanicHandler will, if set, be called on panics.
	// You can use kami.Exception(ctx) within the panic handler to get panic details.
	PanicHandler HandlerType
	// LogHandler will, if set, wrap every request and be called at the very end.
	LogHandler func(context.Context, mutil.WriterProxy, *http.Request)
	// CloseHandler will, if set, called before the log handler and can be used for cleanup actions
	CloseHandler func(context.Context, *http.Request)

	routes *httprouter.Router
	*middlewares
}

// New creates a new independent kami router and middleware stack.
// It is totally separate from the global kami.Context and middleware stack.
func New() *Mux {
	m := &Mux{
		Context:     context.Background(),
		routes:      httprouter.New(),
		middlewares: newMiddlewares(),
	}
	m.NotFound(nil)
	m.MethodNotAllowed(nil)
	return m
}

// ServeHTTP handles an HTTP request, running middleware and forwarding the request to the appropriate handler.
// Implements the http.Handler interface for easy composition with other frameworks.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.routes.ServeHTTP(w, r)
}

// Handle registers an arbitrary method handler under the given path.
func (m *Mux) Handle(method, path string, handler HandlerType) {
	m.routes.Handle(method, path, m.bless(wrap(handler)))
}

// Get registers a GET handler under the given path.
func (m *Mux) Get(path string, handler HandlerType) {
	m.Handle("GET", path, handler)
}

// Post registers a POST handler under the given path.
func (m *Mux) Post(path string, handler HandlerType) {
	m.Handle("POST", path, handler)
}

// Put registers a PUT handler under the given path.
func (m *Mux) Put(path string, handler HandlerType) {
	m.Handle("PUT", path, handler)
}

// Patch registers a PATCH handler under the given path.
func (m *Mux) Patch(path string, handler HandlerType) {
	m.Handle("PATCH", path, handler)
}

// Head registers a HEAD handler under the given path.
func (m *Mux) Head(path string, handler HandlerType) {
	m.Handle("HEAD", path, handler)
}

// Head registers a OPTIONS handler under the given path.
func (m *Mux) Options(path string, handler HandlerType) {
	m.Handle("OPTIONS", path, handler)
}

// Delete registers a DELETE handler under the given path.
func (m *Mux) Delete(path string, handler HandlerType) {
	m.Handle("DELETE", path, handler)
}

// NotFound registers a special handler for unregistered (404) paths.
// If handle is nil, use the default http.NotFound behavior.
func (m *Mux) NotFound(handler HandlerType) {
	// set up the default handler if needed
	// we need to bless this so middleware will still run for a 404 request
	if handler == nil {
		handler = HandlerFunc(func(_ context.Context, w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}

	h := m.bless(wrap(handler))
	m.routes.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h(w, r, nil)
	})
}

// MethodNotAllowed registers a special handler for automatically responding
// to invalid method requests (405).
func (m *Mux) MethodNotAllowed(handler HandlerType) {
	if handler == nil {
		handler = HandlerFunc(func(_ context.Context, w http.ResponseWriter, r *http.Request) {
			http.Error(w,
				http.StatusText(http.StatusMethodNotAllowed),
				http.StatusMethodNotAllowed,
			)
		})
	}

	h := m.bless(wrap(handler))
	m.routes.MethodNotAllowed = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h(w, r, nil)
	})
}

// EnableMethodNotAllowed enables or disables automatic Method Not Allowed handling.
// Note that this is enabled by default.
func (m *Mux) EnableMethodNotAllowed(enabled bool) {
	m.routes.HandleMethodNotAllowed = enabled
}

func (m *Mux) bless(k ContextHandler) httprouter.Handle {
	return bless(k, &m.Context, m.middlewares, &m.PanicHandler, &m.LogHandler, &m.CloseHandler)
}
