package kami

import (
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
)

type key int

var (
	paramsKey key = 0
	panicKey      = 1
)

func newContextWithParams(ctx context.Context, params httprouter.Params) context.Context {
	return context.WithValue(ctx, paramsKey, params)
}

func newContextWithException(ctx context.Context, exception interface{}) context.Context {
	return context.WithValue(ctx, panicKey, exception)
}

// Params returns a request URL parameter, or a blank string if it doesn't exist.
// For example, with the path /v2/papers/:page
// use kami.Param(ctx, "page") to access the :page variable.
func Param(ctx context.Context, name string) string {
	params, ok := ctx.Value(paramsKey).(httprouter.Params)
	if !ok {
		return ""
	}
	return params.ByName(name)
}

// Exception gets the panic(details) when in a panic recovery.
func Exception(ctx context.Context) interface{} {
	return ctx.Value(panicKey)
}
