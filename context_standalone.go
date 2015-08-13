// +build !appengine

package kami

import (
	"net/http"

	"golang.org/x/net/context"
)

func defaultContext(r *http.Request, c context.Context) context.Context {
	return c
}
