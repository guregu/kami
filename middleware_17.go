// +build go1.7

package kami

import (
	"context"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/zenazn/goji/web/mutil"
	netcontext "golang.org/x/net/context"
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
// The old x/net/context is also supported.
type MiddlewareType interface{}

// Afterware is a function that will run after middleware and the request.
// Afterware takes the request context and returns a new context, but unlike middleware,
// returning nil won't halt execution of other afterware.
type Afterware func(context.Context, mutil.WriterProxy, *http.Request) context.Context

// Afterware represents types that kami can convert to Afterware.
// The following concrete types are accepted:
//  - Afterware
//  - func(context.Context, mutil.WriterProxy, *http.Request) context.Context
//  - func(context.Context, http.ResponseWriter, *http.Request) context.Context
//  - func(context.Context, *http.Request) context.Context
//  - func(context.Context) context.Context
//  - Middleware
// The old x/net/context is also supported.
type AfterwareType interface{}

// run runs the middleware chain for a particular request.
// run returns false if it should stop early.
func (m *wares) run(ctx context.Context, w http.ResponseWriter, r *http.Request) (*http.Request, context.Context, bool) {
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
						return r, ctx, false
					}
					if result != ctx {
						r = r.WithContext(result)
					}
					ctx = result
				}
			}
		}
	}

	if m.wildcards != nil {
		// wildcard middleware
		if wild, params := m.wildcards.Get(r.URL.Path); wild != nil {
			if mw, ok := wild.(Middleware); ok {
				ctx = mergeParams(ctx, params)
				r = r.WithContext(ctx)
				result := mw(ctx, w, r)
				if result == nil {
					return r, ctx, false
				}
				if result != ctx {
					r = r.WithContext(result)
				}
				ctx = result
			}
		}
	}

	return r, ctx, true
}

// after runs the afterware chain for a particular request.
// after can't stop early
func (m *wares) after(ctx context.Context, w mutil.WriterProxy, r *http.Request) (*http.Request, context.Context) {
	if m.afterWildcards != nil {
		// wildcard afterware
		if wild, params := m.afterWildcards.Get(r.URL.Path); wild != nil {
			if aw, ok := wild.(Afterware); ok {
				ctx = mergeParams(ctx, params)
				r = r.WithContext(ctx)
				result := aw(ctx, w, r)
				if result != nil {
					if result != ctx {
						r = r.WithContext(result)
					}
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
						if result != ctx {
							r = r.WithContext(result)
						}
						ctx = result
					}
				}
			}
			path = path[:len(path)-size]
		}
	}

	return r, ctx
}

// convert turns standard http middleware into kami Middleware if needed.
func convert(mw MiddlewareType) Middleware {
	switch x := mw.(type) {
	case Middleware:
		return x
	case func(context.Context, http.ResponseWriter, *http.Request) context.Context:
		return Middleware(x)
	case func(netcontext.Context, http.ResponseWriter, *http.Request) netcontext.Context:
		return Middleware(func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			return x(ctx, w, r)
		})
	case func(ContextHandler) ContextHandler:
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			var dh dummyHandler
			x(&dh).ServeHTTPContext(ctx, w, r)
			if !dh {
				return nil
			}
			return ctx
		}
	case func(OldContextHandler) OldContextHandler:
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			var dh oldDummyHandler
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
	case func(netcontext.Context, mutil.WriterProxy, *http.Request) netcontext.Context:
		return func(ctx context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
			return x(ctx, w, r)
		}
	case func(context.Context, *http.Request) context.Context:
		return func(ctx context.Context, _ mutil.WriterProxy, r *http.Request) context.Context {
			return x(ctx, r)
		}
	case func(netcontext.Context, *http.Request) netcontext.Context:
		return func(ctx context.Context, _ mutil.WriterProxy, r *http.Request) context.Context {
			return x(ctx, r)
		}
	case func(context.Context) context.Context:
		return func(ctx context.Context, _ mutil.WriterProxy, _ *http.Request) context.Context {
			return x(ctx)
		}
	case func(netcontext.Context) netcontext.Context:
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
	case func(netcontext.Context, http.ResponseWriter, *http.Request) netcontext.Context:
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

// oldDummyHandler is dummyHandler compatible with the old context type.
type oldDummyHandler bool

func (dh *oldDummyHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
	*dh = true
}

func (dh *oldDummyHandler) ServeHTTPContext(_ netcontext.Context, _ http.ResponseWriter, _ *http.Request) {
	*dh = true
}
