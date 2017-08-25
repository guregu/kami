// +build go1.7

package kami

import (
	"context"
	"net/http"
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

// HandlerFunc is like http.HandlerFunc with context.
type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (h HandlerFunc) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h(ctx, w, r)
}
