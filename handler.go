package kami

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"
)

/*
HandlerType is the type of Handlers and types that kami internally converts to
ContextHandler. In order to provide an expressive API, this type is an alias for
interface{} that is named for the purposes of documentation, however only the
following concrete types are accepted:
	- types that implement http.Handler
	- types that implement ContextHandler
	- func(http.ResponseWriter, *http.Request)
	- func(context.Context, http.ResponseWriter, *http.Request)
*/
type HandlerType interface{}

// ContextHandler is like http.Handler but supports context.
type ContextHandler interface {
	ServeHTTPContext(context.Context, http.ResponseWriter, *http.Request)
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
