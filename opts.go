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

// APITokenOpt returns an Option that sets the Postmark API token used by this
// client. The same token value is sent as X-Postmark-Account-Token for
// account-level endpoints (e.g. server management) and as
// X-Postmark-Server-Token for server-level endpoints (e.g. the Bounce API).
//
// Postmark distinguishes two token types:
//   - Account token: found in "Account Settings → API Tokens". Used for
//     account-level operations (creating/listing servers, etc.).
//   - Server token: found in "Server → API Credentials". Used for
//     server-level operations (sending email, bounce API, etc.).
//
// Supply the appropriate token for the operations your application performs.
// Using an account token with server-level endpoints (or vice-versa) will
// result in 401 Unauthorized responses from the Postmark API.
func APITokenOpt(token string) Option {
	return func(api *API) {
		api.token = token
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
