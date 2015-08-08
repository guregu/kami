package kami

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"
)

// Server is an independent kami router and middleware stack. Manipulating it is not threadsafe.
type Server struct {
	// Context is the root "god object" for this server,
	// from which every request's context will derive.
	Context context.Context
	// PanicHandler will, if set, be called on panics.
	// You can use kami.Exception(ctx) within the panic handler to get panic details.
	PanicHandler HandlerType
	// LogHandler will, if set, wrap every request and be called at the very end.
	LogHandler func(context.Context, mutil.WriterProxy, *http.Request)

	routes *httprouter.Router
	*middlewares
}

// NewServer creates a new independent kami router and middleware stack.
// It is totally separate from the global kami.Context.
func NewServer() *Server {
	return &Server{
		routes:      httprouter.New(),
		middlewares: newMiddlewares(),
	}
}

// Handler returns an http.Handler serving this server's routes.
func (s *Server) Handler() http.Handler {
	return s.routes
}

// Handle registers an arbitrary method handler under the given path.
func (s *Server) Handle(method, path string, handler HandlerType) {
	s.routes.Handle(method, path, s.bless(wrap(handler)))
}

// Get registers a GET handler under the given path.
func (s *Server) Get(path string, handler HandlerType) {
	s.Handle("GET", path, handler)
}

// Post registers a POST handler under the given path.
func (s *Server) Post(path string, handler HandlerType) {
	s.Handle("POST", path, handler)
}

// Put registers a PUT handler under the given path.
func (s *Server) Put(path string, handler HandlerType) {
	s.Handle("PUT", path, handler)
}

// Patch registers a PATCH handler under the given path.
func (s *Server) Patch(path string, handler HandlerType) {
	s.Handle("PATCH", path, handler)
}

// Head registers a HEAD handler under the given path.
func (s *Server) Head(path string, handler HandlerType) {
	s.Handle("HEAD", path, handler)
}

// Delete registers a DELETE handler under the given path.
func (s *Server) Delete(path string, handler HandlerType) {
	s.Handle("DELETE", path, handler)
}

// NotFound registers a special handler for unregistered (404) paths.
// If handle is nil, use the default http.NotFound behavior.
func (s *Server) NotFound(handler HandlerType) {
	// set up the default handler if needed
	// we need to bless this so middleware will still run for a 404 request
	if handler == nil {
		handler = HandlerFunc(func(_ context.Context, w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}

	h := s.bless(wrap(handler))
	routes.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h(w, r, nil)
	})
}

func (s *Server) bless(k ContextHandler) httprouter.Handle {
	return bless(k, &s.Context, s.middlewares, &s.PanicHandler, &s.LogHandler)
}
