package kami

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"
)

type HandleFn func(context.Context, http.ResponseWriter, *http.Request)

var (
	Context      = context.Background()
	PanicHandler HandleFn
	LogHandler   func(context.Context, mutil.WriterProxy, *http.Request)
)

var routes = httprouter.New()

func Handler() http.Handler {
	return routes
}

func Handle(method, path string, handle HandleFn) {
	routes.Handle(method, path, wrap(handle))
}

func Get(path string, handle HandleFn) {
	Handle("GET", path, handle)
}

func Post(path string, handle HandleFn) {
	Handle("POST", path, handle)
}

func Put(path string, handle HandleFn) {
	Handle("PUT", path, handle)
}

func Patch(path string, handle HandleFn) {
	Handle("PATCH", path, handle)
}

func Head(path string, handle HandleFn) {
	Handle("HEAD", path, handle)
}

// Panic sets the global panic handler
func Panic(handle HandleFn) {
	routes.PanicHandler = func(w http.ResponseWriter, r *http.Request, mystery interface{}) {
		// TODO use local, not root contest
		handle(Context, w, r)
	}
}

func wrap(k HandleFn) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := newContextWithParams(Context, params)
		ranLogHandler := false

		writer := w
		var wrapped mutil.WriterProxy
		if LogHandler != nil {
			wrapped = mutil.WrapWriter(w)
			writer = wrapped
		}

		if PanicHandler != nil {
			defer func() {
				if err := recover(); err != nil {
					ctx = newContextWithException(ctx, err)
					PanicHandler(ctx, writer, r)

					if LogHandler != nil && !ranLogHandler {
						LogHandler(ctx, wrapped, r)
						// should only happen if header hasn't been written
						wrapped.WriteHeader(500)
					}
				}
			}()
		}

		ctx, ok := run(ctx, writer, r)
		if !ok {
			return
		}
		k(ctx, writer, r)

		if LogHandler != nil {
			ranLogHandler = true
			LogHandler(ctx, wrapped, r)
			// should only happen if header hasn't been written
			wrapped.WriteHeader(500)
		}
	}
}
