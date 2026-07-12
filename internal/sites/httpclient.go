package sites

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

type ClientOptions struct {
	ProxyURL string
	Cookie   string
	Headers  map[string]string
}

func NewHTTPClient(proxyURL string) (*http.Client, error) {
	return NewHTTPClientWithOptions(ClientOptions{ProxyURL: proxyURL})
}

func NewHTTPClientWithOptions(opts ClientOptions) (*http.Client, error) {
	transport := &http.Transport{
		DialContext: (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if opts.ProxyURL != "" {
		u, err := url.Parse(opts.ProxyURL)
		if err != nil { return nil, err }
		if u.Scheme == "socks5" || u.Scheme == "socks5h" {
			dialer, err := proxy.FromURL(u, proxy.Direct)
			if err != nil { return nil, err }
			transport.Proxy = nil
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) { return dialer.Dial(network, addr) }
		} else {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	base := &http.Client{Timeout: 30 * time.Second, Transport: transport}
	if opts.Cookie == "" && len(opts.Headers) == 0 { return base, nil }
	return &http.Client{Timeout: 30 * time.Second, Transport: roundTripperWithHeaders{base: transport, cookie: opts.Cookie, headers: opts.Headers}}, nil
}

type roundTripperWithHeaders struct {
	base    http.RoundTripper
	cookie  string
	headers map[string]string
}

func (r roundTripperWithHeaders) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	for k, v := range r.headers { clone.Header.Set(k, v) }
	if r.cookie != "" { clone.Header.Set("Cookie", r.cookie) }
	return r.base.RoundTrip(clone)
}

func ParseHeadersJSON(v string) map[string]string {
	if v == "" { return map[string]string{} }
	m := map[string]string{}
	_ = json.Unmarshal([]byte(v), &m)
	return m
}
