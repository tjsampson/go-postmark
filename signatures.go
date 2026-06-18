package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// SignatureDetails contains the full details of a sender signature.
	SignatureDetails struct {
		ID                         int    `json:"ID"`
		Domain                     string `json:"Domain"`
		EmailAddress               string `json:"EmailAddress"`
		ReplyToEmailAddress        string `json:"ReplyToEmailAddress"`
		Name                       string `json:"Name"`
		Confirmed                  bool   `json:"Confirmed"`
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
		ConfirmationPersonalNote   string `json:"ConfirmationPersonalNote"`
	}

	// SignatureListEntry is the abbreviated entry returned in list responses.
	SignatureListEntry struct {
		ID           int    `json:"ID"`
		Domain       string `json:"Domain"`
		EmailAddress string `json:"EmailAddress"`
		Name         string `json:"Name"`
		Confirmed    bool   `json:"Confirmed"`
	}

	// ListSignaturesResp is the response envelope returned by the list-senders endpoint.
	ListSignaturesResp struct {
		TotalCount       int                  `json:"TotalCount"`
		SenderSignatures []SignatureListEntry  `json:"SenderSignatures"`
	}

	// CreateSignatureReq is the request body for creating a new sender signature.
	CreateSignatureReq struct {
		FromEmail                string `json:"FromEmail"`
		Name                     string `json:"Name"`
		ReplyToEmail             string `json:"ReplyToEmail,omitempty"`
		ReturnPathDomain         string `json:"ReturnPathDomain,omitempty"`
		ConfirmationPersonalNote string `json:"ConfirmationPersonalNote,omitempty"`
	}

	// EditSignatureReq is the request body for editing an existing sender signature.
	EditSignatureReq struct {
		Name                     string `json:"Name,omitempty"`
		ReplyToEmail             string `json:"ReplyToEmail,omitempty"`
		ReturnPathDomain         string `json:"ReturnPathDomain,omitempty"`
		ConfirmationPersonalNote string `json:"ConfirmationPersonalNote,omitempty"`
	}

	// ResendResp is the response returned by the resend-confirmation endpoint.
	// Postmark returns {"ErrorCode": 0, "Message": "..."} for this call.
	ResendResp struct {
		ErrorCode int    `json:"ErrorCode"`
		Message   string `json:"Message"`
	}
)

// SignatureResp is the full sender signature response returned by
// create/get/edit/rotate operations. It is an alias for SignatureDetails so
// callers access fields directly without an extra embedding layer.
type SignatureResp = SignatureDetails

// ListSignatures returns a paginated list of sender signatures on the account.
// count controls the page size and offset controls the starting position.
func (a *API) ListSignatures(count, offset int) (*ListSignaturesResp, error) {
	params := url.Values{}
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
	req, err := a.newRequest(http.MethodGet, "senders?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ListSignaturesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetSignature fetches the sender signature identified by sigID.
func (a *API) GetSignature(sigID int) (*SignatureResp, error) {
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("senders/%d", sigID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data SignatureResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateSignature creates a new sender signature with the settings in sigReq.
func (a *API) CreateSignature(sigReq *CreateSignatureReq) (*SignatureResp, error) {
	req, err := a.newRequest(http.MethodPost, "senders", sigReq)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data SignatureResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// EditSignature applies the changes in editReq to the sender signature identified by sigID.
func (a *API) EditSignature(sigID int, editReq *EditSignatureReq) (*SignatureResp, error) {
	req, err := a.newRequest(http.MethodPut, fmt.Sprintf("senders/%d", sigID), editReq)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data SignatureResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteSignature deletes the sender signature identified by sigID.
func (a *API) DeleteSignature(sigID int) (*DeleteResp, error) {
	req, err := a.newRequest(http.MethodDelete, fmt.Sprintf("senders/%d", sigID), nil)
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

// ResendSignatureConfirmation resends the confirmation email for the sender
// signature identified by sigID.
func (a *API) ResendSignatureConfirmation(sigID int) (*ResendResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("senders/%d/resend", sigID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ResendResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// RotateSignatureDKIM rotates the DKIM key for the sender signature identified by sigID.
func (a *API) RotateSignatureDKIM(sigID int) (*SignatureResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("senders/%d/rotateDkim", sigID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data SignatureResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
