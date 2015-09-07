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

func TestKami(t *testing.T) {
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
		t.Error("should return HTTP OK", resp.Code, "≠", http.StatusOK)
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
	kami.PanicHandler = kami.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		err := kami.Exception(ctx)
		if err != "test panic" {
			t.Error("unexpected exception:", err)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	kami.Post("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	kami.Put("/ok", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	expectResponseCode(t, "POST", "/test", http.StatusServiceUnavailable)
	if status != http.StatusServiceUnavailable {
		t.Error("log handler received wrong status code", status, "≠", http.StatusServiceUnavailable)
	}

	// test loggers without panics
	expectResponseCode(t, "PUT", "/ok", http.StatusOK)
	if status != http.StatusOK {
		t.Error("log handler received wrong status code", status, "≠", http.StatusOK)
	}
}

func TestPanickingLogger(t *testing.T) {
	kami.Reset()
	kami.LogHandler = func(ctx context.Context, w mutil.WriterProxy, r *http.Request) {
		t.Log("log handler")
		panic("test panic")
	}
	kami.PanicHandler = kami.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		t.Log("panic handler")
		err := kami.Exception(ctx)
		if err != "test panic" {
			t.Error("unexpected exception:", err)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("error 503"))
	})
	kami.Post("/test", noop)

	expectResponseCode(t, "POST", "/test", http.StatusServiceUnavailable)
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
		w.WriteHeader(http.StatusTeapot)
	})

	expectResponseCode(t, "GET", "/missing/hello", http.StatusTeapot)
}

func TestNotFoundDefault(t *testing.T) {
	kami.Reset()

	expectResponseCode(t, "GET", "/missing/hello", http.StatusNotFound)
}

func TestMethodNotAllowed(t *testing.T) {
	kami.Reset()
	kami.Use("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "ok", true)
	})
	kami.Post("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	kami.MethodNotAllowed(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ok, _ := ctx.Value("ok").(bool)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusTeapot)
	})

	expectResponseCode(t, "GET", "/test", http.StatusTeapot)
}

func TestEnableMethodNotAllowed(t *testing.T) {
	kami.Reset()
	kami.Post("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Handling enabled by default
	expectResponseCode(t, "GET", "/test", http.StatusMethodNotAllowed)

	// Not found deals with it when handling disabled
	kami.EnableMethodNotAllowed(false)
	expectResponseCode(t, "GET", "/test", http.StatusNotFound)

	// And MethodNotAllowed status when handling enabled
	kami.EnableMethodNotAllowed(true)
	expectResponseCode(t, "GET", "/test", http.StatusMethodNotAllowed)
}

func TestMethodNotAllowedDefault(t *testing.T) {
	kami.Reset()
	kami.Post("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	expectResponseCode(t, "GET", "/test", http.StatusMethodNotAllowed)
}

func TestCloseHandler(t *testing.T) {
	called := false
	closeHandler := func(ctx context.Context, r *http.Request) {
		called = true
	}

	kami.Reset()

	kami.CloseHandler = closeHandler
	kami.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		t.Log("TestCloseHandler")
	})

	expectResponseCode(t, "GET", "/test", http.StatusOK)
	if called != true {
		t.Fatal("expected closeHandler to be called")
	}
}

func noop(ctx context.Context, w http.ResponseWriter, r *http.Request) {}

func expectResponseCode(t *testing.T, method, path string, expected int) {
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatal(err)
	}

	kami.Handler().ServeHTTP(resp, req)

	if resp.Code != expected {
		t.Error("should return HTTP", http.StatusText(expected)+":", resp.Code, "≠", expected)
	}
}
