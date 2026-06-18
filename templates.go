package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// TemplateResp represents a Postmark Template as returned by the API.
	TemplateResp struct {
		TemplateID     int    `json:"TemplateId"`
		Name           string `json:"Name"`
		Subject        string `json:"Subject"`
		HtmlBody       string `json:"HtmlBody"`
		TextBody       string `json:"TextBody"`
		Alias          string `json:"Alias"`
		TemplateType   string `json:"TemplateType"`
		LayoutTemplate string `json:"LayoutTemplate"`
		Active         bool   `json:"Active"`
		AssociatedServerId int `json:"AssociatedServerId"`
	}

	// CreateTemplateReq is the request body for creating a new Postmark Template.
	CreateTemplateReq struct {
		Name           string `json:"Name"`
		Subject        string `json:"Subject"`
		HtmlBody       string `json:"HtmlBody,omitempty"`
		TextBody       string `json:"TextBody,omitempty"`
		Alias          string `json:"Alias,omitempty"`
		TemplateType   string `json:"TemplateType,omitempty"`
		LayoutTemplate string `json:"LayoutTemplate,omitempty"`
	}

	// EditTemplateReq is the request body for updating an existing Postmark Template.
	EditTemplateReq struct {
		Name           string `json:"Name,omitempty"`
		Subject        string `json:"Subject,omitempty"`
		HtmlBody       string `json:"HtmlBody,omitempty"`
		TextBody       string `json:"TextBody,omitempty"`
		Alias          string `json:"Alias,omitempty"`
		TemplateType   string `json:"TemplateType,omitempty"`
		LayoutTemplate string `json:"LayoutTemplate,omitempty"`
	}

	// TemplateListItem is a summary entry returned by ListTemplates.
	TemplateListItem struct {
		Active         bool   `json:"Active"`
		Alias          string `json:"Alias"`
		Name           string `json:"Name"`
		TemplateID     int    `json:"TemplateId"`
		TemplateType   string `json:"TemplateType"`
		LayoutTemplate string `json:"LayoutTemplate"`
	}

	// ListTemplatesResp is the response envelope returned by the list-templates endpoint.
	ListTemplatesResp struct {
		TotalCount int                `json:"TotalCount"`
		Templates  []TemplateListItem `json:"Templates"`
	}

	// Header represents an email header key-value pair.
	Header struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}

	// Attachment represents an email attachment.
	Attachment struct {
		Name        string `json:"Name"`
		Content     string `json:"Content"`
		ContentType string `json:"ContentType"`
		ContentID   string `json:"ContentID,omitempty"`
	}

	// SendWithTemplateReq is the request body for sending an email using a template.
	SendWithTemplateReq struct {
		TemplateID    int         `json:"TemplateId,omitempty"`
		TemplateAlias string      `json:"TemplateAlias,omitempty"`
		TemplateModel interface{} `json:"TemplateModel"`
		InlineCss     bool        `json:"InlineCss"`
		From          string      `json:"From"`
		To            string      `json:"To"`
		Cc            string      `json:"Cc,omitempty"`
		Bcc           string      `json:"Bcc,omitempty"`
		ReplyTo       string      `json:"ReplyTo,omitempty"`
		Tag           string      `json:"Tag,omitempty"`
		TrackOpens    bool        `json:"TrackOpens"`
		TrackLinks    string      `json:"TrackLinks,omitempty"`
		MessageStream string      `json:"MessageStream,omitempty"`
		Headers       []Header    `json:"Headers,omitempty"`
		Attachments   []Attachment `json:"Attachments,omitempty"`
		Metadata      interface{} `json:"Metadata,omitempty"`
	}

	// SendEmailResp is the response returned after sending an email.
	SendEmailResp struct {
		To          string `json:"To"`
		SubmittedAt string `json:"SubmittedAt"`
		MessageID   string `json:"MessageID"`
		ErrorCode   int    `json:"ErrorCode"`
		Message     string `json:"Message"`
	}

	// ValidateTemplateReq is the request body for validating a template.
	ValidateTemplateReq struct {
		Subject                string      `json:"Subject,omitempty"`
		HtmlBody               string      `json:"HtmlBody,omitempty"`
		TextBody               string      `json:"TextBody,omitempty"`
		TestRenderModel        interface{} `json:"TestRenderModel,omitempty"`
		TemplateType           string      `json:"TemplateType,omitempty"`
		LayoutTemplate         string      `json:"LayoutTemplate,omitempty"`
		InlineCssForHtmlTestRender bool    `json:"InlineCssForHtmlTestRender"`
	}

	// TemplateValidationError represents a single validation error in a template field.
	TemplateValidationError struct {
		Message           string `json:"Message"`
		Line              int    `json:"Line"`
		CharacterPosition int    `json:"CharacterPosition"`
	}

	// TemplateValidationResult holds the validation outcome for a single template field.
	TemplateValidationResult struct {
		ContentIsValid   bool                      `json:"ContentIsValid"`
		ValidationErrors []TemplateValidationError `json:"ValidationErrors"`
		RenderedContent  string                    `json:"RenderedContent"`
	}

	// ValidateTemplateResp is the response from the template validation endpoint.
	ValidateTemplateResp struct {
		AllContentIsValid      bool                     `json:"AllContentIsValid"`
		HtmlBody               TemplateValidationResult `json:"HtmlBody"`
		TextBody               TemplateValidationResult `json:"TextBody"`
		Subject                TemplateValidationResult `json:"Subject"`
		SuggestedTemplateModel interface{}              `json:"SuggestedTemplateModel"`
	}

	// PushTemplateReq is the request body for pushing a template between servers.
	PushTemplateReq struct {
		SourceServerID      int  `json:"SourceServerId"`
		DestinationServerID int  `json:"DestinationServerId"`
		PerformChanges      bool `json:"PerformChanges"`
	}

	// PushTemplateChange describes a single template push change.
	PushTemplateChange struct {
		Action     string `json:"Action"`
		TemplateID int    `json:"TemplateId"`
		Alias      string `json:"Alias"`
		Name       string `json:"Name"`
	}

	// PushTemplateResp is the response from the push template endpoint.
	PushTemplateResp struct {
		TotalCount int                  `json:"TotalCount"`
		Templates  []PushTemplateChange `json:"Templates"`
	}
)

// SendEmailWithTemplate sends an email using a Postmark template.
// Requires a server API token. Use TemplateID or TemplateAlias to identify the template.
func (a *API) SendEmailWithTemplate(req *SendWithTemplateReq) (*SendEmailResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "email/withTemplate", req)
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

// GetTemplate fetches a single Postmark Template identified by templateID.
// templateID may be a numeric ID or an alias string.
func (a *API) GetTemplate(templateID string) (*TemplateResp, error) {
	httpReq, err := a.newRequest(http.MethodGet, fmt.Sprintf("templates/%s", templateID), nil)
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

// CreateTemplate creates a new Postmark Template with the settings in req.
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

// EditTemplate applies the changes in req to the Postmark Template identified
// by templateID and returns the updated TemplateResp.
// templateID may be a numeric ID or an alias string.
func (a *API) EditTemplate(templateID string, req *EditTemplateReq) (*TemplateResp, error) {
	httpReq, err := a.newRequest(http.MethodPut, fmt.Sprintf("templates/%s", templateID), req)
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

// ListTemplates returns a paginated list of all Postmark Templates on the server.
// count controls the page size and offset controls the starting position.
func (a *API) ListTemplates(count, offset int) (*ListTemplatesResp, error) {
	params := url.Values{}
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
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

// DeleteTemplate deletes the Postmark Template identified by templateID.
// It returns a DeleteResp containing the outcome message from the API.
// templateID may be a numeric ID or an alias string.
func (a *API) DeleteTemplate(templateID string) (*DeleteResp, error) {
	httpReq, err := a.newRequest(http.MethodDelete, fmt.Sprintf("templates/%s", templateID), nil)
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

// ValidateTemplate validates the content of a template (HtmlBody, TextBody, Subject)
// against a test render model and returns validation results.
func (a *API) ValidateTemplate(req *ValidateTemplateReq) (*ValidateTemplateResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "templates/validate", req)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data ValidateTemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// PushTemplate pushes templates from a source server to a destination server.
// Set PerformChanges to true to apply the changes; false performs a dry run.
func (a *API) PushTemplate(req *PushTemplateReq) (*PushTemplateResp, error) {
	httpReq, err := a.newRequest(http.MethodPut, "templates/push", req)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data PushTemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
