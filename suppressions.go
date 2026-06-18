package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type (
	// SuppressionEntry represents a single suppressed email address.
	SuppressionEntry struct {
		EmailAddress      string     `json:"EmailAddress"`
		SuppressionReason string     `json:"SuppressionReason"`
		Origin            string     `json:"Origin"`
		CreatedAt         time.Time  `json:"CreatedAt"`
	}

	// SuppressionsParams holds optional query parameters for listing suppressions.
	SuppressionsParams struct {
		// SuppressionReason filters by reason: "HardBounce", "SpamComplaint",
		// "ManualSuppression".
		SuppressionReason string
		// Origin filters by origin: "Recipient", "Customer", "Admin".
		Origin string
		// EmailAddress filters by a specific email address.
		EmailAddress string
		// FromDate filters suppressions created on or after this date (YYYY-MM-DD).
		FromDate string
		// ToDate filters suppressions created on or before this date (YYYY-MM-DD).
		ToDate string
	}

	// SuppressionsResp is the response envelope returned by the suppressions
	// dump endpoint.
	SuppressionsResp struct {
		Suppressions []SuppressionEntry `json:"Suppressions"`
	}

	// SuppressionAddress is a single email address used in create/delete requests.
	SuppressionAddress struct {
		EmailAddress string `json:"EmailAddress"`
	}

	// CreateSuppressionReq is the request body for adding suppressions.
	CreateSuppressionReq struct {
		Suppressions []SuppressionAddress `json:"Suppressions"`
	}

	// DeleteSuppressionReq is the request body for removing suppressions.
	DeleteSuppressionReq struct {
		Suppressions []SuppressionAddress `json:"Suppressions"`
	}

	// SuppressionResult is the outcome of a single suppression create/delete
	// operation.
	SuppressionResult struct {
		EmailAddress string `json:"EmailAddress"`
		Status       string `json:"Status"`
		Message      string `json:"Message"`
	}

	// SuppressionResp is the response envelope returned by the create/delete
	// suppression endpoints.
	SuppressionResp struct {
		Suppressions []SuppressionResult `json:"Suppressions"`
	}
)

// ListSuppressions returns the suppression dump for the message stream
// identified by streamID. Use params to narrow results.
func (a *API) ListSuppressions(streamID string, params SuppressionsParams) (*SuppressionsResp, error) {
	qp := url.Values{}
	if params.SuppressionReason != "" {
		qp.Set("SuppressionReason", params.SuppressionReason)
	}
	if params.Origin != "" {
		qp.Set("Origin", params.Origin)
	}
	if params.EmailAddress != "" {
		qp.Set("EmailAddress", params.EmailAddress)
	}
	if params.FromDate != "" {
		qp.Set("FromDate", params.FromDate)
	}
	if params.ToDate != "" {
		qp.Set("ToDate", params.ToDate)
	}

	path := fmt.Sprintf("message-streams/%s/suppressions/dump", streamID)
	if len(qp) > 0 {
		path = path + "?" + qp.Encode()
	}

	req, err := a.newServerRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data SuppressionsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateSuppression adds one or more suppressions to the message stream
// identified by streamID.
func (a *API) CreateSuppression(streamID string, req *CreateSuppressionReq) (*SuppressionResp, error) {
	httpReq, err := a.newServerRequest(
		http.MethodPost,
		fmt.Sprintf("message-streams/%s/suppressions", streamID),
		req,
	)
	if err != nil {
		return nil, err
	}

	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data SuppressionResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteSuppression removes one or more suppressions from the message stream
// identified by streamID.
func (a *API) DeleteSuppression(streamID string, req *DeleteSuppressionReq) (*SuppressionResp, error) {
	httpReq, err := a.newServerRequest(
		http.MethodPost,
		fmt.Sprintf("message-streams/%s/suppressions/delete", streamID),
		req,
	)
	if err != nil {
		return nil, err
	}

	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data SuppressionResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
