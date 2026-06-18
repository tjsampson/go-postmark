package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type (
	// SuppressionResp represents a single suppression entry returned by the API.
	SuppressionResp struct {
		EmailAddress      string    `json:"EmailAddress"`
		SuppressionReason string    `json:"SuppressionReason"`
		Origin            string    `json:"Origin"`
		CreatedAt         time.Time `json:"CreatedAt"`
	}

	// ListSuppressionsParams holds optional query parameters for the list suppressions endpoint.
	ListSuppressionsParams struct {
		SuppressionReason string
		Origin            string
		EmailAddress      string
		FromDate          string
		ToDate            string
	}

	// ListSuppressionsResp is the response envelope returned by the list suppressions endpoint.
	ListSuppressionsResp struct {
		Suppressions []SuppressionResp `json:"Suppressions"`
	}

	// SuppressionEmail holds a single email address for create/delete suppression requests.
	SuppressionEmail struct {
		EmailAddress string `json:"EmailAddress"`
	}

	// CreateSuppressionsReq is the request body for creating suppressions.
	CreateSuppressionsReq struct {
		Suppressions []SuppressionEmail `json:"Suppressions"`
	}

	// SuppressionResult represents the per-email result in a create or delete suppressions response.
	SuppressionResult struct {
		EmailAddress string `json:"EmailAddress"`
		Status       string `json:"Status"`
		Message      string `json:"Message,omitempty"`
	}

	// CreateSuppressionsResp is the response envelope for the create suppressions endpoint.
	CreateSuppressionsResp struct {
		Suppressions []SuppressionResult `json:"Suppressions"`
	}

	// DeleteSuppressionsReq is the request body for deleting suppressions.
	DeleteSuppressionsReq struct {
		Suppressions []SuppressionEmail `json:"Suppressions"`
	}

	// DeleteSuppressionsResp is the response envelope for the delete suppressions endpoint.
	DeleteSuppressionsResp struct {
		Suppressions []SuppressionResult `json:"Suppressions"`
	}
)

// ListSuppressions returns the suppressions dump for the given message stream.
// An optional ListSuppressionsParams can be provided to filter results.
// Corresponds to GET /message-streams/{streamID}/suppressions/dump.
func (a *API) ListSuppressions(streamID string, params *ListSuppressionsParams) (*ListSuppressionsResp, error) {
	path := fmt.Sprintf("message-streams/%s/suppressions/dump", streamID)
	if params != nil {
		q := url.Values{}
		if params.SuppressionReason != "" {
			q.Set("SuppressionReason", params.SuppressionReason)
		}
		if params.Origin != "" {
			q.Set("Origin", params.Origin)
		}
		if params.EmailAddress != "" {
			q.Set("EmailAddress", params.EmailAddress)
		}
		if params.FromDate != "" {
			q.Set("FromDate", params.FromDate)
		}
		if params.ToDate != "" {
			q.Set("ToDate", params.ToDate)
		}
		if len(q) > 0 {
			path += "?" + q.Encode()
		}
	}

	httpReq, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data ListSuppressionsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateSuppressions adds one or more email addresses to the suppression list
// for the given message stream.
// Corresponds to POST /message-streams/{streamID}/suppressions.
func (a *API) CreateSuppressions(streamID string, body *CreateSuppressionsReq) (*CreateSuppressionsResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, fmt.Sprintf("message-streams/%s/suppressions", streamID), body)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data CreateSuppressionsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteSuppressions removes one or more email addresses from the suppression list
// for the given message stream.
// Corresponds to POST /message-streams/{streamID}/suppressions/delete.
func (a *API) DeleteSuppressions(streamID string, body *DeleteSuppressionsReq) (*DeleteSuppressionsResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, fmt.Sprintf("message-streams/%s/suppressions/delete", streamID), body)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data DeleteSuppressionsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
