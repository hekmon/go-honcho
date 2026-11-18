package honcho

import (
	"net/http"
	"net/url"

	"github.com/hashicorp/go-cleanhttp"
)

const (
	managedServiceURL = "https://api.honcho.dev"
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
	HTTP    *http.Client // if empty, defaults to cleanhttp.DefaultPooledClient()
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
