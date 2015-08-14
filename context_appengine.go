// +build appengine

package kami

import (
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
)

func defaultContext(ctx context.Context, r *http.Request) context.Context {
	return appengine.WithContext(ctx, r)
}
