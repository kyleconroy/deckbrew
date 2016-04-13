package main

import (
	"net/http"
	"strconv"

	"github.com/opentracing/opentracing-go"

	"goji.io"
	"goji.io/middleware"
	"goji.io/pat"
	"golang.org/x/net/context"
)

func Headers(next goji.Handler) goji.Handler {
	mw := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mtg/cards/random" {
			w.Header().Set("Cache-Control", "public,max-age=3600")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "link,content-length")
		w.Header().Set("License", "The textual information presented through this API about Magic: The Gathering is copyrighted by Wizards of the Coast.")
		w.Header().Set("Disclaimer", "This API is not produced, endorsed, supported, or affiliated with Wizards of the Coast.")
		w.Header().Set("Pricing", "store.tcgplayer.com allows you to buy cards from any of our vendors, all at the same time, in a simple checkout experience. Shop, Compare & Save with TCGplayer.com!")
		w.Header().Set("Strict-Transport-Security", "max-age=86400")
		next.ServeHTTPC(ctx, w, r)
	}
	return goji.HandlerFunc(mw)
}

const timeFormat = "2006-01-02T15:04:05.999999999Z"

type responseWriter struct {
	status int
	http.ResponseWriter
}

func (crw *responseWriter) WriteHeader(status int) {
	crw.status = status
	crw.ResponseWriter.WriteHeader(status)
}

func Tracing(next goji.Handler) goji.Handler {
	return goji.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		name := "/not-found"
		pattern := middleware.Pattern(ctx)
		if ppattern, ok := pattern.(*pat.Pattern); ok {
			name = ppattern.String()
		}

		span, nctx := opentracing.StartSpanFromContext(ctx, name)
		defer span.Finish()

		sw := responseWriter{ResponseWriter: w}
		next.ServeHTTPC(nctx, &sw, r)

		span.SetTag("http/host", r.Host)
		span.SetTag("http/url", r.URL.String())
		span.SetTag("http/response/size", w.Header().Get("Content-Length"))
		span.SetTag("http/method", r.Method)
		span.SetTag("http/status_code", strconv.Itoa(sw.status))
	})
}
