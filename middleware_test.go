package kami_test

import (
	"net/http"
	"testing"

	"github.com/guregu/kami"
	"golang.org/x/net/context"
)

func TestWildcardMiddleware(t *testing.T) {
	kami.Reset()
	kami.Use("/user/:mid/edit", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		if kami.Param(ctx, "mid") == "403" {
			w.WriteHeader(http.StatusForbidden)
			return nil
		}

		return context.WithValue(ctx, "middleware id", kami.Param(ctx, "mid"))
	})
	kami.Patch("/user/:id/edit", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if kami.Param(ctx, "mid") != kami.Param(ctx, "id") {
			t.Error("mid != id")
		}

		if ctx.Value("middleware id").(string) != kami.Param(ctx, "id") {
			t.Error("middleware values not propagating")
		}
	})
	kami.Head("/user/:id", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if ctx.Value("middleware id") != nil {
			t.Error("wildcard middleware shouldn't have been called")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// normal case
	expectResponseCode(t, "PATCH", "/user/42/edit", http.StatusOK)

	// should stop early
	expectResponseCode(t, "PATCH", "/user/403/edit", http.StatusForbidden)

	// make sure the middleware isn't over eager
	expectResponseCode(t, "HEAD", "/user/403", http.StatusOK)
}

func TestHierarchicalStop(t *testing.T) {
	kami.Reset()
	kami.Use("/nope/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		w.WriteHeader(http.StatusForbidden)
		return nil
	})
	kami.Delete("/nope/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	expectResponseCode(t, "DELETE", "/nope/test", http.StatusForbidden)
}
