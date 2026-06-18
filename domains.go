package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// DomainDetails contains the full DNS and configuration details of a domain.
	DomainDetails struct {
		ID                         int    `json:"ID"`
		Name                       string `json:"Name"`
		SPFVerified                bool   `json:"SPFVerified"`
		SPFHost                    string `json:"SPFHost"`
		SPFTextValue               string `json:"SPFTextValue"`
		DKIMVerified               bool   `json:"DKIMVerified"`
		WeakDKIM                   bool   `json:"WeakDKIM"`
		DKIMHost                   string `json:"DKIMHost"`
		DKIMTextValue              string `json:"DKIMTextValue"`
		DKIMPendingHost            string `json:"DKIMPendingHost"`
		DKIMPendingTextValue       string `json:"DKIMPendingTextValue"`
		DKIMRevokedHost            string `json:"DKIMRevokedHost"`
		DKIMRevokedTextValue       string `json:"DKIMRevokedTextValue"`
		SafeToRemoveRevokedKey     bool   `json:"SafeToRemoveRevokedKey"`
		DKIMUpdateStatus           string `json:"DKIMUpdateStatus"`
		ReturnPathDomain           string `json:"ReturnPathDomain"`
		ReturnPathDomainCNAMEValue string `json:"ReturnPathDomainCNAMEValue"`
		ReturnPathDomainVerified   bool   `json:"ReturnPathDomainVerified"`
	}

	// DomainListEntry is the abbreviated domain entry returned in list responses.
	DomainListEntry struct {
		ID                       int    `json:"ID"`
		Name                     string `json:"Name"`
		SPFVerified              bool   `json:"SPFVerified"`
		DKIMVerified             bool   `json:"DKIMVerified"`
		ReturnPathDomainVerified bool   `json:"ReturnPathDomainVerified"`
	}

	// ListDomainsResp is the response envelope returned by the list-domains endpoint.
	ListDomainsResp struct {
		TotalCount int               `json:"TotalCount"`
		Domains    []DomainListEntry `json:"Domains"`
	}

	// CreateDomainReq is the request body for creating a new domain.
	CreateDomainReq struct {
		Name             string `json:"Name"`
		ReturnPathDomain string `json:"ReturnPathDomain,omitempty"`
	}

	// EditDomainReq is the request body for editing an existing domain.
	EditDomainReq struct {
		ReturnPathDomain string `json:"ReturnPathDomain,omitempty"`
	}
)

// DomainResp is the full domain response returned by create/get/edit/verify
// operations. It is an alias for DomainDetails so callers access fields
// directly without an extra embedding layer.
type DomainResp = DomainDetails

// ListDomains returns a paginated list of domains on the account.
// count controls the page size and offset controls the starting position.
func (a *API) ListDomains(count, offset int) (*ListDomainsResp, error) {
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

// GetDomain fetches the domain identified by domainID.
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

// CreateDomain creates a new domain with the settings in domainReq.
func (a *API) CreateDomain(domainReq *CreateDomainReq) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPost, "domains", domainReq)
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

// EditDomain applies the changes in editReq to the domain identified by domainID.
func (a *API) EditDomain(domainID int, editReq *EditDomainReq) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPut, fmt.Sprintf("domains/%d", domainID), editReq)
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

// DeleteDomain deletes the domain identified by domainID.
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

// VerifyDomainDKIM triggers DKIM verification for the domain identified by domainID.
func (a *API) VerifyDomainDKIM(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("domains/%d/verifyDkim", domainID), nil)
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

// VerifyDomainReturnPath triggers Return-Path verification for the domain identified by domainID.
func (a *API) VerifyDomainReturnPath(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("domains/%d/verifyReturnPath", domainID), nil)
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

// RotateDomainDKIM rotates the DKIM key for the domain identified by domainID.
func (a *API) RotateDomainDKIM(domainID int) (*DomainResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("domains/%d/rotateDkim", domainID), nil)
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
