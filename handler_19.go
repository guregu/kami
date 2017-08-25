// +build go1.9

package kami

import (
	"context"
	"fmt"
	"net/http"
)

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
