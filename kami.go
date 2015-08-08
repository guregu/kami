package kami

import (
	"net/http"

	"github.com/go-kami/tree"
	"github.com/julienschmidt/httprouter"
	"github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"
)

// Server holds the state for a kami server. A new struct should be used per
// server. This is not threadsafe.
type Server struct {
	BaseContext context.Context

	PanicHandler HandlerType

	LogHandler func(context.Context, mutil.WriterProxy, *http.Request)

	routes *httprouter.Router

	middleware map[string][]Middleware
	wildcardMW *tree.Node
}

// NewServer creates and initializes a new Server.
func NewServer() *Server {
	server := &Server{
		BaseContext: context.Background(),
		routes:      httprouter.New(),
		middleware:  make(map[string][]Middleware),
		wildcardMW:  new(tree.Node)}
	server.NotFound(nil)
	return server
}

// DefaultServer holds an by-default initialized Server that is used by the
// methods that do not define a Server.
var DefaultServer *Server

func init() {
	DefaultServer = NewServer()
}

// Handler returns an http.Handler serving registered routes for the server.
func (s *Server) Handler() http.Handler {
	return s.routes
}

// Handler returns an http.Handler serving registered routes.
func Handler() http.Handler {
	return DefaultServer.Handler()
}

// Handle registers an arbitrary method handler un the given path for the server.
func (s *Server) Handle(method, path string, handler HandlerType) {
	s.routes.Handle(method, path, s.bless(wrap(handler)))
}

// Handle registers an arbitrary method handler under the given path.
func Handle(method, path string, handler HandlerType) {
	DefaultServer.routes.Handle(method, path, DefaultServer.bless(wrap(handler)))
}

// Get registers a GET handler under the given path for the server.
func (s *Server) Get(path string, handler HandlerType) {
	s.Handle("GET", path, handler)
}

// Get registers a GET handler under the given path.
func Get(path string, handler HandlerType) {
	DefaultServer.Get(path, handler)
}

// Post registers a POST handler under the given path
func (s *Server) Post(path string, handler HandlerType) {
	s.Handle("POST", path, handler)
}

// Post registers a POST handler under the given path.
func Post(path string, handler HandlerType) {
	DefaultServer.Post(path, handler)
}

// Put registers a PUT handler under the given path for the server.
func (s *Server) Put(path string, handler HandlerType) {
	s.Handle("PUT", path, handler)
}

// Put registers a PUT handler under the given path.
func Put(path string, handler HandlerType) {
	DefaultServer.Put(path, handler)
}

// Patch registers a PATCH handler un the given path for the server.
func (s *Server) Patch(path string, handler HandlerType) {
	s.Handle("PATCH", path, handler)
}

// Patch registers a PATCH handler under the given path.
func Patch(path string, handler HandlerType) {
	DefaultServer.Patch(path, handler)
}

// Head registers a HEAD handler under the given path for the server.
func (s *Server) Head(path string, handler HandlerType) {
	s.Handle("HEAD", path, handler)
}

// Head registers a HEAD handler under the given path.
func Head(path string, handler HandlerType) {
	DefaultServer.Head(path, handler)
}

// Delete registers a DELETE handler under the given path for the server.
func (s *Server) Delete(path string, handler HandlerType) {
	s.Handle("DELETE", path, handler)
}

// Delete registers a DELETE handler under the given path.
func Delete(path string, handler HandlerType) {
	DefaultServer.Delete(path, handler)
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
	s.routes.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h(w, r, nil)
	})
}

// NotFound registers a special handler for unregistered (404) paths.
// If handle is nil, use the default http.NotFound behavior.
func NotFound(handler HandlerType) {
	DefaultServer.NotFound(handler)
}

// bless is the meat of kami.
// It wraps a HandleFn into an httprouter compatible request,
// in order to run all the middleware and other special handlers.
func (s *Server) bless(k ContextHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := s.BaseContext
		if len(params) > 0 {
			ctx = newContextWithParams(s.BaseContext, params)
		}
		ranLogHandler := false // track this in case the log handler blows up

		writer := w
		var proxy mutil.WriterProxy
		if s.LogHandler != nil {
			proxy = mutil.WrapWriter(w)
			writer = proxy
		}

		if s.PanicHandler != nil {
			defer func() {
				if err := recover(); err != nil {
					ctx = newContextWithException(ctx, err)
					wrap(s.PanicHandler).ServeHTTPContext(ctx, writer, r)

					if s.LogHandler != nil && !ranLogHandler {
						s.LogHandler(ctx, proxy, r)
						// should only happen if header hasn't been written
						proxy.WriteHeader(http.StatusInternalServerError)
					}
				}
			}()
		}

		ctx, ok := s.run(ctx, writer, r)
		if ok {
			k.ServeHTTPContext(ctx, writer, r)
		}

		if s.LogHandler != nil {
			ranLogHandler = true
			s.LogHandler(ctx, proxy, r)
			// should only happen if header hasn't been written
			proxy.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func (s *Server) reset() {
	s.BaseContext = context.Background()
	s.PanicHandler = nil
	s.LogHandler = nil
	s.routes = httprouter.New()
	s.middleware = make(map[string][]Middleware)
	s.wildcardMW = new(tree.Node)
	NotFound(nil)
}

// Reset changes the root Context to context.Background().
// It removes every handler and all middleware.
func Reset() {
	DefaultServer.reset()
}
