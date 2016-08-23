// +build go1.7

package kami

import (
	"context"
	"fmt"
	"net/http"

	netcontext "golang.org/x/net/context"
)

// HandlerType is the type of Handlers and types that kami internally converts to
// ContextHandler. In order to provide an expressive API, this type is an alias for
// interface{} that is named for the purposes of documentation, however only the
// following concrete types are accepted:
// 	- types that implement http.Handler
// 	- types that implement ContextHandler
// 	- func(http.ResponseWriter, *http.Request)
// 	- func(context.Context, http.ResponseWriter, *http.Request)
type HandlerType interface{}

// ContextHandler is like http.Handler but supports context.
type ContextHandler interface {
	ServeHTTPContext(context.Context, http.ResponseWriter, *http.Request)
}

// OldContextHandler is like ContextHandler but uses the old x/net/context.
type OldContextHandler interface {
	ServeHTTPContext(netcontext.Context, http.ResponseWriter, *http.Request)
}

func old2new(old OldContextHandler) ContextHandler {
	return HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		old.ServeHTTPContext(ctx, w, r)
	})
}

// HandlerFunc is like http.HandlerFunc with context.
type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (h HandlerFunc) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h(ctx, w, r)
}

// wrap tries to turn a HandlerType into a ContextHandler
func wrap(h HandlerType) ContextHandler {
	switch x := h.(type) {
	case ContextHandler:
		return x
	case func(context.Context, http.ResponseWriter, *http.Request):
		return HandlerFunc(x)
	case func(netcontext.Context, http.ResponseWriter, *http.Request):
		return HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			x(ctx, w, r)
		})
	case http.Handler:
		return HandlerFunc(func(_ context.Context, w http.ResponseWriter, r *http.Request) {
			x.ServeHTTP(w, r)
		})
	case func(http.ResponseWriter, *http.Request):
		return HandlerFunc(func(_ context.Context, w http.ResponseWriter, r *http.Request) {
			x(w, r)
		})
	}
	panic(fmt.Errorf("unsupported HandlerType: %T", h))
}
