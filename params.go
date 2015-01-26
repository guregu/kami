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

// Params returns the request URL parameter.
// For example, with the path /v2/papers/:page
// use kami.Param(ctx, "page") to access the :page variable.
func Param(ctx context.Context, name string) (string, bool) {
	params, ok := ctx.Value(paramsKey).(httprouter.Params)
	if !ok {
		return "", false
	}
	return params.ByName(name), true
}

// Exception gets the panic(details) when in a panic recovery.
func Exception(ctx context.Context) interface{} {
	return ctx.Value(panicKey)
}
