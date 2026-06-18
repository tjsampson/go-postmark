package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type (
	// MailHeader represents a custom email header as a name/value pair.
	MailHeader struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}

	// Attachment represents a file attachment on an outgoing email.
	Attachment struct {
		Name        string `json:"Name"`
		Content     string `json:"Content"`
		ContentType string `json:"ContentType"`
		ContentID   string `json:"ContentID,omitempty"`
	}

	// SendEmailReq is the request body for sending a single email via POST /email.
	// All fields correspond directly to the Postmark Email API documented at
	// https://postmarkapp.com/developer/api/email-api.
	SendEmailReq struct {
		From          string            `json:"From"`
		To            string            `json:"To"`
		Cc            string            `json:"Cc,omitempty"`
		Bcc           string            `json:"Bcc,omitempty"`
		Subject       string            `json:"Subject,omitempty"`
		Tag           string            `json:"Tag,omitempty"`
		HtmlBody      string            `json:"HtmlBody,omitempty"`
		TextBody      string            `json:"TextBody,omitempty"`
		ReplyTo       string            `json:"ReplyTo,omitempty"`
		Metadata      map[string]string `json:"Metadata,omitempty"`
		Headers       []MailHeader      `json:"Headers,omitempty"`
		Attachments   []Attachment      `json:"Attachments,omitempty"`
		TrackOpens    bool              `json:"TrackOpens,omitempty"`
		TrackLinks    string            `json:"TrackLinks,omitempty"`
		MessageStream string            `json:"MessageStream,omitempty"`
	}

	// SendEmailResp is the response returned by POST /email and the individual
	// items within a POST /email/batch response array.
	SendEmailResp struct {
		To          string `json:"To"`
		SubmittedAt string `json:"SubmittedAt"`
		MessageID   string `json:"MessageID"`
		ErrorCode   int    `json:"ErrorCode"`
		Message     string `json:"Message"`
	}

	// TemplateMessage is a single message entry in a batch-with-templates request.
	// It mirrors the individual message object accepted by POST /email/batchWithTemplates.
	TemplateMessage struct {
		From          string            `json:"From"`
		To            string            `json:"To"`
		Cc            string            `json:"Cc,omitempty"`
		Bcc           string            `json:"Bcc,omitempty"`
		ReplyTo       string            `json:"ReplyTo,omitempty"`
		Tag           string            `json:"Tag,omitempty"`
		Metadata      map[string]string `json:"Metadata,omitempty"`
		Headers       []MailHeader      `json:"Headers,omitempty"`
		Attachments   []Attachment      `json:"Attachments,omitempty"`
		TrackOpens    bool              `json:"TrackOpens,omitempty"`
		TrackLinks    string            `json:"TrackLinks,omitempty"`
		MessageStream string            `json:"MessageStream,omitempty"`
		// Template identification — supply either TemplateID or TemplateAlias.
		TemplateID    int64             `json:"TemplateId,omitempty"`
		TemplateAlias string            `json:"TemplateAlias,omitempty"`
		TemplateModel map[string]interface{} `json:"TemplateModel,omitempty"`
		InlineCss     bool              `json:"InlineCss,omitempty"`
	}

	// SendBatchWithTemplatesReq is the request body for POST /email/batchWithTemplates.
	SendBatchWithTemplatesReq struct {
		Messages []*TemplateMessage `json:"Messages"`
	}

	// SendBatchWithTemplatesResp is the response envelope for POST /email/batchWithTemplates.
	SendBatchWithTemplatesResp struct {
		TotalCount int              `json:"TotalCount"`
		Errors     []BatchEmailErr  `json:"Errors"`
		Messages   []*SendEmailResp `json:"Messages"`
	}

	// BatchEmailErr describes a single per-message error returned within a batch response.
	BatchEmailErr struct {
		ErrorCode int    `json:"ErrorCode"`
		Message   string `json:"Message"`
	}
)

// maxBatchSize is the maximum number of messages allowed in a single batch
// send call, as documented by the Postmark API.
const maxBatchSize = 500

// SendEmail sends a single email via POST /email using the server token.
// It returns a SendEmailResp describing the submission outcome.
func (a *API) SendEmail(emailReq *SendEmailReq) (*SendEmailResp, error) {
	req, err := a.newServerRequest(http.MethodPost, "email", emailReq)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data SendEmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// SendEmailBatch sends up to 500 emails in a single POST /email/batch call.
// It returns a slice of SendEmailResp values — one per submitted message — in
// the same order as the input slice.
// An error is returned if the batch exceeds the Postmark limit of 500 messages.
func (a *API) SendEmailBatch(reqs []*SendEmailReq) ([]*SendEmailResp, error) {
	if len(reqs) > maxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds the maximum of %d", len(reqs), maxBatchSize)
	}
	req, err := a.newServerRequest(http.MethodPost, "email/batch", reqs)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data []*SendEmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SendEmailBatchWithTemplates sends a batch of template-based emails via
// POST /email/batchWithTemplates. Each message in the batch must reference a
// Postmark template by ID or alias and supply the corresponding model.
func (a *API) SendEmailBatchWithTemplates(batchReq *SendBatchWithTemplatesReq) (*SendBatchWithTemplatesResp, error) {
	req, err := a.newServerRequest(http.MethodPost, "email/batchWithTemplates", batchReq)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data SendBatchWithTemplatesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
