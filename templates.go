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
	// rendering a Postmark template. Note: an empty (non-nil) map will be
	// serialised as "TemplateModel":{} and sent to the API. Set the field to
	// nil (or leave it unset) to omit it entirely from the request body.
	TemplateModel map[string]interface{}

	// SendEmailWithTemplateReq is the request body for sending a single email
	// using a Postmark template.
	SendEmailWithTemplateReq struct {
		// TemplateID is the numeric ID of the template to use.
		// Either TemplateID or TemplateAlias must be set.
		TemplateID    int    `json:"TemplateId,omitempty"`
		TemplateAlias string `json:"TemplateAlias,omitempty"`
		// TemplateModel is the data model used to render the template.
		// omitempty omits a nil map rather than sending "TemplateModel":null.
		// An empty (non-nil) map is sent as "TemplateModel":{}.
		TemplateModel TemplateModel `json:"TemplateModel,omitempty"`
		From          string        `json:"From"`
		To            string        `json:"To"`
		Cc            string        `json:"Cc,omitempty"`
		Bcc           string        `json:"Bcc,omitempty"`
		ReplyTo       string        `json:"ReplyTo,omitempty"`
		Tag           string        `json:"Tag,omitempty"`
		// TrackOpens uses *bool so that an explicit false is serialised correctly.
		// omitempty on a plain bool would silently drop false.
		TrackOpens    *bool        `json:"TrackOpens,omitempty"`
		TrackLinks    string       `json:"TrackLinks,omitempty"`
		MessageStream string       `json:"MessageStream,omitempty"`
		Attachments   []Attachment `json:"Attachments,omitempty"`
		Headers       []Header     `json:"Headers,omitempty"`
		Metadata      interface{}  `json:"Metadata,omitempty"`
	}

	// BatchWithTemplatesReq is the request body for sending a batch of emails
	// using Postmark templates.
	BatchWithTemplatesReq struct {
		Messages []SendEmailWithTemplateReq `json:"Messages"`
	}

	// batchWithTemplatesResp is the internal response envelope for the
	// POST /email/batchWithTemplates endpoint. Postmark returns
	// {"Messages":[...]} rather than a bare JSON array.
	batchWithTemplatesResp struct {
		Messages []EmailResp `json:"Messages"`
	}

	// TemplateResp represents a Postmark template as returned by the API.
	TemplateResp struct {
		// TemplateID is the numeric ID of the template.
		TemplateID         int    `json:"TemplateId"`
		Name               string `json:"Name"`
		Subject            string `json:"Subject"`
		HtmlBody           string `json:"HtmlBody"`
		TextBody           string `json:"TextBody"`
		Alias              string `json:"Alias"`
		LayoutTemplate     string `json:"LayoutTemplate"`
		AssociatedServerId int    `json:"AssociatedServerId"`
		Active             bool   `json:"Active"`
		ErrorCode          int    `json:"ErrorCode"`
		Message            string `json:"Message"`
	}

	// ListTemplatesResp is the response envelope returned by the list-templates endpoint.
	ListTemplatesResp struct {
		TotalCount int            `json:"TotalCount"`
		Templates  []TemplateResp `json:"Templates"`
	}

	// CreateTemplateReq is the request body for creating a new Postmark template.
	// All fields except Alias and LayoutTemplate are required on create.
	CreateTemplateReq struct {
		Name           string `json:"Name"`
		Subject        string `json:"Subject"`
		HtmlBody       string `json:"HtmlBody"`
		TextBody       string `json:"TextBody"`
		Alias          string `json:"Alias,omitempty"`
		LayoutTemplate string `json:"LayoutTemplate,omitempty"`
	}

	// UpdateTemplateReq is the request body for updating an existing Postmark template.
	// All fields are optional; only supplied fields are changed.
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
// Either TemplateID or TemplateAlias must be set in the request.
func (a *API) SendEmailWithTemplate(req *SendEmailWithTemplateReq) (*EmailResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "email/withTemplate", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data EmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	if data.ErrorCode != 0 {
		return nil, PostmarkErr{ErrorCode: data.ErrorCode, Message: data.Message}
	}
	return &data, nil
}

// SendEmailBatchWithTemplates sends a batch of emails using Postmark templates.
// Each message in the batch may reference its own template.
// The Postmark API returns {"Messages":[...]} which is unwrapped before returning.
// Note: Postmark reports per-message delivery failures via the ErrorCode field on
// each EmailResp element rather than as a top-level error. Callers must inspect
// each element's ErrorCode to detect individual message failures.
func (a *API) SendEmailBatchWithTemplates(req *BatchWithTemplatesReq) (BatchEmailResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "email/batchWithTemplates", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	// The API returns {"Messages":[...]} — unmarshal the envelope then return
	// the inner slice so callers get a flat BatchEmailResp.
	var envelope batchWithTemplatesResp
	if err = json.Unmarshal(resp.rawBody, &envelope); err != nil {
		return nil, err
	}
	return BatchEmailResp(envelope.Messages), nil
}

// GetTemplate fetches the Postmark template identified by templateIDOrAlias.
// The argument may be either a numeric template ID (as a string) or an alias.
func (a *API) GetTemplate(templateIDOrAlias string) (*TemplateResp, error) {
	httpReq, err := a.newRequest(
		http.MethodGet,
		fmt.Sprintf("templates/%s", url.PathEscape(templateIDOrAlias)),
		nil,
	)
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
	if data.ErrorCode != 0 {
		return nil, PostmarkErr{ErrorCode: data.ErrorCode, Message: data.Message}
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
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data TemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	if data.ErrorCode != 0 {
		return nil, PostmarkErr{ErrorCode: data.ErrorCode, Message: data.Message}
	}
	return &data, nil
}

// UpdateTemplate applies the changes in req to the Postmark template identified
// by templateIDOrAlias and returns the updated TemplateResp.
func (a *API) UpdateTemplate(templateIDOrAlias string, req *UpdateTemplateReq) (*TemplateResp, error) {
	httpReq, err := a.newRequest(
		http.MethodPut,
		fmt.Sprintf("templates/%s", url.PathEscape(templateIDOrAlias)),
		req,
	)
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
	if data.ErrorCode != 0 {
		return nil, PostmarkErr{ErrorCode: data.ErrorCode, Message: data.Message}
	}
	return &data, nil
}

// ListTemplates returns a paginated list of all Postmark templates on the server.
// count controls the page size and offset controls the starting position.
func (a *API) ListTemplates(count, offset int) (*ListTemplatesResp, error) {
	httpReq, err := a.newRequest(http.MethodGet, "templates", nil)
	if err != nil {
		return nil, err
	}
	// Set the query string on the already-parsed URL so the parameters are
	// never double-encoded (unlike manually appending "?...Encode()" to the path).
	q := url.Values{}
	q.Set("Count", strconv.Itoa(count))
	q.Set("Offset", strconv.Itoa(offset))
	httpReq.URL.RawQuery = q.Encode()

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

// DeleteTemplate deletes the Postmark template identified by templateIDOrAlias.
// It returns a DeleteTemplateResp containing the outcome message from the API.
func (a *API) DeleteTemplate(templateIDOrAlias string) (*DeleteTemplateResp, error) {
	httpReq, err := a.newRequest(
		http.MethodDelete,
		fmt.Sprintf("templates/%s", url.PathEscape(templateIDOrAlias)),
		nil,
	)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data DeleteTemplateResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	if data.ErrorCode != 0 {
		return nil, PostmarkErr{ErrorCode: data.ErrorCode, Message: data.Message}
	}
	return &data, nil
}
