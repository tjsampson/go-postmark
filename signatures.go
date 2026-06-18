package postmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// SenderSignatureResp represents a Postmark Sender Signature as returned by the API.
	SenderSignatureResp struct {
		ID                            int    `json:"ID"`
		Domain                        string `json:"Domain"`
		EmailAddress                  string `json:"EmailAddress"`
		ReplyToEmailAddress           string `json:"ReplyToEmailAddress"`
		Name                          string `json:"Name"`
		Confirmed                     bool   `json:"Confirmed"`
		SPFVerified                   bool   `json:"SPFVerified"`
		SPFHost                       string `json:"SPFHost"`
		SPFTextValue                  string `json:"SPFTextValue"`
		DKIMVerified                  bool   `json:"DKIMVerified"`
		WeakDKIM                      bool   `json:"WeakDKIM"`
		DKIMHost                      string `json:"DKIMHost"`
		DKIMTextValue                 string `json:"DKIMTextValue"`
		DKIMPendingHost               string `json:"DKIMPendingHost"`
		DKIMPendingTextValue          string `json:"DKIMPendingTextValue"`
		DKIMRevokedHost               string `json:"DKIMRevokedHost"`
		DKIMRevokedTextValue          string `json:"DKIMRevokedTextValue"`
		SafeToRemoveRevokedKeyFromDNS bool   `json:"SafeToRemoveRevokedKeyFromDNS"`
		DKIMUpdateStatus              string `json:"DKIMUpdateStatus"`
		ReturnPathDomain              string `json:"ReturnPathDomain"`
		ReturnPathDomainVerified      bool   `json:"ReturnPathDomainVerified"`
		ReturnPathDomainCNAMEValue    string `json:"ReturnPathDomainCNAMEValue"`
	}

	// ListSenderSignaturesResp is the response envelope returned by the
	// list sender signatures endpoint.
	ListSenderSignaturesResp struct {
		TotalCount       int                  `json:"TotalCount"`
		SenderSignatures []SenderSignatureResp `json:"SenderSignatures"`
	}

	// CreateSenderSignatureReq is the request body for creating a new Sender Signature.
	CreateSenderSignatureReq struct {
		FromEmail        string `json:"FromEmail"`
		Name             string `json:"Name"`
		ReplyToEmail     string `json:"ReplyToEmail,omitempty"`
		ReturnPathDomain string `json:"ReturnPathDomain,omitempty"`
	}

	// UpdateSenderSignatureReq is the request body for updating an existing Sender Signature.
	UpdateSenderSignatureReq struct {
		Name             string `json:"Name"`
		ReplyToEmail     string `json:"ReplyToEmail,omitempty"`
		ReturnPathDomain string `json:"ReturnPathDomain,omitempty"`
	}
)

// ListSenderSignatures returns a paginated list of all Sender Signatures on the account.
// count controls the page size and offset controls the starting position.
// count must be positive.
func (a *API) ListSenderSignatures(count, offset int) (*ListSenderSignaturesResp, error) {
	if count <= 0 {
		return nil, errors.New("postmark: count must be positive")
	}
	params := url.Values{}
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
	req, err := a.newRequest(http.MethodGet, "senders?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}
	var data ListSenderSignaturesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateSenderSignature creates a new Sender Signature with the settings in req.
// It returns the full SenderSignatureResp on success.
func (a *API) CreateSenderSignature(req *CreateSenderSignatureReq) (*SenderSignatureResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "senders", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}
	var data SenderSignatureResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetSenderSignature fetches the Sender Signature identified by signatureID.
// signatureID must be positive.
func (a *API) GetSenderSignature(signatureID int64) (*SenderSignatureResp, error) {
	if signatureID <= 0 {
		return nil, errors.New("postmark: signatureID must be positive")
	}
	httpReq, err := a.newRequest(http.MethodGet, fmt.Sprintf("senders/%d", signatureID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}
	var data SenderSignatureResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateSenderSignature applies the changes in req to the Sender Signature
// identified by signatureID and returns the updated SenderSignatureResp.
// signatureID must be positive.
func (a *API) UpdateSenderSignature(signatureID int64, req *UpdateSenderSignatureReq) (*SenderSignatureResp, error) {
	if signatureID <= 0 {
		return nil, errors.New("postmark: signatureID must be positive")
	}
	httpReq, err := a.newRequest(http.MethodPut, fmt.Sprintf("senders/%d", signatureID), req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}
	var data SenderSignatureResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteSenderSignature deletes the Sender Signature identified by signatureID.
// It returns a DeleteResp containing the outcome message from the API.
// signatureID must be positive.
func (a *API) DeleteSenderSignature(signatureID int64) (*DeleteResp, error) {
	if signatureID <= 0 {
		return nil, errors.New("postmark: signatureID must be positive")
	}
	httpReq, err := a.newRequest(http.MethodDelete, fmt.Sprintf("senders/%d", signatureID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}
	var data DeleteResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ResendSenderSignatureConfirmation resends the confirmation email for the
// Sender Signature identified by signatureID.
// signatureID must be positive.
func (a *API) ResendSenderSignatureConfirmation(signatureID int64) (*DeleteResp, error) {
	if signatureID <= 0 {
		return nil, errors.New("postmark: signatureID must be positive")
	}
	httpReq, err := a.newRequest(http.MethodPost, fmt.Sprintf("senders/%d/resend", signatureID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}
	var data DeleteResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// VerifySenderSignatureSPF requests SPF verification for the Sender Signature
// identified by signatureID.
// signatureID must be positive.
func (a *API) VerifySenderSignatureSPF(signatureID int64) (*SenderSignatureResp, error) {
	if signatureID <= 0 {
		return nil, errors.New("postmark: signatureID must be positive")
	}
	httpReq, err := a.newRequest(http.MethodPost, fmt.Sprintf("senders/%d/verifyspf", signatureID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}
	var data SenderSignatureResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// RequestNewDKIMForSenderSignature requests a new DKIM key for the Sender
// Signature identified by signatureID.
// signatureID must be positive.
func (a *API) RequestNewDKIMForSenderSignature(signatureID int64) (*SenderSignatureResp, error) {
	if signatureID <= 0 {
		return nil, errors.New("postmark: signatureID must be positive")
	}
	httpReq, err := a.newRequest(http.MethodPost, fmt.Sprintf("senders/%d/requestnewdkim", signatureID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}
	var data SenderSignatureResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
