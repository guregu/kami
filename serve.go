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
// It works (exactly) like Goji, looking for Einhorn, the bind flag, GOJI_BIND...
func Serve() {
	if !flag.Parsed() {
		flag.Parse()
	}

	ServeListener(bind.Default())
}

// ServeTLS is like Serve, but enables TLS using the given config.
func ServeTLS(config *tls.Config) {
	if !flag.Parsed() {
		flag.Parse()
	}

	ServeListener(tls.NewListener(bind.Default(), config))
}

// ServeListener is like Serve, but runs kami on top of an arbitrary net.Listener.
func ServeListener(listener net.Listener) {
	// Install our handler at the root of the standard net/http default mux.
	// This allows packages like expvar to continue working as expected.
	http.Handle("/", Handler())

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
