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
	if options == nil {
		options = &Options{
			BaseURL: ManagedServiceURL,
			HTTP:    cleanhttp.DefaultPooledClient(),
		}
	} else {
		if options.BaseURL == nil {
			options.BaseURL = ManagedServiceURL
		}
		if options.HTTP == nil {
			options.HTTP = cleanhttp.DefaultPooledClient()
		}
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
