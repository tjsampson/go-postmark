package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// BounceResp represents a single bounce record returned by the Postmark API.
	BounceResp struct {
		ID          int64  `json:"ID"`
		Type        string `json:"Type"`
		TypeCode    int    `json:"TypeCode"`
		Name        string `json:"Name"`
		Tag         string `json:"Tag"`
		MessageID   string `json:"MessageID"`
		ServerID    int    `json:"ServerID"`
		Description string `json:"Description"`
		Details     string `json:"Details"`
		Email       string `json:"Email"`
		From        string `json:"From"`
		// BouncedAt is the RFC 3339 timestamp at which the bounce occurred,
		// stored as a string to preserve the exact format returned by the API.
		// Callers that need time arithmetic should parse it with time.Parse(time.RFC3339, …).
		BouncedAt     string `json:"BouncedAt"`
		DumpAvailable bool   `json:"DumpAvailable"`
		Inactive      bool   `json:"Inactive"`
		CanActivate   bool   `json:"CanActivate"`
		Subject       string `json:"Subject"`
	}

	// ListBouncesParams holds the optional query parameters for the ListBounces
	// endpoint. Pointer fields (Inactive, Count, Offset) distinguish "not
	// provided" from an explicit zero/false value, which matters because
	// Postmark treats omitted and zero-valued parameters differently.
	ListBouncesParams struct {
		Type          string
		Inactive      *bool  // nil = omit; &false = send inactive=false; &true = send inactive=true
		EmailFilter   string
		Tag           string
		MessageID     string
		FromDate      string
		ToDate        string
		Count         *int // nil = omit; pointer to 0 sends count=0
		Offset        *int // nil = omit; pointer to 0 sends offset=0 (first page)
		MessageStream string
	}

	// ListBouncesResp is the response envelope returned by the list-bounces endpoint.
	ListBouncesResp struct {
		TotalCount int          `json:"TotalCount"`
		Bounces    []BounceResp `json:"Bounces"`
	}

	// BounceDumpResp is the response returned by the get-bounce-dump endpoint.
	BounceDumpResp struct {
		Body string `json:"Body"`
	}

	// ActivateBounceResp is the response returned by the activate-bounce endpoint.
	ActivateBounceResp struct {
		Message string     `json:"Message"`
		Bounce  BounceResp `json:"Bounce"`
	}

	// BounceCount represents a named count of bounces by type, as returned by
	// the delivery stats endpoint.
	BounceCount struct {
		Name  string `json:"Name"`
		Count int    `json:"Count"`
		Type  string `json:"Type"`
	}

	// DeliveryStatsResp is the response returned by the get-delivery-stats endpoint.
	DeliveryStatsResp struct {
		InactiveMails int           `json:"InactiveMails"`
		Bounces       []BounceCount `json:"Bounces"`
	}
)

// serverToken returns the token that should be sent in X-Postmark-Server-Token.
// It uses the explicitly configured server token when available, and falls back
// to the account token so that callers who only supply APITokenOpt still work.
func (a *API) effectiveServerToken() string {
	if a.serverToken != "" {
		return a.serverToken
	}
	return a.token
}

// newServerTokenRequest builds an *http.Request that carries
// X-Postmark-Server-Token instead of X-Postmark-Account-Token.
// Bounce and Delivery Stats endpoints require a server-scoped token.
func (a *API) newServerTokenRequest(method, path string, body interface{}) (*http.Request, error) {
	req, err := a.newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	// Replace the account-token header with the server token.
	req.Header.Del("X-Postmark-Account-Token")
	req.Header.Set("X-Postmark-Server-Token", a.effectiveServerToken())
	return req, nil
}

// ListBounces retrieves a paginated list of bounces matching the supplied
// parameters. Nil pointer fields in params are omitted from the query string;
// use a pointer to the zero value to send an explicit 0 or false.
func (a *API) ListBounces(params ListBouncesParams) (*ListBouncesResp, error) {
	q := url.Values{}
	if params.Type != "" {
		q.Set("type", params.Type)
	}
	if params.Inactive != nil {
		if *params.Inactive {
			q.Set("inactive", "true")
		} else {
			q.Set("inactive", "false")
		}
	}
	if params.EmailFilter != "" {
		q.Set("emailFilter", params.EmailFilter)
	}
	if params.Tag != "" {
		q.Set("tag", params.Tag)
	}
	if params.MessageID != "" {
		q.Set("messageID", params.MessageID)
	}
	if params.FromDate != "" {
		q.Set("fromdate", params.FromDate)
	}
	if params.ToDate != "" {
		q.Set("todate", params.ToDate)
	}
	if params.Count != nil {
		q.Set("count", strconv.Itoa(*params.Count))
	}
	if params.Offset != nil {
		q.Set("offset", strconv.Itoa(*params.Offset))
	}
	if params.MessageStream != "" {
		q.Set("messagestream", params.MessageStream)
	}

	u := url.URL{Path: "bounces", RawQuery: q.Encode()}
	req, err := a.newServerTokenRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data ListBouncesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetBounce fetches the bounce record identified by bounceID.
func (a *API) GetBounce(bounceID int64) (*BounceResp, error) {
	req, err := a.newServerTokenRequest(http.MethodGet, fmt.Sprintf("bounces/%d", bounceID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data BounceResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetBounceDump retrieves the raw SMTP dump for the bounce identified by bounceID.
func (a *API) GetBounceDump(bounceID int64) (*BounceDumpResp, error) {
	req, err := a.newServerTokenRequest(http.MethodGet, fmt.Sprintf("bounces/%d/dump", bounceID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data BounceDumpResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ActivateBounce reactivates the bounce identified by bounceID so that future
// messages to the same address are no longer suppressed.
func (a *API) ActivateBounce(bounceID int64) (*ActivateBounceResp, error) {
	req, err := a.newServerTokenRequest(http.MethodPut, fmt.Sprintf("bounces/%d/activate", bounceID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data ActivateBounceResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetDeliveryStats returns overall delivery statistics including the number of
// inactive mails and a breakdown of bounces by type.
func (a *API) GetDeliveryStats() (*DeliveryStatsResp, error) {
	req, err := a.newServerTokenRequest(http.MethodGet, "deliverystats", nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data DeliveryStatsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
