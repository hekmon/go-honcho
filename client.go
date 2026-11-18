package honcho

import (
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-cleanhttp"
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
	HTTP    *http.Client // if empty, defaults to cleanhttp.DefaultPooledClient() with a 4 second IdleConnTimeout; if set, used as is
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
		opts.HTTP = cleanhttp.DefaultPooledClient()
		opts.HTTP.Transport.(*http.Transport).IdleConnTimeout = defaultIdleConnTimeout
	}
	return &Client{
		baseURL: opts.BaseURL,
		apiKey:  opts.APIKey,
		http:    opts.HTTP,
	}
}

type Client struct {
	baseURL *url.URL
	apiKey  string
	http    *http.Client
}
