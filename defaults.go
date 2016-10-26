// +build !appengine,!appenginevm

package kami

import (
	"net/http"

	"golang.org/x/net/context"
)

func defaultContext(ctx context.Context, r *http.Request) context.Context {
	return ctx
}
