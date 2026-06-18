package postmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// TemplateResp represents a Postmark Template as returned by the API.
	TemplateResp struct {
		TemplateID           int    `json:"TemplateId"`
		Name                 string `json:"Name"`
		Alias                string `json:"Alias"`
		Subject              string `json:"Subject"`
		HtmlBody             string `json:"HtmlBody"`
		TextBody             string `json:"TextBody"`
		AssociatedServerId   int    `json:"AssociatedServerId"`
		Active               bool   `json:"Active"`
		TemplateType         string `json:"TemplateType"`
		LayoutTemplate       string `json:"LayoutTemplate"`
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
	SendWithTemplateReq struct {
		TemplateID    int                    `json:"TemplateId,omitempty"`
		TemplateAlias string                 `json:"TemplateAlias,omitempty"`
		TemplateModel map[string]interface{} `json:"TemplateModel"`
		From          string                 `json:"From"`
		To            string                 `json:"To"`
		Cc            string                 `json:"Cc,omitempty"`
		Bcc           string                 `json:"Bcc,omitempty"`
		ReplyTo       string                 `json:"ReplyTo,omitempty"`
		Tag           string                 `json:"Tag,omitempty"`
		TrackOpens    bool                   `json:"TrackOpens,omitempty"`
		TrackLinks    string                 `json:"TrackLinks,omitempty"`
		MessageStream string                 `json:"MessageStream,omitempty"`
	}

	// BatchWithTemplatesReq is the request body for sending a batch of emails using templates.
	BatchWithTemplatesReq struct {
		Messages []SendWithTemplateReq `json:"Messages"`
	}

	// batchWithTemplatesResp is the internal response wrapper for batch template sends.
	batchWithTemplatesResp struct {
		TotalSent  int             `json:"TotalSent"`
		TotalFailed int            `json:"TotalFailed"`
		Responses  []SendEmailResp `json:"Responses"`
	}
)

// newServerTokenRequest builds an *http.Request that uses the
// X-Postmark-Server-Token header instead of the account token.
// This is required for the Templates and email-sending endpoints.
func (a *API) newServerTokenRequest(method, path string, body interface{}) (*http.Request, error) {
	var reqBody io.Reader = http.NoBody
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
	req.Header.Set("X-Postmark-Server-Token", a.token)

	return req, nil
}

// ListTemplates returns a paginated list of templates for the server.
// count controls the page size, offset controls the starting position, and
// layoutTemplate optionally filters by a layout template name/alias.
func (a *API) ListTemplates(count, offset int, layoutTemplate string) (*ListTemplatesResp, error) {
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
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
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
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data TemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetTemplate fetches the Postmark Template identified by idOrAlias.
func (a *API) GetTemplate(idOrAlias string) (*TemplateResp, error) {
	httpReq, err := a.newServerTokenRequest(http.MethodGet, fmt.Sprintf("templates/%s", idOrAlias), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
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
	httpReq, err := a.newServerTokenRequest(http.MethodPut, fmt.Sprintf("templates/%s", idOrAlias), req)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data TemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteTemplate deletes the Postmark Template identified by idOrAlias.
// It returns a DeleteResp containing the outcome message from the API.
func (a *API) DeleteTemplate(idOrAlias string) (*DeleteResp, error) {
	httpReq, err := a.newServerTokenRequest(http.MethodDelete, fmt.Sprintf("templates/%s", idOrAlias), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
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
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data SendEmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// SendEmailBatchWithTemplates sends a batch of emails each using a Postmark Template.
// Each message in req.Messages may reference a different template.
func (a *API) SendEmailBatchWithTemplates(req *BatchWithTemplatesReq) ([]SendEmailResp, error) {
	httpReq, err := a.newServerTokenRequest(http.MethodPost, "email/batchWithTemplates", req)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	// Postmark returns { "TotalSent": N, "TotalFailed": M, "Responses": [...] }
	var wrapper batchWithTemplatesResp
	if err = json.Unmarshal(resp.rawBody, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Responses, nil
}
