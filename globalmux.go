package kami

import (
	"net/http"

	"github.com/dimfeld/httptreemux"
)

var (
	routes    = newRouter()
	enable405 = true
)

func init() {
	// set up the default 404/405 handlers
	NotFound(nil)
	MethodNotAllowed(nil)
}

func newRouter() *httptreemux.TreeMux {
	r := httptreemux.New()
	r.PathSource = httptreemux.URLPath
	r.RedirectBehavior = httptreemux.Redirect307
	r.RedirectMethodBehavior = map[string]httptreemux.RedirectBehavior{
		"GET": httptreemux.Redirect301,
	}
	return r
}

// Handler returns an http.Handler serving registered routes.
func Handler() http.Handler {
	return routes
}

// Handle registers an arbitrary method handler under the given path.
func Handle(method, path string, handler HandlerType) {
	routes.Handle(method, path, bless(wrap(handler)))
}

// Get registers a GET handler under the given path.
func Get(path string, handler HandlerType) {
	Handle("GET", path, handler)
}

// Post registers a POST handler under the given path.
func Post(path string, handler HandlerType) {
	Handle("POST", path, handler)
}

// Put registers a PUT handler under the given path.
func Put(path string, handler HandlerType) {
	Handle("PUT", path, handler)
}

// Patch registers a PATCH handler under the given path.
func Patch(path string, handler HandlerType) {
	Handle("PATCH", path, handler)
}

// Head registers a HEAD handler under the given path.
func Head(path string, handler HandlerType) {
	Handle("HEAD", path, handler)
}

// Head registers a OPTIONS handler under the given path.
func Options(path string, handler HandlerType) {
	Handle("OPTIONS", path, handler)
}

// Delete registers a DELETE handler under the given path.
func Delete(path string, handler HandlerType) {
	Handle("DELETE", path, handler)
}

// EnableMethodNotAllowed enables or disables automatic Method Not Allowed handling.
// Note that this is enabled by default.
func EnableMethodNotAllowed(enabled bool) {
	enable405 = enabled
}
