package kami_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"

	"github.com/guregu/kami"
)

func routeBench(b *testing.B, route string) {
	kami.Reset()
	kami.Use("/Z/", noopMW)
	kami.After("/Z/", noopMW)
	kami.Get(route, noop)
	req, _ := http.NewRequest("GET", route, nil)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		kami.Handler().ServeHTTP(resp, req)
	}
}

func BenchmarkShortRoute(b *testing.B) {
	routeBench(b, "/hello")
}

func BenchmarkLongRoute(b *testing.B) {
	routeBench(b, "/aaaaaaaaaaaa/")
}

func BenchmarkDeepRoute(b *testing.B) {
	routeBench(b, "/a/b/c/d/e/f/g")
}

func BenchmarkDeepRouteUnicode(b *testing.B) {
	routeBench(b, "/ä/蜂/海/🐶/神/🍺/🍻")
}

func BenchmarkSuperDeepRoute(b *testing.B) {
	routeBench(b, "/a/b/c/d/e/f/g/h/i/l/k/l/m/n/o/p/q/r/hello world")
}

// Param benchmarks test accessing URL params

func BenchmarkParameter(b *testing.B) {
	kami.Reset()
	kami.Get("/hello/:name", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		kami.Param(ctx, "name")
	})
	req, _ := http.NewRequest("GET", "/hello/bob", nil)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		kami.Handler().ServeHTTP(resp, req)
	}
}

func BenchmarkParameter5(b *testing.B) {
	kami.Reset()
	kami.Get("/:a/:b/:c/:d/:e", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		for _, v := range []string{"a", "b", "c", "d", "e"} {
			kami.Param(ctx, v)
		}
	})
	req, _ := http.NewRequest("GET", "/a/b/c/d/e", nil)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		kami.Handler().ServeHTTP(resp, req)
	}
}

// Middleware tests setting and using values with middleware
// These test the speed of kami's middleware engine AND using
// x/net/context to store values, so it could be a somewhat
// realitic idea of what using kami would be like.

func BenchmarkMiddleware(b *testing.B) {
	kami.Reset()
	kami.Use("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "test", "ok")
	})
	kami.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if ctx.Value("test") != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	req, _ := http.NewRequest("GET", "/test", nil)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		kami.Handler().ServeHTTP(resp, req)
	}
}

func BenchmarkMiddleware5(b *testing.B) {
	kami.Reset()
	numbers := []int{1, 2, 3, 4, 5}
	for _, n := range numbers {
		n := n // wtf
		kami.Use("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			return context.WithValue(ctx, n, n)
		})
	}
	kami.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		for _, n := range numbers {
			if ctx.Value(n) != n {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
	})
	req, _ := http.NewRequest("GET", "/test", nil)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		kami.Handler().ServeHTTP(resp, req)
	}
}

func BenchmarkMiddleware1Afterware1(b *testing.B) {
	kami.Reset()
	numbers := []int{1}
	for _, n := range numbers {
		n := n // wtf
		kami.Use("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			return context.WithValue(ctx, n, n)
		})
	}
	kami.After("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		for _, n := range numbers {
			if ctx.Value(n) != n {
				panic(n)
			}
		}
		return ctx
	})
	kami.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		// ...
	})
	req, _ := http.NewRequest("GET", "/test", nil)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		kami.Handler().ServeHTTP(resp, req)
	}
}

func BenchmarkMiddleware5Afterware1(b *testing.B) {
	kami.Reset()
	numbers := []int{1, 2, 3, 4, 5}
	for _, n := range numbers {
		n := n // wtf
		kami.Use("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
			return context.WithValue(ctx, n, n)
		})
	}
	kami.After("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		for _, n := range numbers {
			if ctx.Value(n) != n {
				panic(n)
			}
		}
		return ctx
	})
	kami.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		for _, n := range numbers {
			if ctx.Value(n) != n {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
	})
	req, _ := http.NewRequest("GET", "/test", nil)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		kami.Handler().ServeHTTP(resp, req)
	}
}

// This tests just the URL walking middleware engine.
func BenchmarkMiddlewareAfterwareMiss(b *testing.B) {
	kami.Reset()
	kami.Use("/dog/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return nil
	})
	kami.After("/dog/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return nil
	})
	kami.Get("/a/bbb/cc/d/e", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req, _ := http.NewRequest("GET", "/a/bbb/cc/d/e", nil)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		kami.Handler().ServeHTTP(resp, req)
	}
}
