package kami

import (
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
)

type key int

const (
	paramsKey key = iota
	panicKey
)

// Param returns a request URL parameter, or a blank string if it doesn't exist.
// For example, with the path /v2/papers/:page
// use kami.Param(ctx, "page") to access the :page variable.
func Param(ctx context.Context, name string) string {
	params, ok := ctx.Value(paramsKey).(httprouter.Params)
	if !ok {
		return ""
	}
	return params.ByName(name)
}

// Exception gets the "v" in panic(v). The panic details.
// Only PanicHandler will receive a context you can use this with.
func Exception(ctx context.Context) interface{} {
	return ctx.Value(panicKey)
}

func newContextWithParams(ctx context.Context, params httprouter.Params) context.Context {
	return context.WithValue(ctx, paramsKey, params)
}

func mergeParams(ctx context.Context, params httprouter.Params) context.Context {
	current, _ := ctx.Value(paramsKey).(httprouter.Params)
	current = append(current, params...)
	return context.WithValue(ctx, paramsKey, current)
}

func newContextWithException(ctx context.Context, exception interface{}) context.Context {
	return context.WithValue(ctx, panicKey, exception)
}
