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

	http.Handle("/", &handler{p})

	log.Fatal(http.ListenAndServe("0.0.0.0:80", nil))
}

type handler struct {
	p *pool.ProxyListDownloadPool
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

	client, err := proxi.Client(h.p)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		e := fmt.Sprintf("cannot create client: %v", err)
		log.Println(e)
		io.WriteString(w, e)
		return
	}

	f := proxi.NewFetcher(client, 10*time.Second, 5)

	resp, err := f.Fetch(r.Context(), url.String(), r.Header.Clone())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		e := fmt.Sprintf("cannot request url %s: %v", pageURL, err)
		log.Println(e)
		io.WriteString(w, e)
		return
	}

	w.Write([]byte(resp))
}
