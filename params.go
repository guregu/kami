package kami

import (
	"golang.org/x/net/context"
)

type key int
type param string

const (
	paramsKey key = iota
	panicKey
)

// Param returns a request path parameter, or a blank string if it doesn't exist.
// For example, with the path /v2/papers/:page
// use kami.Param(ctx, "page") to access the :page variable.
func Param(ctx context.Context, name string) string {
	value, _ := ctx.Value(param(name)).(string)
	return value
}

// SetParam will set the value of a path parameter in a given context.
// This is intended for testing and should not be used otherwise.
func SetParam(ctx context.Context, name string, value string) context.Context {
	return context.WithValue(ctx, param(name), value)
}

// Exception gets the "v" in panic(v). The panic details.
// Only PanicHandler will receive a context you can use this with.
func Exception(ctx context.Context) interface{} {
	return ctx.Value(panicKey)
}

func newContextWithParams(ctx context.Context, params map[string]string) context.Context {
	for k, v := range params {
		ctx = SetParam(ctx, k, v)
	}
	return ctx
}

func mergeParams(ctx context.Context, params map[string]string) context.Context {
	for k, v := range params {
		if Param(ctx, k) != v {
			ctx = SetParam(ctx, k, v)
		}
	}
	return ctx
}

func newContextWithException(ctx context.Context, exception interface{}) context.Context {
	return context.WithValue(ctx, panicKey, exception)
}
