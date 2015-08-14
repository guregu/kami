// +build appengine

package kami

import (
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
)

func defaultContext(r *http.Request, c context.Context) context.Context {
	return appengine.NewContext(r)
}
