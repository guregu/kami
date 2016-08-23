// +build !go1.7

package kami_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"

	"github.com/guregu/kami"
)

func TestKami(t *testing.T) {
	kami.Reset()

	expect := func(ctx context.Context, i int) context.Context {
		if prev := ctx.Value(i - 1).(int); prev != i-1 {
			t.Error("missing", i)
		}
		if curr := ctx.Value(i); curr != nil {
			t.Error("pre-existing", i)
		}
		return context.WithValue(ctx, i, i)
	}

	kami.Use("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		ctx = context.WithValue(ctx, 1, 1)
		ctx = context.WithValue(ctx, "handler", new(bool))
		ctx = context.WithValue(ctx, "done", new(bool))
		ctx = context.WithValue(ctx, "recovered", new(bool))
		return ctx
	})
	kami.Use("/a/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		ctx = expect(ctx, 2)
		return ctx
	})
	kami.Use("/a/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		ctx = expect(ctx, 3)
		return ctx
	})
	kami.Use("/a/b", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		ctx = expect(ctx, 4)
		return ctx
	})
	kami.Use("/a/*files", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		ctx = expect(ctx, 5)
		return ctx
	})
	kami.Get("/a/b", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if prev := ctx.Value(5).(int); prev != 5 {
			t.Error("handler: missing", 5)
		}
		*(ctx.Value("handler").(*bool)) = true

		w.WriteHeader(http.StatusTeapot)
	})
	kami.After("/a/*files", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		ctx = expect(ctx, 6)
		if !*(ctx.Value("handler").(*bool)) {
			t.Error("ran before handler")
		}
		return ctx
	})
	kami.After("/a/b", kami.Afterware(func(ctx context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
		ctx = expect(ctx, 7)
		return ctx
	}))
	kami.After("/a/", func(ctx context.Context) context.Context {
		ctx = expect(ctx, 9)
		return ctx
	})
	kami.After("/a/", func(ctx context.Context) context.Context {
		ctx = expect(ctx, 8)
		return ctx
	})
	kami.After("/", func(ctx context.Context, w mutil.WriterProxy, r *http.Request) context.Context {
		if status := w.Status(); status != http.StatusTeapot {
			t.Error("wrong status", status)
		}

		ctx = expect(ctx, 10)
		*(ctx.Value("done").(*bool)) = true
		panic("🍣")
		return nil
	})
	kami.PanicHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if got := kami.Exception(ctx); got.(string) != "🍣" {
			t.Error("panic handler: expected sushi, got", got)
		}
		if !*(ctx.Value("done").(*bool)) {
			t.Error("didn't finish")
		}
		*(ctx.Value("recovered").(*bool)) = true
	}
	kami.LogHandler = func(ctx context.Context, w mutil.WriterProxy, r *http.Request) {
		if !*(ctx.Value("recovered").(*bool)) {
			t.Error("didn't recover")
		}
	}

	expectResponseCode(t, "GET", "/a/b", http.StatusTeapot)
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
	})
	kami.Options("/test", noop)

	expectResponseCode(t, "OPTIONS", "/test", http.StatusServiceUnavailable)
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

func noop(ctx context.Context, w http.ResponseWriter, r *http.Request) {}

func noopMW(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	return ctx
}

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
