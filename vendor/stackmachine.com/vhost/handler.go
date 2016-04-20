package vhost

import "net/http"

type Handler map[string]http.Handler

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler, ok := h[r.Host]; ok {
		handler.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}
}
