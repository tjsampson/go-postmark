package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// TemplateModel is a dynamic key-value map used as the data model when
	// rendering a Postmark template.
	TemplateModel map[string]interface{}

	// SendEmailWithTemplateReq is the request body for sending a single email
	// using a Postmark template.
	SendEmailWithTemplateReq struct {
		TemplateId    int           `json:"TemplateId,omitempty"`
		TemplateAlias string        `json:"TemplateAlias,omitempty"`
		TemplateModel TemplateModel `json:"TemplateModel"`
		From          string        `json:"From"`
		To            string        `json:"To"`
		Cc            string        `json:"Cc,omitempty"`
		Bcc           string        `json:"Bcc,omitempty"`
		ReplyTo       string        `json:"ReplyTo,omitempty"`
		Tag           string        `json:"Tag,omitempty"`
		TrackOpens    bool          `json:"TrackOpens,omitempty"`
		TrackLinks    string        `json:"TrackLinks,omitempty"`
		MessageStream string        `json:"MessageStream,omitempty"`
		Attachments   []Attachment  `json:"Attachments,omitempty"`
		Headers       []Header      `json:"Headers,omitempty"`
		Metadata      interface{}   `json:"Metadata,omitempty"`
	}

	// BatchWithTemplatesReq is the request body for sending a batch of emails
	// using Postmark templates.
	BatchWithTemplatesReq struct {
		Messages []SendEmailWithTemplateReq `json:"Messages"`
	}

	// TemplateResp represents a Postmark template as returned by the API.
	TemplateResp struct {
		TemplateId         int    `json:"TemplateId"`
		Name               string `json:"Name"`
		Subject            string `json:"Subject"`
		HtmlBody           string `json:"HtmlBody"`
		TextBody           string `json:"TextBody"`
		Alias              string `json:"Alias"`
		LayoutTemplate     string `json:"LayoutTemplate"`
		AssociatedServerId int    `json:"AssociatedServerId"`
		Active             bool   `json:"Active"`
	}

	// ListTemplatesResp is the response envelope returned by the list-templates endpoint.
	ListTemplatesResp struct {
		TotalCount int            `json:"TotalCount"`
		Templates  []TemplateResp `json:"Templates"`
	}

	// CreateTemplateReq is the request body for creating a new Postmark template.
	CreateTemplateReq struct {
		Name           string `json:"Name"`
		Subject        string `json:"Subject"`
		HtmlBody       string `json:"HtmlBody"`
		TextBody       string `json:"TextBody"`
		Alias          string `json:"Alias,omitempty"`
		LayoutTemplate string `json:"LayoutTemplate,omitempty"`
	}

	// UpdateTemplateReq is the request body for updating an existing Postmark template.
	UpdateTemplateReq struct {
		Name           string `json:"Name,omitempty"`
		Subject        string `json:"Subject,omitempty"`
		HtmlBody       string `json:"HtmlBody,omitempty"`
		TextBody       string `json:"TextBody,omitempty"`
		Alias          string `json:"Alias,omitempty"`
		LayoutTemplate string `json:"LayoutTemplate,omitempty"`
	}

	// DeleteTemplateResp is the response returned when a template is deleted.
	DeleteTemplateResp struct {
		ErrorCode int    `json:"ErrorCode"`
		Message   string `json:"Message"`
	}
)

// SendEmailWithTemplate sends a single email using a Postmark template.
// Either TemplateId or TemplateAlias must be set in the request.
func (a *API) SendEmailWithTemplate(req *SendEmailWithTemplateReq) (*EmailResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "email/withTemplate", req)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data EmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// SendEmailBatchWithTemplates sends a batch of emails using Postmark templates.
// Each message in the batch may reference its own template.
func (a *API) SendEmailBatchWithTemplates(req *BatchWithTemplatesReq) (BatchEmailResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "email/batchWithTemplates", req)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data BatchEmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetTemplate fetches the Postmark template identified by templateIdOrAlias.
// The argument may be either a numeric template ID (as a string) or an alias.
func (a *API) GetTemplate(templateIdOrAlias string) (*TemplateResp, error) {
	httpReq, err := a.newRequest(http.MethodGet, fmt.Sprintf("templates/%s", templateIdOrAlias), nil)
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

// CreateTemplate creates a new Postmark template with the settings in req.
// It returns the full TemplateResp on success.
func (a *API) CreateTemplate(req *CreateTemplateReq) (*TemplateResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "templates", req)
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

// UpdateTemplate applies the changes in req to the Postmark template identified
// by templateIdOrAlias and returns the updated TemplateResp.
func (a *API) UpdateTemplate(templateIdOrAlias string, req *UpdateTemplateReq) (*TemplateResp, error) {
	httpReq, err := a.newRequest(http.MethodPut, fmt.Sprintf("templates/%s", templateIdOrAlias), req)
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

// ListTemplates returns a paginated list of all Postmark templates on the server.
// count controls the page size and offset controls the starting position.
func (a *API) ListTemplates(count, offset int) (*ListTemplatesResp, error) {
	params := url.Values{}
	params.Set("Count", strconv.Itoa(count))
	params.Set("Offset", strconv.Itoa(offset))
	httpReq, err := a.newRequest(http.MethodGet, "templates?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data ListTemplatesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteTemplate deletes the Postmark template identified by templateIdOrAlias.
// It returns a DeleteTemplateResp containing the outcome message from the API.
func (a *API) DeleteTemplate(templateIdOrAlias string) (*DeleteTemplateResp, error) {
	httpReq, err := a.newRequest(http.MethodDelete, fmt.Sprintf("templates/%s", templateIdOrAlias), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data DeleteTemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
