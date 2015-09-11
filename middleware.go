package kami

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/context"

	"github.com/go-kami/tree"
	"github.com/zenazn/goji/web/mutil"
)

// Middleware is a function that takes the current request context and returns a new request context.
// You can use middleware to build your context before your handler handles a request.
// As a special case, middleware that returns nil will halt middleware and handler execution (LogHandler will still run).
type Middleware func(context.Context, http.ResponseWriter, *http.Request) context.Context

// MiddlewareType represents types that kami can convert to Middleware.
// kami will try its best to convert standard, non-context middleware.
// See the Use function for important information about how kami middleware is run.
// The following concrete types are accepted:
// 	- Middleware
// 	- func(context.Context, http.ResponseWriter, *http.Request) context.Context
// 	- func(http.Handler) http.Handler               [* see Use docs]
// 	- func(http.ContextHandler) http.ContextHandler [* see Use docs]
type MiddlewareType interface{}

// Afterware is a function that will run after middleware and the request.
// Afterware takes the request context and returns a new context, but unlike middleware,
// returning nil won't halt execution of other afterware.
type Afterware func(context.Context, mutil.WriterProxy, *http.Request) context.Context

// Afterware represents types that kami can convert to Afterware.
// The following concrete types are accepted:
//  - Afterware
//  - func(context.Context, mutil.WriterProxy, *http.Request) context.Context
// 	- func(context.Context, http.ResponseWriter, *http.Request) context.Context
//  - func(context.Context, *http.Request) context.Context
//  - func(context.Context) context.Context
// 	- Middleware
type AfterwareType interface{}

type wares struct {
	middleware     map[string][]Middleware
	afterware      map[string][]Afterware
	wildcards      *tree.Node
	afterWildcards *tree.Node
}

func newWares() *wares {
	return new(wares)
}

// Use registers middleware to run for the given path.
// See the global Use function's documents for information on how middleware works.
func (m *wares) Use(path string, mw MiddlewareType) {
	if containsWildcard(path) {
		if m.wildcards == nil {
			m.wildcards = new(tree.Node)
		}
		m.wildcards.AddRoute(path, convert(mw))
	} else {
		if m.middleware == nil {
			m.middleware = make(map[string][]Middleware)
		}
		fn := convert(mw)
		chain := m.middleware[path]
		chain = append(chain, fn)
		m.middleware[path] = chain
	}
}

// After registers middleware to run for the given path after normal middleware added with Use has run.
// See the global After function's documents for information on how middleware works.
func (m *wares) After(path string, afterware AfterwareType) {
	aw := convertAW(afterware)
	if containsWildcard(path) {
		if m.afterWildcards == nil {
			m.afterWildcards = new(tree.Node)
		}
		m.afterWildcards.AddRoute(path, aw)
	} else {
		if m.afterware == nil {
			m.afterware = make(map[string][]Afterware)
		}
		m.afterware[path] = append([]Afterware{aw}, m.afterware[path]...)
	}
}

var defaultMW = newWares() // for the global router

// Use registers middleware to run for the given path.
// Middleware will be executed hierarchically, starting with the least specific path.
// Middleware under the same path will be executed in order of registration.
// You may use wildcards in the path. Wildcard middleware will be run last,
// after all hierarchical middleware has run.
//
// Adding middleware is not threadsafe.
//
// WARNING: kami middleware is run in sequence, but standard middleware is chained;
// middleware that expects its code to run after the next handler, such as
// standard loggers and panic handlers, will not work as expected.
// Use kami.LogHandler and kami.PanicHandler instead.
// Standard middleware that does not call the next handler to stop the request is supported.
func Use(path string, mw MiddlewareType) {
	defaultMW.Use(path, mw)
}

// After registers afterware to run after middleware and the request handler has run.
// Afterware is like middleware, but everything is in reverse.
// Afterware will be executed hierarchically, starting with wildcards and then
// the most specific path, ending with /.
// Afterware under the same path will be executed in the opposite order of registration.
func After(path string, aw AfterwareType) {
	defaultMW.After(path, aw)
}

// run runs the middleware chain for a particular request.
// run returns false if it should stop early.
func (m *wares) run(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, bool) {
	if m.middleware != nil {
		// hierarchical middleware
		for i, c := range r.URL.Path {
			if c == '/' || i == len(r.URL.Path)-1 {
				mws, ok := m.middleware[r.URL.Path[:i+1]]
				if !ok {
					continue
				}
				for _, mw := range mws {
					// return nil context to stop
					result := mw(ctx, w, r)
					if result == nil {
						return ctx, false
					}
					ctx = result
				}
			}
		}
	}

	if m.wildcards != nil {
		// wildcard middleware
		if wild, params, _ := m.wildcards.GetValue(r.URL.Path); wild != nil {
			if mw, ok := wild.(Middleware); ok {
				ctx = mergeParams(ctx, params)
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

// after runs the afterware chain for a particular request.
// after can't stop early
func (m *wares) after(ctx context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
	if m.afterWildcards != nil {
		// wildcard afterware
		if wild, params, _ := m.afterWildcards.GetValue(r.URL.Path); wild != nil {
			if aw, ok := wild.(Afterware); ok {
				ctx = mergeParams(ctx, params)
				result := aw(ctx, w, r)
				if result != nil {
					ctx = result
				}
			}
		}
	}

	if m.afterware != nil {
		// hierarchical afterware, like middleware in reverse
		path := r.URL.Path
		for len(path) > 0 {
			chr, size := utf8.DecodeLastRuneInString(path)
			if chr == '/' || len(path) == len(r.URL.Path) {
				for _, aw := range m.afterware[path] {
					result := aw(ctx, w, r)
					if result != nil {
						ctx = result
					}
				}
			}
			path = path[:len(path)-size]
		}
	}

	return ctx
}

func (m *wares) needsWrapper() bool {
	return m.afterware != nil || m.afterWildcards != nil
}

// convert turns standard http middleware into kami Middleware if needed.
func convert(mw MiddlewareType) Middleware {
	switch x := mw.(type) {
	case Middleware:
		return x
	case func(context.Context, http.ResponseWriter, *http.Request) context.Context:
		return Middleware(x)
	case func(ContextHandler) ContextHandler:
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			var dh dummyHandler
			x(&dh).ServeHTTPContext(ctx, w, r)
			if !dh {
				return nil
			}
			return ctx
		}
	case func(http.Handler) http.Handler:
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			var dh dummyHandler
			x(&dh).ServeHTTP(w, r)
			if !dh {
				return nil
			}
			return ctx
		}
	}
	panic(fmt.Errorf("unsupported MiddlewareType: %T", mw))
}

// convertAW
func convertAW(aw AfterwareType) Afterware {
	switch x := aw.(type) {
	case Afterware:
		return x
	case func(context.Context, mutil.WriterProxy, *http.Request) context.Context:
		return Afterware(x)
	case func(context.Context, *http.Request) context.Context:
		return func(ctx context.Context, _ mutil.WriterProxy, r *http.Request) context.Context {
			return x(ctx, r)
		}
	case func(context.Context) context.Context:
		return func(ctx context.Context, _ mutil.WriterProxy, _ *http.Request) context.Context {
			return x(ctx)
		}
	case Middleware:
		return func(ctx context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
			return x(ctx, w, r)
		}
	case func(context.Context, http.ResponseWriter, *http.Request) context.Context:
		return func(ctx context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
			return x(ctx, w, r)
		}
	}
	panic(fmt.Errorf("unsupported AfterwareType: %T", aw))
}

// dummyHandler is used to keep track of whether the next middleware was called or not.
type dummyHandler bool

func (dh *dummyHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
	*dh = true
}

func (dh *dummyHandler) ServeHTTPContext(_ context.Context, _ http.ResponseWriter, _ *http.Request) {
	*dh = true
}

func containsWildcard(path string) bool {
	return strings.Contains(path, "/:") || strings.Contains(path, "/*")
}
