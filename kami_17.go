// +build go1.7

package kami

import (
	"context"
	"net/http"

	"github.com/zenazn/goji/web/mutil"
)

// kami is the heart of the package.
// It wraps a ContextHandler into an httprouter compatible request,
// in order to run all the middleware and other special handlers.
type kami struct {
	handler      ContextHandler
	base         *context.Context
	middleware   *wares
	panicHandler *HandlerType
	logHandler   *func(context.Context, mutil.WriterProxy, *http.Request)
}

func (k kami) handle(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var (
		ctx           = defaultContext(*k.base, r)
		handler       = k.handler
		mw            = *k.middleware
		panicHandler  = *k.panicHandler
		logHandler    = *k.logHandler
		ranLogHandler = false // track this in case the log handler blows up
	)
	if len(params) > 0 {
		ctx = newContextWithParams(ctx, params)
	}

	if ctx != context.Background() {
		r = r.WithContext(ctx)
	}

	var proxy mutil.WriterProxy
	if logHandler != nil || mw.needsWrapper() {
		proxy = mutil.WrapWriter(w)
		w = proxy
	}

	if panicHandler != nil {
		defer func() {
			if err := recover(); err != nil {
				ctx = newContextWithException(ctx, err)
				r = r.WithContext(ctx)
				wrap(panicHandler).ServeHTTPContext(ctx, w, r)

				if logHandler != nil && !ranLogHandler {
					logHandler(ctx, proxy, r)
					// should only happen if header hasn't been written
					proxy.WriteHeader(http.StatusInternalServerError)
				}
			}
		}()
	}

	r, ctx, ok := mw.run(ctx, w, r)
	if ok {
		handler.ServeHTTPContext(ctx, w, r)
	}
	if proxy != nil {
		r, ctx = mw.after(ctx, proxy, r)
	}

	if logHandler != nil {
		ranLogHandler = true
		logHandler(ctx, proxy, r)
		// should only happen if header hasn't been written
		proxy.WriteHeader(http.StatusInternalServerError)
	}
}
