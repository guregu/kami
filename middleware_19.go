// +build go1.9

package kami

import (
	"context"
	"fmt"
	"net/http"

	"github.com/zenazn/goji/web/mutil"
)

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
	case http.Handler:
		return Middleware(func(_ context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			x.ServeHTTP(w, r)
			return r.Context()
		})
	case func(w http.ResponseWriter, r *http.Request):
		return Middleware(func(_ context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			x(w, r)
			return r.Context()
		})
	case func(w http.ResponseWriter, r *http.Request) context.Context:
		return Middleware(func(_ context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			return x(w, r)
		})
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
	case func(w http.ResponseWriter, r *http.Request) context.Context:
		return Afterware(func(_ context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
			return x(w, r)
		})
	case func(w mutil.WriterProxy, r *http.Request) context.Context:
		return Afterware(func(_ context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
			return x(w, r)
		})
	case http.Handler:
		return Afterware(func(_ context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
			x.ServeHTTP(w, r)
			return r.Context()
		})
	case func(w http.ResponseWriter, r *http.Request):
		return Afterware(func(_ context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
			x(w, r)
			return r.Context()
		})
	case func(w mutil.WriterProxy, r *http.Request):
		return Afterware(func(_ context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
			x(w, r)
			return r.Context()
		})
	}
	panic(fmt.Errorf("unsupported AfterwareType: %T", aw))
}
