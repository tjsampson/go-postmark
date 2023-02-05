package postmark

import (
	"net/http"
	"time"
)

func HTTPClientOpt(client *http.Client) Option {
	return func(api *API) {
		api.client = client
	}
}

func APITokenOpt(token string) Option {
	return func(api *API) {
		api.token = token
	}
}

func TimeoutOpt(timeout time.Duration) Option {
	return func(api *API) {
		api.timeout = timeout
	}
}
