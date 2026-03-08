package honcho

import (
	"net/http"
	"net/url"
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
	BaseURL *url.URL
	APIKey  string
}

func New(options Options) *Client {
	if options.BaseURL == nil {
		options.BaseURL = ManagedServiceURL
	}
	return &Client{
		baseURL: options.BaseURL,
		apiKey:  options.APIKey,
		http:    http.DefaultClient,
	}
}

type Client struct {
	baseURL *url.URL
	apiKey  string
	http    *http.Client
}
