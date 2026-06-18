package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// DomainResp represents a Postmark Domain as returned by the API.
	DomainResp struct {
		ID                            int    `json:"ID"`
		Name                          string `json:"Name"`
		SPFVerified                   bool   `json:"SPFVerified"`
		DKIMVerified                  bool   `json:"DKIMVerified"`
		WeakDKIM                      bool   `json:"WeakDKIM"`
		ReturnPathDomain              string `json:"ReturnPathDomain"`
		ReturnPathDomainVerified      bool   `json:"ReturnPathDomainVerified"`
		DKIMHost                      string `json:"DKIMHost"`
		DKIMTextValue                 string `json:"DKIMTextValue"`
		DKIMPendingHost               string `json:"DKIMPendingHost"`
		DKIMPendingTextValue          string `json:"DKIMPendingTextValue"`
		DKIMRevokedHost               string `json:"DKIMRevokedHost"`
		DKIMRevokedTextValue          string `json:"DKIMRevokedTextValue"`
		SafeToRemoveRevokedKeyFromDNS bool   `json:"SafeToRemoveRevokedKeyFromDNS"`
		DKIMUpdateStatus              string `json:"DKIMUpdateStatus"`
		SafeDomain                    bool   `json:"SafeDomain"`
	}

	// CreateDomainReq is the request body for creating a new Postmark Domain.
	CreateDomainReq struct {
		Name             string `json:"Name"`
		ReturnPathDomain string `json:"ReturnPathDomain,omitempty"`
	}

	// UpdateDomainReq is the request body for updating an existing Postmark Domain.
	// All fields are optional; fields left at their zero value are omitted from
	// the JSON body via omitempty.  Callers must set at least one field;
	// UpdateDomain returns an error if the struct is nil or equals its zero value.
	UpdateDomainReq struct {
		ReturnPathDomain string `json:"ReturnPathDomain,omitempty"`
	}

	// ListDomainsResp is the response envelope returned by the list-domains endpoint.
	ListDomainsResp struct {
		TotalCount int          `json:"TotalCount"`
		Domains    []DomainResp `json:"Domains"`
	}

	// emptyBody is a named sentinel type used as the JSON body for POST endpoints
	// that require a Content-Type: application/json header but accept no input
	// fields (e.g. verifyspf, rotatedkim).  Using a named type rather than an
	// anonymous struct{}{}  makes the intent explicit at each call site.
	emptyBody struct{}
)

// ListDomains returns a paginated list of all Postmark Domains on the account.
// count controls the page size and offset controls the starting position.
// count must be at least 1; the Postmark API rejects count=0.
func (a *API) ListDomains(count, offset int) (*ListDomainsResp, error) {
	if count < 1 {
		return nil, fmt.Errorf("postmark: ListDomains count must be at least 1, got %d", count)
	}
	req, err := a.newRequest(http.MethodGet, "domains", nil)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = params.Encode()

	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data ListDomainsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetDomain fetches the Postmark Domain identified by domainID.
func (a *API) GetDomain(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("domains/%d", domainID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateDomain creates a new Postmark Domain with the settings in req.
// It returns the full DomainResp on success.
func (a *API) CreateDomain(req *CreateDomainReq) (*DomainResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "domains", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateDomain applies the changes in req to the Postmark Domain identified
// by domainID and returns the updated DomainResp.
// It returns an error immediately if req is nil or entirely zero-valued, since
// submitting an empty JSON object would be a silent no-op.
func (a *API) UpdateDomain(domainID int, req *UpdateDomainReq) (*DomainResp, error) {
	if req == nil || *req == (UpdateDomainReq{}) {
		return nil, fmt.Errorf("postmark: UpdateDomainReq has no fields to update")
	}
	httpReq, err := a.newRequest(http.MethodPut, fmt.Sprintf("domains/%d", domainID), req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteDomain deletes the Postmark Domain identified by domainID.
// It returns a DeleteResp containing the outcome message from the API.
func (a *API) DeleteDomain(domainID int) (*DeleteResp, error) {
	req, err := a.newRequest(http.MethodDelete, fmt.Sprintf("domains/%d", domainID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data DeleteResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// VerifyDomainDkim triggers DKIM verification for the Postmark Domain
// identified by domainID.
// Note: the path segment "verifyDkim" is camelCase as specified by the
// Postmark API, unlike the all-lowercase "verifyspf" and "rotatedkim" paths
// used by other endpoints in this file — the inconsistency originates from
// Postmark's own API surface and is reproduced faithfully here.
func (a *API) VerifyDomainDkim(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPut, fmt.Sprintf("domains/%d/verifyDkim", domainID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// VerifyDomainReturnPath triggers Return-Path verification for the Postmark
// Domain identified by domainID.
func (a *API) VerifyDomainReturnPath(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPut, fmt.Sprintf("domains/%d/verifyReturnPath", domainID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// VerifyDomainSPF triggers SPF verification for the Postmark Domain
// identified by domainID.  An emptyBody value is sent so that
// Content-Type: application/json is set on the POST request, as required by
// the Postmark API for POST endpoints (see newRequest: Content-Type is only
// set when body != nil).
func (a *API) VerifyDomainSPF(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("domains/%d/verifyspf", domainID), emptyBody{})
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// RotateDomainDKIM initiates a DKIM key rotation for the Postmark Domain
// identified by domainID.  An emptyBody value is sent so that
// Content-Type: application/json is set on the POST request, as required by
// the Postmark API for POST endpoints (see newRequest: Content-Type is only
// set when body != nil).
func (a *API) RotateDomainDKIM(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("domains/%d/rotatedkim", domainID), emptyBody{})
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
