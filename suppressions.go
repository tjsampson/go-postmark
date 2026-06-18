package postmark

import (
	"encoding/json"
	"errors"
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

	// ListSuppressionsParams holds optional query parameters for the list suppressions
	// endpoint. Each field maps directly to the query parameter of the same name.
	ListSuppressionsParams struct {
		SuppressionReason string
		Origin            string
		// EmailAddress may contain characters such as '+' that are special in URLs;
		// they are always encoded via url.Values.Encode before being appended to the path.
		EmailAddress string
		FromDate     string
		ToDate       string
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

	// SuppressionResult represents the per-email result in a create or delete
	// suppressions response. A single type is used for both operations because the
	// Postmark API returns identical fields for each (verified against the Postmark
	// Suppressions API documentation as of June 2025 — revisit if the API diverges).
	// Create responses use Status values "Suppressed"/"Failed"; delete responses use
	// "Deleted"/"Failed". Both share the same EmailAddress and optional Message fields.
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

// errEmptyStreamID is returned by any suppression method that receives an
// empty streamID, which would otherwise silently produce a malformed URL.
var errEmptyStreamID = errors.New("postmark: streamID must not be empty")

// ListSuppressions returns the suppressions dump for the given message stream.
// An optional ListSuppressionsParams can be provided to filter results.
// Corresponds to GET /message-streams/{streamID}/suppressions/dump.
func (a *API) ListSuppressions(streamID string, params *ListSuppressionsParams) (*ListSuppressionsResp, error) {
	if streamID == "" {
		return nil, errEmptyStreamID
	}
	path := fmt.Sprintf("message-streams/%s/suppressions/dump", url.PathEscape(streamID))
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
	if streamID == "" {
		return nil, errEmptyStreamID
	}
	httpReq, err := a.newRequest(
		http.MethodPost,
		fmt.Sprintf("message-streams/%s/suppressions", url.PathEscape(streamID)),
		body,
	)
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
	if streamID == "" {
		return nil, errEmptyStreamID
	}
	httpReq, err := a.newRequest(
		http.MethodPost,
		fmt.Sprintf("message-streams/%s/suppressions/delete", url.PathEscape(streamID)),
		body,
	)
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
