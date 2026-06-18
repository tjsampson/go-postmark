package postmark

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// TemplateResp represents a Postmark Template as returned by the API.
	TemplateResp struct {
		TemplateID         int    `json:"TemplateId"`
		Name               string `json:"Name"`
		Alias              string `json:"Alias"`
		Subject            string `json:"Subject"`
		HtmlBody           string `json:"HtmlBody"`
		TextBody           string `json:"TextBody"`
		AssociatedServerId int    `json:"AssociatedServerId"`
		Active             bool   `json:"Active"`
		TemplateType       string `json:"TemplateType"`
		LayoutTemplate     string `json:"LayoutTemplate"`
	}

	// CreateTemplateReq is the request body for creating a new Postmark Template.
	CreateTemplateReq struct {
		Name           string `json:"Name"`
		Alias          string `json:"Alias,omitempty"`
		Subject        string `json:"Subject"`
		HtmlBody       string `json:"HtmlBody,omitempty"`
		TextBody       string `json:"TextBody,omitempty"`
		TemplateType   string `json:"TemplateType,omitempty"`
		LayoutTemplate string `json:"LayoutTemplate,omitempty"`
	}

	// UpdateTemplateReq is the request body for updating an existing Postmark Template.
	// Only the fields provided will be changed.
	UpdateTemplateReq struct {
		Name           string `json:"Name,omitempty"`
		Alias          string `json:"Alias,omitempty"`
		Subject        string `json:"Subject,omitempty"`
		HtmlBody       string `json:"HtmlBody,omitempty"`
		TextBody       string `json:"TextBody,omitempty"`
		TemplateType   string `json:"TemplateType,omitempty"`
		LayoutTemplate string `json:"LayoutTemplate,omitempty"`
	}

	// ListTemplatesResp is the response envelope returned by the list-templates endpoint.
	ListTemplatesResp struct {
		TotalCount int            `json:"TotalCount"`
		Templates  []TemplateResp `json:"Templates"`
	}

	// SendEmailResp represents the response from sending an email via Postmark.
	SendEmailResp struct {
		To          string `json:"To"`
		SubmittedAt string `json:"SubmittedAt"`
		MessageID   string `json:"MessageID"`
		ErrorCode   int    `json:"ErrorCode"`
		Message     string `json:"Message"`
	}

	// SendWithTemplateReq is the request body for sending an email using a template.
	// Exactly one of TemplateID (non-zero) or TemplateAlias (non-empty) must be set.
	//
	// TrackOpens is a *bool so that an explicit false value is serialised to the
	// JSON body (rather than omitted), honouring the caller's intent to disable
	// open tracking. A nil pointer omits the field, deferring to the Postmark
	// server-side default.
	//
	// TrackLinks valid values per the Postmark API spec: "None", "HtmlAndText",
	// "HtmlOnly", "TextOnly". An empty string omits the field.
	//
	// TemplateModel is omitted from the JSON body when nil (omitempty). Callers
	// that want to explicitly send an empty model should pass an initialised but
	// empty map[string]interface{}{}.
	SendWithTemplateReq struct {
		TemplateID    int                    `json:"TemplateId,omitempty"`
		TemplateAlias string                 `json:"TemplateAlias,omitempty"`
		TemplateModel map[string]interface{} `json:"TemplateModel,omitempty"`
		From          string                 `json:"From"`
		To            string                 `json:"To"`
		Cc            string                 `json:"Cc,omitempty"`
		Bcc           string                 `json:"Bcc,omitempty"`
		ReplyTo       string                 `json:"ReplyTo,omitempty"`
		Tag           string                 `json:"Tag,omitempty"`
		TrackOpens    *bool                  `json:"TrackOpens,omitempty"`
		TrackLinks    string                 `json:"TrackLinks,omitempty"`
		MessageStream string                 `json:"MessageStream,omitempty"`
	}

	// BatchWithTemplatesReq is the request body for sending a batch of emails using templates.
	BatchWithTemplatesReq struct {
		Messages []SendWithTemplateReq `json:"Messages"`
	}

	// batchWithTemplatesAPIResp is an unexported type used only to decode the
	// Postmark batch-send response envelope:
	//
	//	{ "TotalSent": N, "TotalFailed": M, "Messages": [...] }
	//
	// All fields are exported within the struct so that standard encoding/json
	// marshaling works without any custom MarshalJSON/UnmarshalJSON logic. The
	// type itself stays unexported because callers only ever receive the
	// []SendEmailResp slice; the aggregate counts are decoded but not currently
	// surfaced in the public API (they exist for future use and for test helpers
	// that need to construct the full Postmark envelope shape).
	batchWithTemplatesAPIResp struct {
		TotalSent   int             `json:"TotalSent"`
		TotalFailed int             `json:"TotalFailed"`
		Messages    []SendEmailResp `json:"Messages"`
	}
)

// newServerTokenRequest builds an *http.Request that authenticates with the
// X-Postmark-Server-Token header. This is required for the Templates and
// email-sending endpoints, which are scoped to an individual server rather than
// the account as a whole. The server token is kept in a.serverToken, separate
// from the account token in a.token, so that a single API instance can issue
// both account-level and server-level requests with the correct credentials.
//
// It returns an error immediately if a.serverToken is empty, because an empty
// token will always produce a 401 Unauthorized from Postmark with no
// actionable message.
func (a *API) newServerTokenRequest(method, path string, body interface{}) (*http.Request, error) {
	if a.serverToken == "" {
		return nil, errors.New("postmark: server token is empty; set POSTMARK_SERVER_TOKEN or use ServerTokenOpt")
	}

	// For GET/DELETE we pass a nil io.Reader to http.NewRequest so that
	// req.Body is nil rather than http.NoBody. This avoids a spurious
	// Content-Length: 0 header on bodyless requests that some proxies or
	// middleware may mishandle. For requests with a body we use a concrete
	// *bytes.Reader.
	var reqBody io.Reader // nil by default (correct for GET/DELETE)
	hasBody := body != nil
	if hasBody {
		reqPayload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(reqPayload)
	}

	req, err := http.NewRequest(
		method,
		fmt.Sprintf("%s/%s", a.baseHost, path),
		reqBody,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Postmark-Server-Token", a.serverToken)

	return req, nil
}

// ListTemplates returns a paginated list of templates for the server.
// count must be ≥ 1 and offset must be ≥ 0; the function returns a local error
// for invalid values rather than letting the remote API produce a cryptic 422.
// layoutTemplate optionally filters by a layout template name/alias.
func (a *API) ListTemplates(count, offset int, layoutTemplate string) (*ListTemplatesResp, error) {
	if count < 1 {
		return nil, errors.New("postmark: ListTemplates count must be >= 1")
	}
	if offset < 0 {
		return nil, errors.New("postmark: ListTemplates offset must be >= 0")
	}

	params := url.Values{}
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
	if layoutTemplate != "" {
		params.Set("layoutTemplate", layoutTemplate)
	}

	req, err := a.newServerTokenRequest(http.MethodGet, "templates?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data ListTemplatesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateTemplate creates a new Postmark Template with the settings in req.
// It returns the full TemplateResp on success.
func (a *API) CreateTemplate(req *CreateTemplateReq) (*TemplateResp, error) {
	httpReq, err := a.newServerTokenRequest(http.MethodPost, "templates", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data TemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetTemplate fetches the Postmark Template identified by idOrAlias.
func (a *API) GetTemplate(idOrAlias string) (*TemplateResp, error) {
	httpReq, err := a.newServerTokenRequest(http.MethodGet, "templates/"+url.PathEscape(idOrAlias), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data TemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateTemplate applies the changes in req to the Postmark Template identified
// by idOrAlias and returns the updated TemplateResp.
func (a *API) UpdateTemplate(idOrAlias string, req *UpdateTemplateReq) (*TemplateResp, error) {
	httpReq, err := a.newServerTokenRequest(http.MethodPut, "templates/"+url.PathEscape(idOrAlias), req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data TemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteTemplate deletes the Postmark Template identified by idOrAlias.
// It returns a DeleteResp (defined in servers.go) containing the outcome
// message from the API.
func (a *API) DeleteTemplate(idOrAlias string) (*DeleteResp, error) {
	httpReq, err := a.newServerTokenRequest(http.MethodDelete, "templates/"+url.PathEscape(idOrAlias), nil)
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

// SendEmailWithTemplate sends a single email using a Postmark Template.
// Specify either TemplateID (numeric) or TemplateAlias in req.
func (a *API) SendEmailWithTemplate(req *SendWithTemplateReq) (*SendEmailResp, error) {
	httpReq, err := a.newServerTokenRequest(http.MethodPost, "email/withTemplate", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data SendEmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// SendEmailBatchWithTemplates sends a batch of emails each using a Postmark Template.
// Each message in req.Messages may reference a different template.
//
// Postmark returns a { "TotalSent": N, "TotalFailed": M, "Messages": [...] }
// envelope. The per-message slice is returned directly; callers should inspect
// each element's ErrorCode to detect per-message failures. If the batch-level
// call itself fails (non-2xx HTTP status) the error is returned in the second
// return value.
func (a *API) SendEmailBatchWithTemplates(req *BatchWithTemplatesReq) ([]SendEmailResp, error) {
	httpReq, err := a.newServerTokenRequest(http.MethodPost, "email/batchWithTemplates", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var wrapper batchWithTemplatesAPIResp
	if err = json.Unmarshal(resp.rawBody, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Messages, nil
}
