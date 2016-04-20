package image

import (
	"net/http"
	"net/http/httputil"
	"regexp"
)

var pattern = regexp.MustCompile(`^/mtg/multiverseid/(\d+)\.jpg$`)

func NewSingleHostReverseProxy() *httputil.ReverseProxy {

	director := func(req *http.Request) {
		// Incoming requests: /mtg/multiverseid/\d+.jpg
		matches := pattern.FindStringSubmatch(req.URL.Path)
		id := "0"

		if matches != nil && len(matches) > 1 {
			id = matches[1]
		}

		// Gatherer is confused about Westvale Abbey and Ormendahl, Profane
		// Prince. For now, we switch the images to make everything line up. My
		// guess is that mtgjson will eventually fix this, and then I'll need
		// to remove this hack
		switch id {
		case "410049":
			id = "410050"
		case "410050":
			id = "410049"
		}

		req.URL.Scheme = "http"
		req.URL.Host = "gatherer.wizards.com"
		req.URL.Path = "/Handlers/Image.ashx"
		values := req.URL.Query()
		values.Set("type", "card")
		values.Set("multiverseid", id)
		req.URL.RawQuery = values.Encode()
	}
	return &httputil.ReverseProxy{Director: director}
}

func images(w http.ResponseWriter, r *http.Request) {
	// change the request host to match the target
	r.Host = "gatherer.wizards.com"
	proxy := NewSingleHostReverseProxy()
	proxy.ServeHTTP(w, r)
}

func New() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/mtg/multiverseid/", images)
	return mux
}
