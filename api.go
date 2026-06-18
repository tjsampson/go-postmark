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

	"github.com/pkg/errors"
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
		client   Doer
		timeout  time.Duration
		baseHost string
		token    string
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

	return api
}

// newRequest builds an *http.Request for the given HTTP method and API path.
// If body is non-nil it is JSON-encoded as the request body.
func (a *API) newRequest(method, path string, body interface{}) (*http.Request, error) {
	var reqPayload []byte
	if body != nil {
		var err error
		reqPayload, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(
		method,
		fmt.Sprintf("%s/%s", a.baseHost, path),
		bytes.NewReader(reqPayload))

	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Postmark-Account-Token", a.token)

	return req, nil
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
// For non-2xx / non-404 status codes it attempts to unmarshal a PostmarkErr.
func readResponse(resp *http.Response) (*Resp, error) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return newResponse(respBody, resp), nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return newResponse(respBody, resp), nil
	}

	var pmError PostmarkErr
	err = json.Unmarshal(respBody, &pmError)
	if err != nil {
		return newResponse(respBody, resp), errors.Wrap(err, "failed to unmarshal postmark err")
	}

	return newResponse(respBody, resp), pmError
}

// newResponse is a helper constructor for *Resp.
func newResponse(body []byte, resp *http.Response) *Resp {
	return &Resp{rawBody: body, resp: resp}
}
