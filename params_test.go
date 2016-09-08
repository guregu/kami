package kami_test

import (
	"testing"

	"github.com/guregu/kami"
	"golang.org/x/net/context"
)

func TestParams(t *testing.T) {
	ctx := context.Background()
	if result := kami.Param(ctx, "test"); result != "" {
		t.Error("expected blank, got", result)
	}
	ctx = kami.SetParam(ctx, "test", "abc")
	if result := kami.Param(ctx, "test"); result != "abc" {
		t.Error("expected abc, got", result)
	}
	ctx = kami.SetParam(ctx, "test", "overwritten")
	if result := kami.Param(ctx, "test"); result != "overwritten" {
		t.Error("expected overwritten, got", result)
	}
}
