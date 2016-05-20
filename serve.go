// +build !appengine

package kami

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
)

func init() {
	bind.WithFlag()
	graceful.DoubleKickWindow(2 * time.Second)
}

// Serve starts kami with reasonable defaults.
// The bind address can be changed by setting the GOJI_BIND environment variable, or
// by setting the "bind" command line flag.
// Serve detects einhorn and systemd for you.
// It works exactly like zenazn/goji.
func Serve() {
	if !flag.Parsed() {
		flag.Parse()
	}

	serveListener(Handler(), bind.Default())
}

// ServeTLS is like Serve, but enables TLS using the given config.
func ServeTLS(config *tls.Config) {
	if !flag.Parsed() {
		flag.Parse()
	}

	serveListener(Handler(), tls.NewListener(bind.Default(), config))
}

// ServeListener is like Serve, but runs kami on top of an arbitrary net.Listener.
func ServeListener(listener net.Listener) {
	serveListener(Handler(), listener)
}

// Serve starts serving this mux with reasonable defaults.
// The bind address can be changed by setting the GOJI_BIND environment variable, or
// by setting the "--bind" command line flag.
// Serve detects einhorn and systemd for you.
// It works exactly like zenazn/goji. Only one mux may be served at a time.
func (m *Mux) Serve() {
	if !flag.Parsed() {
		flag.Parse()
	}

	serveListener(m, bind.Default())
}

// ServeTLS is like Serve, but enables TLS using the given config.
func (m *Mux) ServeTLS(config *tls.Config) {
	if !flag.Parsed() {
		flag.Parse()
	}

	serveListener(m, tls.NewListener(bind.Default(), config))
}

// ServeListener is like Serve, but runs kami on top of an arbitrary net.Listener.
func (m *Mux) ServeListener(listener net.Listener) {
	serveListener(m, listener)
}

// ServeListener is like Serve, but runs kami on top of an arbitrary net.Listener.
func serveListener(h http.Handler, listener net.Listener) {
	// Install our handler at the root of the standard net/http default mux.
	// This allows packages like expvar to continue working as expected.
	http.Handle("/", h)

	log.Println("Starting kami on", listener.Addr())

	graceful.HandleSignals()
	bind.Ready()
	graceful.PreHook(func() { log.Printf("kami received signal, gracefully stopping") })
	graceful.PostHook(func() { log.Printf("kami stopped") })

	err := graceful.Serve(listener, http.DefaultServeMux)

	if err != nil {
		log.Fatal(err)
	}

	graceful.Wait()
}
