package postmark

import (
	"net/http"
	"time"
)

// HTTPClientOpt returns an Option that replaces the default *http.Client
// with the provided one. Useful for injecting a mock in tests or for
// customizing transport settings (e.g. TLS, proxies).
func HTTPClientOpt(client *http.Client) Option {
	return func(api *API) {
		api.client = client
	}
}

// APITokenOpt returns an Option that sets the Postmark account API token
// used in the X-Postmark-Account-Token request header.
func APITokenOpt(token string) Option {
	return func(api *API) {
		api.token = token
	}
}

// TimeoutOpt returns an Option that overrides the default 10-second HTTP
// request timeout on the underlying client.
func TimeoutOpt(timeout time.Duration) Option {
	return func(api *API) {
		api.timeout = timeout
	}
}
