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
		ID            int64  `json:"ID"`
		Type          string `json:"Type"`
		TypeCode      int    `json:"TypeCode"`
		Name          string `json:"Name"`
		Tag           string `json:"Tag"`
		MessageID     string `json:"MessageID"`
		ServerID      int    `json:"ServerID"`
		Description   string `json:"Description"`
		Details       string `json:"Details"`
		Email         string `json:"Email"`
		From          string `json:"From"`
		BouncedAt     string `json:"BouncedAt"`
		DumpAvailable bool   `json:"DumpAvailable"`
		Inactive      bool   `json:"Inactive"`
		CanActivate   bool   `json:"CanActivate"`
		Subject       string `json:"Subject"`
	}

	// ListBouncesParams holds the optional query parameters for the ListBounces endpoint.
	ListBouncesParams struct {
		Type          string
		Inactive      bool
		EmailFilter   string
		Tag           string
		MessageID     string
		FromDate      string
		ToDate        string
		Count         int
		Offset        int
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
		InactiveMails int          `json:"InactiveMails"`
		Bounces       []BounceCount `json:"Bounces"`
	}
)

// newServerTokenRequest builds an *http.Request that uses the
// X-Postmark-Server-Token header instead of X-Postmark-Account-Token.
// The Bounce and Delivery Stats APIs require a server token.
func (a *API) newServerTokenRequest(method, path string, body interface{}) (*http.Request, error) {
	req, err := a.newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	// Swap the account token header for the server token header.
	req.Header.Del("X-Postmark-Account-Token")
	req.Header.Set("X-Postmark-Server-Token", a.token)
	return req, nil
}

// ListBounces retrieves a paginated list of bounces matching the supplied
// parameters. Any zero-value field in params is omitted from the query string.
func (a *API) ListBounces(params ListBouncesParams) (*ListBouncesResp, error) {
	q := url.Values{}
	if params.Type != "" {
		q.Set("type", params.Type)
	}
	if params.Inactive {
		q.Set("inactive", "true")
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
	if params.Count > 0 {
		q.Set("count", strconv.Itoa(params.Count))
	}
	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.MessageStream != "" {
		q.Set("messagestream", params.MessageStream)
	}

	path := "bounces"
	if len(q) > 0 {
		path = "bounces?" + q.Encode()
	}

	req, err := a.newServerTokenRequest(http.MethodGet, path, nil)
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
