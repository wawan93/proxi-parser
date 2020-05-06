package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/wawan93/proxi"
	"github.com/wawan93/proxi/pool"
)

func main() {
	cfg := &pool.ProxyListDownloadConfig{
		Type: "https",
	}
	p := pool.NewProxyListDownloadPool(cfg)

	if err := p.Update(); err != nil {
		log.Fatalf("cannot get proxies: %v", err)
	}

	go func() {
		for {
			select {
			case <-time.After(35 * time.Minute):
				p.Update()
			}
		}
	}()

	client, err := proxi.Client(p)
	if err != nil {
		log.Fatal(err)
	}

	f := proxi.NewFetcher(client, 10*time.Second, 5)

	http.Handle("/", &handler{f})

	log.Fatal(http.ListenAndServe("0.0.0.0:80", nil))
}

type handler struct {
	f proxi.Fetcher
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, fmt.Sprintf("cannot decode request url: %v", err))
		return
	}
	pageURL := query.Get("url")
	url, err := url.Parse(pageURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		e := fmt.Sprintf("cannot parse url %s: %v", pageURL, err)
		log.Println(e)
		io.WriteString(w, e)
		return
	}

	resp, err := h.f.Fetch(r.Context(), url.String(), r.Header.Clone())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		e := fmt.Sprintf("cannot request url %s: %v", pageURL, err)
		log.Println(e)
		io.WriteString(w, e)
		return
	}

	w.Write([]byte(resp))
}
