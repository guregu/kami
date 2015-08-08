package kami_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"

	"github.com/timcooijmans/kami"
)

func BenchmarkStaticRoute(b *testing.B) {
	kami.Reset()
	kami.Get("/hello", noop)
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/hello", nil)
		kami.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			panic(resp.Code)
		}
	}
}

// Param benchmarks test accessing URL params

func BenchmarkParameter(b *testing.B) {
	kami.Reset()
	kami.Get("/hello/:name", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		kami.Param(ctx, "name")
	})
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/hello/bob", nil)
		kami.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			panic(resp.Code)
		}
	}
}

func BenchmarkParameter5(b *testing.B) {
	kami.Reset()
	kami.Get("/:a/:b/:c/:d/:e", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		for _, v := range []string{"a", "b", "c", "d", "e"} {
			kami.Param(ctx, v)
		}
	})
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/a/b/c/d/e", nil)
		kami.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			panic(resp.Code)
		}
	}
}

// Middleware tests setting and using values with middleware

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
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		kami.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			panic(resp.Code)
		}
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
	for n := 0; n < b.N; n++ {
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		kami.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			panic(resp.Code)
		}
	}
}
