package kami

import (
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
	params, ok := ctx.Value(paramsKey).(map[string]string)
	if !ok {
		return ""
	}
	return params[name]
}

// SetParameter will set the value of a path parameter in a given context.
func SetParameter(ctx context.Context, name string, value string) context.Context {
	params, ok := ctx.Value(paramsKey).(map[string]string)
	if !ok {
		params = make(map[string]string)
	}
	params[name] = value
	return context.WithValue(ctx, paramsKey, params)
}

// Exception gets the "v" in panic(v). The panic details.
// Only PanicHandler will receive a context you can use this with.
func Exception(ctx context.Context) interface{} {
	return ctx.Value(panicKey)
}

func newContextWithParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, paramsKey, params)
}

func mergeParams(ctx context.Context, params map[string]string) context.Context {
	current, _ := ctx.Value(paramsKey).(map[string]string)
	if current == nil {
		return context.WithValue(ctx, paramsKey, params)
	}

	for k, v := range params {
		current[k] = v
	}
	return ctx
}

func newContextWithException(ctx context.Context, exception interface{}) context.Context {
	return context.WithValue(ctx, panicKey, exception)
}
