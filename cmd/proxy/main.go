package main

import (
	"context"
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
	client.Timeout = 10 * time.Second

	resp, err := fetchUrl(r.Context(), client, url.String(), r.Header.Clone())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		e := fmt.Sprintf("cannot request url %s: %v", pageURL, err)
		log.Println(e)
		io.WriteString(w, e)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	for key, value := range resp.Header {
		for _, val := range value {
			w.Header().Add(key, val)
		}
	}
	io.Copy(w, resp.Body)
}

func fetchUrl(ctx context.Context, client *http.Client, url string, headers http.Header) (*http.Response, error) {
	tries := 1
	for {
		log.Printf("url: %v", url)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot create request: %v", err)
		}
		req = req.WithContext(ctx)
		delete(headers, "Host")
		req.Header = headers

		resp, err := client.Do(req)
		if err != nil {
			if tries >= 3 {
				return nil, fmt.Errorf("cannot do request: %v", err)
			}
			tries++
			continue
		}
		return resp, nil
	}
}
