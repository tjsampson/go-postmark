// Package postmark provides a Go client for the Postmark API,
// focused on administrative operations such as creating, reading,
// updating, listing, and deleting Postmark Servers.
package postmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type (
	// Doer is the interface satisfied by *http.Client and any custom HTTP transport.
	// It allows callers to inject a mock client for testing.
	Doer interface {
		Do(req *http.Request) (*http.Response, error)
	}

	// API is the main client for the Postmark API.
	// Create one with New() and supply functional options to configure it.
	API struct {
		client      Doer
		timeout     time.Duration
		baseHost    string
		token       string // account token, used in X-Postmark-Account-Token
		serverToken string // server token, used in X-Postmark-Server-Token; falls back to token when empty
		timeoutSet  bool   // true when TimeoutOpt was explicitly supplied
	}

	// Req holds the URI and optional JSON body string for an outgoing request.
	Req struct {
		URI, Body string
	}

	// Resp wraps the raw response body bytes and the underlying *http.Response.
	Resp struct {
		rawBody []byte
		resp    *http.Response
	}

	// Option is a functional option for configuring an API client.
	Option func(*API)
)

var (
	defaultTimeOut = time.Duration(10) * time.Second
)

// New creates and returns a new Postmark API client.
// By default it reads the API token from the POSTMARK_API_TOKEN environment
// variable and uses a 10-second timeout. Pass Option values to override
// any of those defaults.
func New(options ...Option) *API {
	api := &API{
		baseHost: "https://api.postmarkapp.com",
		token:    os.Getenv("POSTMARK_API_TOKEN"),
		timeout:  defaultTimeOut,
		client:   &http.Client{Timeout: defaultTimeOut},
	}

	// Apply Dynamic Caller Opts
	for _, opt := range options {
		opt(api)
	}

	// Reconcile: if the caller explicitly supplied TimeoutOpt and the underlying
	// client is an *http.Client, propagate the final timeout to it. This ensures
	// the timeout is consistent regardless of option order, without silently
	// overwriting a timeout set on a caller-owned client that was injected without
	// a corresponding TimeoutOpt.
	if api.timeoutSet {
		if hc, ok := api.client.(*http.Client); ok {
			hc.Timeout = api.timeout
		}
	}

	return api
}

// newRequestWithAuthHeader is the single low-level request constructor.
// It builds an *http.Request for method+path, JSON-encodes body when non-nil,
// and sets exactly one authentication header — headerName: headerValue.
// All other request construction (newRequest, newServerTokenRequest) delegates
// here so that auth-header selection is centralised and not duplicated.
func (a *API) newRequestWithAuthHeader(method, path, headerName, headerValue string, body interface{}) (*http.Request, error) {
	var reqBody io.Reader = http.NoBody
	hasBody := body != nil
	if hasBody {
		reqPayload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(reqPayload)
	}

	req, err := http.NewRequest(
		method,
		fmt.Sprintf("%s/%s", a.baseHost, path),
		reqBody)

	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(headerName, headerValue)

	return req, nil
}

// newRequest builds an *http.Request that carries X-Postmark-Account-Token.
// It is used for account-scoped endpoints (servers, etc.).
func (a *API) newRequest(method, path string, body interface{}) (*http.Request, error) {
	return a.newRequestWithAuthHeader(method, path, "X-Postmark-Account-Token", a.token, body)
}

// newServerTokenRequest builds an *http.Request that carries
// X-Postmark-Server-Token. Bounce and Delivery Stats endpoints require a
// server-scoped credential. It uses effectiveServerToken() to select the token.
func (a *API) newServerTokenRequest(method, path string, body interface{}) (*http.Request, error) {
	return a.newRequestWithAuthHeader(method, path, "X-Postmark-Server-Token", a.effectiveServerToken(), body)
}

// effectiveServerToken returns the token that should be sent in
// X-Postmark-Server-Token. When ServerTokenOpt has been supplied it returns
// that value. Otherwise it falls back to the account token (set via
// APITokenOpt or the POSTMARK_API_TOKEN environment variable).
//
// Note: the fallback exists as a convenience for callers who use a single
// token for both account- and server-scoped operations (e.g. during testing or
// in simple single-server setups where the same token is accepted). Production
// deployments that enforce a strict separation between account and server tokens
// should always supply ServerTokenOpt — without it the account token will be
// sent on server-scoped requests, which Postmark will reject with a 401.
func (a *API) effectiveServerToken() string {
	if a.serverToken != "" {
		return a.serverToken
	}
	return a.token
}

// Do executes an HTTP request and returns the wrapped response.
// It delegates to the underlying Doer (usually *http.Client).
func (a *API) Do(req *http.Request) (*Resp, error) {
	var resp *http.Response

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	return readResponse(resp)
}

// readResponse reads the body from an *http.Response and returns a *Resp.
// For non-2xx status codes it attempts to unmarshal a PostmarkErr and returns
// the appropriate sentinel error so callers can use errors.Is:
//   - A 404 response returns ErrNotFound.
//   - A 409 response returns ErrExists.
//   - Any other non-2xx response returns the unmarshalled PostmarkErr value.
func readResponse(resp *http.Response) (*Resp, error) {
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return newResponse(respBody, resp), nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return newResponse(respBody, resp), ErrNotFound
	}
	if resp.StatusCode == http.StatusConflict {
		return newResponse(respBody, resp), ErrExists
	}

	var pmError PostmarkErr
	err = json.Unmarshal(respBody, &pmError)
	if err != nil {
		return newResponse(respBody, resp), fmt.Errorf("failed to unmarshal postmark err: %w", err)
	}

	return newResponse(respBody, resp), pmError
}

// newResponse is a helper constructor for *Resp.
func newResponse(body []byte, resp *http.Response) *Resp {
	return &Resp{rawBody: body, resp: resp}
}
