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
	// TemplateResp represents a Postmark Template as returned by the API.
	// When no alias or layout template is set, those fields decode to an empty
	// string (the API returns null, which Go decodes to the zero value).
	TemplateResp struct {
		TemplateID         int    `json:"TemplateId"`
		Name               string `json:"Name"`
		Subject            string `json:"Subject"`
		HtmlBody           string `json:"HtmlBody"`
		TextBody           string `json:"TextBody"`
		Alias              string `json:"Alias"`
		TemplateType       string `json:"TemplateType"`
		LayoutTemplate     string `json:"LayoutTemplate"`
		Active             bool   `json:"Active"`
		AssociatedServerID int    `json:"AssociatedServerId"`
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
	// When no alias or layout template is set, those fields decode to an empty
	// string (the API returns null, which Go decodes to the zero value).
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
	//
	// Exactly one of TemplateID or TemplateAlias must be set; the API will return
	// an error if neither is provided. No enforcement is performed client-side
	// beyond what the Postmark API itself rejects.
	//
	// TrackOpens and InlineCss are *bool so that callers can explicitly send
	// false (opting out) rather than having the zero value silently omitted.
	// A nil pointer omits the field entirely, letting the server apply its default.
	//
	// Metadata is typed as map[string]string to match the Postmark API contract
	// (key-value string pairs); this prevents accidentally passing incompatible types.
	SendWithTemplateReq struct {
		TemplateID    int               `json:"TemplateId,omitempty"`
		TemplateAlias string            `json:"TemplateAlias,omitempty"`
		TemplateModel interface{}       `json:"TemplateModel"`
		InlineCss     *bool             `json:"InlineCss,omitempty"`
		From          string            `json:"From"`
		To            string            `json:"To"`
		Cc            string            `json:"Cc,omitempty"`
		Bcc           string            `json:"Bcc,omitempty"`
		ReplyTo       string            `json:"ReplyTo,omitempty"`
		Tag           string            `json:"Tag,omitempty"`
		TrackOpens    *bool             `json:"TrackOpens,omitempty"`
		TrackLinks    string            `json:"TrackLinks,omitempty"`
		MessageStream string            `json:"MessageStream,omitempty"`
		Headers       []Header          `json:"Headers,omitempty"`
		Attachments   []Attachment      `json:"Attachments,omitempty"`
		Metadata      map[string]string `json:"Metadata,omitempty"`
	}

	// SendEmailResp is the response returned after sending an email with a template.
	// Postmark returns HTTP 200 even for logical failures; callers must check ErrorCode.
	SendEmailResp struct {
		To          string `json:"To"`
		SubmittedAt string `json:"SubmittedAt"`
		MessageID   string `json:"MessageID"`
		ErrorCode   int    `json:"ErrorCode"`
		Message     string `json:"Message"`
	}

	// ValidateTemplateReq is the request body for validating a template.
	//
	// InlineCssForHtmlTestRender is *bool rather than bool so that callers can
	// explicitly send false (disable CSS inlining for the test render) without
	// the zero value being silently omitted. A nil pointer omits the field
	// entirely, letting the server apply its default.
	ValidateTemplateReq struct {
		Subject                    string      `json:"Subject,omitempty"`
		HtmlBody                   string      `json:"HtmlBody,omitempty"`
		TextBody                   string      `json:"TextBody,omitempty"`
		TestRenderModel            interface{} `json:"TestRenderModel,omitempty"`
		TemplateType               string      `json:"TemplateType,omitempty"`
		LayoutTemplate             string      `json:"LayoutTemplate,omitempty"`
		InlineCssForHtmlTestRender *bool       `json:"InlineCssForHtmlTestRender,omitempty"`
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
	// SuggestedTemplateModel is typed as map[string]interface{} to reflect the
	// structured object shape described in the Postmark docs, rather than the
	// opaque interface{} that would force callers into unsafe type assertions.
	ValidateTemplateResp struct {
		AllContentIsValid      bool                     `json:"AllContentIsValid"`
		HtmlBody               TemplateValidationResult `json:"HtmlBody"`
		TextBody               TemplateValidationResult `json:"TextBody"`
		Subject                TemplateValidationResult `json:"Subject"`
		SuggestedTemplateModel map[string]interface{}   `json:"SuggestedTemplateModel"`
	}

	// PushTemplateReq is the request body for pushing templates between servers.
	// This endpoint requires an Account API token. This client sends the account
	// token on every request via the X-Postmark-Account-Token header (configured
	// through APITokenOpt or the POSTMARK_API_TOKEN environment variable).
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

// errEmptyTemplateID is returned when a caller passes an empty templateID.
var errEmptyTemplateID = errors.New("templateID must not be empty")

// BoolPtr is a convenience helper that returns a pointer to a bool literal,
// useful when constructing requests that have *bool fields inline.
//
//	req := &SendWithTemplateReq{TrackOpens: postmark.BoolPtr(false)}
func BoolPtr(b bool) *bool { return &b }

// SendEmailWithTemplate sends an email using a Postmark template.
// Requires a server API token. Use TemplateID or TemplateAlias (not both) to
// identify the template; passing a request with neither set will be rejected
// by the Postmark API.
//
// Postmark returns HTTP 200 even for logical failures; this method promotes a
// non-zero ErrorCode in the response body to a *PostmarkErr so callers can
// detect it with errors.As or errors.Is.
//
// req must not be nil.
func (a *API) SendEmailWithTemplate(req *SendWithTemplateReq) (*SendEmailResp, error) {
	if req == nil {
		return nil, errors.New("SendEmailWithTemplate: req must not be nil")
	}
	httpReq, err := a.newRequest(http.MethodPost, "email/withTemplate", req)
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
	if data.ErrorCode != 0 {
		return nil, &PostmarkErr{ErrorCode: data.ErrorCode, Message: data.Message}
	}
	return &data, nil
}

// GetTemplate fetches a single Postmark Template identified by templateID.
// templateID may be a numeric ID (passed as its decimal string, e.g. "123")
// or an alias string (e.g. "welcome-email"); it must not be empty.
func (a *API) GetTemplate(templateID string) (*TemplateResp, error) {
	if templateID == "" {
		return nil, errEmptyTemplateID
	}
	httpReq, err := a.newRequest(http.MethodGet, fmt.Sprintf("templates/%s", url.PathEscape(templateID)), nil)
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

// CreateTemplate creates a new Postmark Template with the settings in req.
// It returns the full TemplateResp on success.
// req must not be nil.
func (a *API) CreateTemplate(req *CreateTemplateReq) (*TemplateResp, error) {
	if req == nil {
		return nil, errors.New("CreateTemplate: req must not be nil")
	}
	httpReq, err := a.newRequest(http.MethodPost, "templates", req)
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

// EditTemplate applies the changes in req to the Postmark Template identified
// by templateID and returns the updated TemplateResp.
// templateID may be a numeric ID (passed as its decimal string, e.g. "123")
// or an alias string (e.g. "welcome-email"); it must not be empty.
// req must not be nil.
func (a *API) EditTemplate(templateID string, req *EditTemplateReq) (*TemplateResp, error) {
	if templateID == "" {
		return nil, errEmptyTemplateID
	}
	if req == nil {
		return nil, errors.New("EditTemplate: req must not be nil")
	}
	httpReq, err := a.newRequest(http.MethodPut, fmt.Sprintf("templates/%s", url.PathEscape(templateID)), req)
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

	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data ListTemplatesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteTemplate deletes the Postmark Template identified by templateID.
// It returns a DeleteResp containing the outcome message from the API.
// templateID may be a numeric ID (passed as its decimal string, e.g. "123")
// or an alias string (e.g. "welcome-email"); it must not be empty.
//
// DeleteResp (defined in servers.go) has ErrorCode and Message fields that
// match the Postmark Templates delete response shape exactly, so it is safe
// to reuse here.
func (a *API) DeleteTemplate(templateID string) (*DeleteResp, error) {
	if templateID == "" {
		return nil, errEmptyTemplateID
	}
	httpReq, err := a.newRequest(http.MethodDelete, fmt.Sprintf("templates/%s", url.PathEscape(templateID)), nil)
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

// ValidateTemplate validates the content of a template (HtmlBody, TextBody, Subject)
// against a test render model and returns validation results.
func (a *API) ValidateTemplate(req *ValidateTemplateReq) (*ValidateTemplateResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "templates/validate", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data ValidateTemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// PushTemplate pushes templates from a source server to a destination server.
// Set PerformChanges to true to apply the changes; false performs a dry run.
//
// This endpoint requires an Account API token, which this client sends on
// every request via the X-Postmark-Account-Token header (configured through
// APITokenOpt or the POSTMARK_API_TOKEN environment variable).
func (a *API) PushTemplate(req *PushTemplateReq) (*PushTemplateResp, error) {
	httpReq, err := a.newRequest(http.MethodPut, "templates/push", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data PushTemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
