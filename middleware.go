package kami

import (
	"strings"

	"github.com/guregu/kami/treemux"
)

type wares struct {
	middleware     map[string][]Middleware
	afterware      map[string][]Afterware
	wildcards      *treemux.TreeMux
	afterWildcards *treemux.TreeMux
}

func newWares() *wares {
	return new(wares)
}

// Use registers middleware to run for the given path.
// See the global Use function's documents for information on how middleware works.
func (m *wares) Use(path string, mw MiddlewareType) {
	if containsWildcard(path) {
		if m.wildcards == nil {
			m.wildcards = treemux.New()
		}
		m.wildcards.Set(path, convert(mw))
	} else {
		if m.middleware == nil {
			m.middleware = make(map[string][]Middleware)
		}
		fn := convert(mw)
		chain := m.middleware[path]
		chain = append(chain, fn)
		m.middleware[path] = chain
	}
}

// After registers middleware to run for the given path after normal middleware added with Use has run.
// See the global After function's documents for information on how middleware works.
func (m *wares) After(path string, afterware AfterwareType) {
	aw := convertAW(afterware)
	if containsWildcard(path) {
		if m.afterWildcards == nil {
			m.afterWildcards = treemux.New()
		}
		m.afterWildcards.Set(path, aw)
	} else {
		if m.afterware == nil {
			m.afterware = make(map[string][]Afterware)
		}
		m.afterware[path] = append([]Afterware{aw}, m.afterware[path]...)
	}
}

var defaultMW = newWares() // for the global router

// Use registers middleware to run for the given path.
// Middleware will be executed hierarchically, starting with the least specific path.
// Middleware under the same path will be executed in order of registration.
// You may use wildcards in the path. Wildcard middleware will be run last,
// after all hierarchical middleware has run.
//
// Adding middleware is not threadsafe.
//
// WARNING: kami middleware is run in sequence, but standard middleware is chained;
// middleware that expects its code to run after the next handler, such as
// standard loggers and panic handlers, will not work as expected.
// Use kami.LogHandler and kami.PanicHandler instead.
// Standard middleware that does not call the next handler to stop the request is supported.
func Use(path string, mw MiddlewareType) {
	defaultMW.Use(path, mw)
}

// After registers afterware to run after middleware and the request handler has run.
// Afterware is like middleware, but everything is in reverse.
// Afterware will be executed hierarchically, starting with wildcards and then
// the most specific path, ending with /.
// Afterware under the same path will be executed in the opposite order of registration.
func After(path string, aw AfterwareType) {
	defaultMW.After(path, aw)
}

// Middleware run functions are in versioned files.

func (m *wares) needsWrapper() bool {
	return m.afterware != nil || m.afterWildcards != nil
}

func containsWildcard(path string) bool {
	return strings.Contains(path, "/:") || strings.Contains(path, "/*")
}
