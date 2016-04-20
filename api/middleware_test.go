package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"

	"goji.io"
	"goji.io/middleware"
	"goji.io/pat"
)

func TestTracing(t *testing.T) {
	mux := goji.NewMux()

	var name string
	mux.HandleFuncC(pat.Get("/mtg/cards/:id"),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			pattern := middleware.Pattern(ctx).(*pat.Pattern)
			name = pattern.String()
		})

	req, _ := http.NewRequest("GET", "/mtg/cards/foo", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if name != "/mtg/cards/:id" {
		t.Errorf("name is %s", name)
	}
}
