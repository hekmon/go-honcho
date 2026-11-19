package honcho

import (
	"net"
	"net/http"
	"net/url"
	"runtime"
	"time"
)

const (
	managedServiceURL = "https://api.honcho.dev"

	// defaultIdleConnTimeout is how long the default HTTP client keeps an idle
	// connection. It must stay below the 5 seconds after which Honcho's server
	// (uvicorn) closes an idle connection: a request written as the server closes
	// the connection fails with EOF, and the transport does not retry a POST,
	// which most of Honcho's list and search calls are.
	defaultIdleConnTimeout = 4 * time.Second
)

var (
	ManagedServiceURL *url.URL
)

func init() {
	var err error
	if ManagedServiceURL, err = url.Parse(managedServiceURL); err != nil {
		panic(err)
	}
}

type Options struct {
	BaseURL *url.URL     // if empty, defaults to ManagedServiceURL
	APIKey  string       // if empty, authorization header will not be set
	HTTP    *http.Client // if empty, defaults to NewHTTPClient(); if set, used as is: keep its idle timeout below 5 seconds, see NewHTTPClient
}

func New(options *Options) *Client {
	// The defaults go into a copy: the caller's options stay as given.
	var opts Options
	if options != nil {
		opts = *options
	}
	if opts.BaseURL == nil {
		opts.BaseURL = ManagedServiceURL
	}
	if opts.HTTP == nil {
		opts.HTTP = NewHTTPClient()
	}
	return &Client{
		baseURL: opts.BaseURL,
		apiKey:  opts.APIKey,
		http:    opts.HTTP,
	}
}

// NewHTTPClient returns a new HTTP client with go-honcho's defaults: New makes
// one when Options.HTTP is nil. To adjust the client, for example its Timeout,
// start from NewHTTPClient and pass the result in Options.HTTP.
//
// Each call builds a new transport, sharing nothing with http.DefaultTransport,
// which other packages may change. Honcho's server closes a connection left
// idle for 5 seconds, so the transport lets go of it sooner: a request written
// on a connection the server is closing fails with EOF, and the transport does
// not retry a POST. A client made by hand must also keep its transport's
// IdleConnTimeout below 5 seconds.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			// Go through the proxy set by HTTP_PROXY, HTTPS_PROXY and NO_PROXY, if any.
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second, // give up on connecting after 30 seconds
				KeepAlive: 30 * time.Second, // send TCP keep-alive probes every 30 seconds
			}).DialContext,
			// Attempt HTTP/2, which setting DialContext would otherwise turn off.
			ForceAttemptHTTP2: true,
			// Keep at most 100 idle connections across all hosts.
			MaxIdleConns: 100,
			// A client talks to a single host: keep up to GOMAXPROCS+1 idle
			// connections to it, not the standard library's 2, so that
			// concurrent requests find one to reuse.
			MaxIdleConnsPerHost: runtime.GOMAXPROCS(0) + 1,
			// Close an idle connection before Honcho's server does, see
			// defaultIdleConnTimeout.
			IdleConnTimeout: defaultIdleConnTimeout,
			// Give up on a TLS handshake after 10 seconds.
			TLSHandshakeTimeout: 10 * time.Second,
			// Wait at most 1 second for the server's go-ahead before sending
			// the body of a request with an "Expect: 100-continue" header.
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

type Client struct {
	baseURL *url.URL
	apiKey  string
	http    *http.Client
}
