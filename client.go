package honcho

import "net/http"

type Client struct {
	http *http.Client
}
