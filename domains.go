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
		ID                           int    `json:"ID"`
		Name                         string `json:"Name"`
		SPFVerified                  bool   `json:"SPFVerified"`
		DKIMVerified                 bool   `json:"DKIMVerified"`
		WeakDKIM                     bool   `json:"WeakDKIM"`
		ReturnPathDomain             string `json:"ReturnPathDomain"`
		ReturnPathDomainVerified     bool   `json:"ReturnPathDomainVerified"`
		DKIMHost                     string `json:"DKIMHost"`
		DKIMTextValue                string `json:"DKIMTextValue"`
		DKIMPendingHost              string `json:"DKIMPendingHost"`
		DKIMPendingTextValue         string `json:"DKIMPendingTextValue"`
		DKIMRevokedHost              string `json:"DKIMRevokedHost"`
		DKIMRevokedTextValue         string `json:"DKIMRevokedTextValue"`
		SafeToRemoveRevokedKeyFromDNS bool   `json:"SafeToRemoveRevokedKeyFromDNS"`
		DKIMUpdateStatus             string `json:"DKIMUpdateStatus"`
		SafeDomain                   bool   `json:"SafeDomain"`
	}

	// CreateDomainReq is the request body for creating a new Postmark Domain.
	CreateDomainReq struct {
		Name             string `json:"Name"`
		ReturnPathDomain string `json:"ReturnPathDomain,omitempty"`
	}

	// UpdateDomainReq is the request body for updating an existing Postmark Domain.
	UpdateDomainReq struct {
		ReturnPathDomain string `json:"ReturnPathDomain,omitempty"`
	}

	// ListDomainsResp is the response envelope returned by the list-domains endpoint.
	ListDomainsResp struct {
		TotalCount int          `json:"TotalCount"`
		Domains    []DomainResp `json:"Domains"`
	}
)

// ListDomains returns a paginated list of all Postmark Domains on the account.
// count controls the page size and offset controls the starting position.
func (a *API) ListDomains(count, offset int) (*ListDomainsResp, error) {
	params := url.Values{}
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
	req, err := a.newRequest(http.MethodGet, "domains?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
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
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
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
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateDomain applies the changes in req to the Postmark Domain identified
// by domainID and returns the updated DomainResp.
func (a *API) UpdateDomain(domainID int, req *UpdateDomainReq) (*DomainResp, error) {
	httpReq, err := a.newRequest(http.MethodPut, fmt.Sprintf("domains/%d", domainID), req)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
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
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data DeleteResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// VerifyDomainDkim triggers DKIM verification for the Postmark Domain
// identified by domainID.
func (a *API) VerifyDomainDkim(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPut, fmt.Sprintf("domains/%d/verifyDkim", domainID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
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
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// VerifyDomainSPF triggers SPF verification for the Postmark Domain
// identified by domainID.
func (a *API) VerifyDomainSPF(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("domains/%d/verifyspf", domainID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// RotateDomainDKIM initiates a DKIM key rotation for the Postmark Domain
// identified by domainID.
func (a *API) RotateDomainDKIM(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("domains/%d/rotatedkim", domainID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data DomainResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
