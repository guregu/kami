package kami_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zenazn/goji/web/mutil"
	"golang.org/x/net/context"

	"github.com/guregu/kami"
)

func TestParams(t *testing.T) {
	kami.Use("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "test1", "1")
	})
	kami.Use("/v2/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		return context.WithValue(ctx, "test2", "2")
	})
	kami.Get("/v2/papers/:page", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		page, ok := kami.Param(ctx, "page")
		if !ok {
			panic("not ok")
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
	if resp.Code != 200 {
		t.Error("should return HTTP OK", resp.Code, "≠", 200)
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
	status := 0
	kami.LogHandler = func(ctx context.Context, w mutil.WriterProxy, r *http.Request) {
		status = w.Status()
	}
	kami.PanicHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		err := kami.Exception(ctx)
		if err != "test panic" {
			t.Error("unexpected exception:", err)
		}
		w.WriteHeader(500)
		w.Write([]byte("error 500"))
	}
	kami.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	resp := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	kami.Handler().ServeHTTP(resp, req)
	if resp.Code != 500 {
		t.Error("should return HTTP 500", resp.Code, "≠", 500)
	}
	if status != 500 {
		t.Error("should return HTTP 500", status, "≠", 500)
	}
}

func BenchmarkPath(b *testing.B) {
	m := make(map[string]bool)

	for i := 0; i < b.N; i++ {
		//path := "/v2/a/thing/qqq"
		path := "/1/2/3/five"
		split := strings.SplitAfter(path, "/")
		for j, _ := range split {
			path := strings.Join(split[0:j+1], "")
			_, ok := m[path]
			_ = ok
			// log.Println(j, path)
		}
	}
}
