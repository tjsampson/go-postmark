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
	Doer interface {
		Do(req *http.Request) (*http.Response, error)
	}
	API struct {
		client   Doer
		timeout  time.Duration
		baseHost string
		token    string
	}
	Req struct {
		URI, Body string
	}
	Resp struct {
		rawBody []byte
		resp    *http.Response
	}
	Option func(*API)
)

var (
	defaultTimeOut    = time.Duration(10) * time.Second
	defaultHttpClient = &http.Client{Timeout: defaultTimeOut}
)

func New(options ...Option) *API {
	api := &API{
		baseHost: "https://api.postmarkapp.com",
		token:    os.Getenv("POSTMARK_API_TOKEN"),
		timeout:  defaultTimeOut,
		client:   defaultHttpClient,
	}

	// Apply Dynamic Caller Opts
	for _, opt := range options {
		opt(api)
	}

	return api
}

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

func (a *API) Do(req *http.Request) (*Resp, error) {
	var resp *http.Response

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	return readResponse(resp)
}

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

func newResponse(body []byte, resp *http.Response) *Resp {
	return &Resp{rawBody: body, resp: resp}
}
