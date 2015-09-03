package kami_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	// "github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"

	"github.com/guregu/kami"
)

// TODO: this mostly a copy/paste of kami_test.go, rewrite it!
func TestKamiMux(t *testing.T) {
	mux := kami.New()

	// normal stuff
	mux.Use("/mux/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "test1", "1")
	})
	mux.Use("/mux/v2/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "test2", "2")
	})
	mux.Get("/mux/v2/papers/:page", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		page := kami.Param(ctx, "page")
		if page == "" {
			panic("blank page")
		}
		io.WriteString(w, page)

		test1 := ctx.Value("test1").(string)
		test2 := ctx.Value("test2").(string)

		if test1 != "1" || test2 != "2" {
			t.Error("unexpected ctx value:", test1, test2)
		}
	})

	// 404 stuff
	mux.Use("/mux/missing/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "ok", true)
	})
	mux.NotFound(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ok, _ := ctx.Value("ok").(bool)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusTeapot)
	})

	// 405 stuff
	mux.Use("/mux/method_not_allowed", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "ok", true)
	})
	mux.MethodNotAllowed(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ok, _ := ctx.Value("ok").(bool)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusTeapot)
	})
	mux.Post("/mux/method_not_allowed", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	stdMux := http.NewServeMux()
	stdMux.Handle("/mux/", mux)

	// test normal stuff
	resp := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/mux/v2/papers/3", nil)
	if err != nil {
		t.Fatal(err)
	}

	stdMux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Error("should return HTTP OK", resp.Code, "≠", http.StatusOK)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if string(data) != "3" {
		t.Error("expected page 3, got", string(data))
	}

	// test 404
	resp = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/mux/missing/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	stdMux.ServeHTTP(resp, req)
	if resp.Code != http.StatusTeapot {
		t.Error("should return HTTP Teapot", resp.Code, "≠", http.StatusTeapot)
	}

	// test 405
	resp = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/mux/method_not_allowed", nil)
	if err != nil {
		t.Fatal(err)
	}

	stdMux.ServeHTTP(resp, req)
	if resp.Code != http.StatusTeapot {
		t.Error("should return HTTP Teapot", resp.Code, "≠", http.StatusTeapot)
	}

	// test HandleMethodNotAllowed method
	resp = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/mux/method_not_allowed", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Reset NotFound handler to receive default 404 instead of custom handler 418(Teapot)
	mux.NotFound(nil)

	mux.HandleMethodNotAllowed(false)
	stdMux.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Error("should return HTTP NotFound", resp.Code, "≠", http.StatusNotFound)
	}	
}
