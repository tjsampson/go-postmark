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

// ServerTokenOpt returns an Option that sets the Postmark server API token
// used in the X-Postmark-Server-Token request header for server-scoped
// endpoints (e.g. Stats API).
func ServerTokenOpt(token string) Option {
	return func(api *API) {
		api.serverToken = token
	}
}

// TimeoutOpt returns an Option that overrides the default 10-second HTTP
// request timeout. The timeout is reconciled with the underlying *http.Client
// (if any) in New() after all options have been applied, so option order does
// not matter.
func TimeoutOpt(timeout time.Duration) Option {
	return func(api *API) {
		api.timeout = timeout
		api.timeoutSet = true
	}
}
