package kami_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"

	"github.com/guregu/kami"
)

func TestParams(t *testing.T) {
	kami.Reset()
	kami.Use("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "test1", "1")
	})
	kami.Use("/v2/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "test2", "2")
	})
	kami.Get("/v2/papers/:page", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

	resp := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v2/papers/3", nil)
	if err != nil {
		t.Fatal(err)
	}

	kami.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Error("should return HTTP StatusOK(200)", resp.Code, "≠", http.StatusOK)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if string(data) != "3" {
		t.Error("expected page 3, got", string(data))
	}
}

func TestLoggerAndPanic(t *testing.T) {
	kami.Reset()
	// test logger with panic
	status := 0
	kami.LogHandler = func(ctx context.Context, w mutil.WriterProxy, r *http.Request) {
		status = w.Status()
	}
	kami.PanicHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		err := kami.Exception(ctx)
		if err != "test panic" {
			t.Error("unexpected exception:", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error StatusInternalServerError(500)"))
	}
	kami.Post("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	kami.Put("/ok", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	resp := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	kami.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusInternalServerError {
		t.Error("should return HTTP StatusInternalServerError(500)", resp.Code, "≠", http.StatusInternalServerError)
	}
	if status != http.StatusInternalServerError {
		t.Error("should return HTTP StatusInternalServerError(500)", status, "≠", http.StatusInternalServerError)
	}

	// test loggers without panics
	resp = httptest.NewRecorder()
	req, err = http.NewRequest("PUT", "/ok", nil)
	if err != nil {
		t.Fatal(err)
	}

	kami.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Error("should return HTTP StatusOK(200)", resp.Code, "≠", http.StatusOK)
	}
	if status != http.StatusOK {
		t.Error("should return HTTP StatusOK(200)", status, "≠", http.StatusOK)
	}
}

func TestPanickingLogger(t *testing.T) {
	kami.Reset()
	kami.LogHandler = func(ctx context.Context, w mutil.WriterProxy, r *http.Request) {
		t.Log("log handler")
		panic("test panic")
	}
	kami.PanicHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		t.Log("panic handler")
		err := kami.Exception(ctx)
		if err != "test panic" {
			t.Error("unexpected exception:", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error StatusInternalServerError(500)"))
	}
	kami.Post("/test", noop)

	resp := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	kami.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusInternalServerError {
		t.Error("should return HTTP StatusInternalServerError(500)", resp.Code, "≠", http.StatusInternalServerError)
	}
}

func TestNotFound(t *testing.T) {
	kami.Reset()
	kami.Use("/missing/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "ok", true)
	})
	kami.NotFound(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ok, _ := ctx.Value("ok").(bool)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(420)
	})

	resp := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/missing/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	kami.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Error("should return HTTP StatusNotFound(404)", resp.Code, "≠", http.StatusNotFound)
	}
}

func TestNotFoundDefault(t *testing.T) {
	kami.Reset()

	resp := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/missing/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	kami.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Error("should return HTTP StatusNotFound(404)", resp.Code, "≠", http.StatusNotFound)
	}
}

func noop(ctx context.Context, w http.ResponseWriter, r *http.Request) {}
