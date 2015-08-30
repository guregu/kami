// +build appengine

package kami

import (
	"net/http"
)

// Serve starts kami with reasonable defaults.
func Serve() {
	http.Handle("/", Handler())
}
