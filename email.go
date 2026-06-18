package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// TrackLinksValue is the set of valid values for the TrackLinks field.
// Postmark only accepts these four values; any other string will result in
// an API error. Using this type and its constants prevents invalid values
// from being set silently.
type TrackLinksValue string

const (
	// TrackLinksNone explicitly disables link tracking. This is distinct from
	// omitting the TrackLinks field entirely: omitting the field (leaving it as
	// the zero value "") causes Postmark to use the message-stream default,
	// whereas TrackLinksNone overrides the stream default to "no tracking".
	TrackLinksNone TrackLinksValue = "None"
	// TrackLinksHtmlAndText enables link tracking in both HTML and plain-text parts.
	TrackLinksHtmlAndText TrackLinksValue = "HtmlAndText"
	// TrackLinksHtmlOnly enables link tracking in the HTML part only.
	TrackLinksHtmlOnly TrackLinksValue = "HtmlOnly"
	// TrackLinksTextOnly enables link tracking in the plain-text part only.
	TrackLinksTextOnly TrackLinksValue = "TextOnly"
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
	//
	// TrackOpens is a *bool so that an explicit false is serialised as
	// {"TrackOpens":false} rather than being omitted from the JSON payload.
	// A nil value causes the field to be omitted, which lets the message-stream
	// default take effect — the same behaviour as not setting the field at all.
	//
	// TrackLinks uses omitempty with the zero value "" (not a valid Postmark
	// value). Leaving TrackLinks unset (zero value) omits the field and defers
	// to the message-stream default. Set TrackLinksNone to explicitly disable
	// link tracking regardless of the stream default — these are distinct
	// behaviours on Postmark's side. Must be one of the TrackLinksValue
	// constants (None, HtmlAndText, HtmlOnly, TextOnly); any other string will
	// be rejected by the Postmark API with an error at send time.
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
		TrackOpens    *bool             `json:"TrackOpens,omitempty"`
		TrackLinks    TrackLinksValue   `json:"TrackLinks,omitempty"`
		MessageStream string            `json:"MessageStream,omitempty"`
	}

	// SendEmailResp is the response returned by POST /email and the individual
	// items within a POST /email/batch response array.
	//
	// SubmittedAt is a raw string containing the RFC 3339 timestamp returned by
	// Postmark. It is intentionally kept as a string to avoid a custom time.Time
	// unmarshaller; callers who need to compare or sort by time should parse it
	// with time.Parse(time.RFC3339, resp.SubmittedAt).
	SendEmailResp struct {
		To          string `json:"To"`
		SubmittedAt string `json:"SubmittedAt"`
		MessageID   string `json:"MessageID"`
		ErrorCode   int    `json:"ErrorCode"`
		Message     string `json:"Message"`
	}

	// TemplateMessage is a single message entry in a batch-with-templates request.
	// It mirrors the individual message object accepted by POST /email/batchWithTemplates.
	//
	// Either TemplateID or TemplateAlias must be supplied; both may not be omitted.
	// SendEmailBatchWithTemplates performs a pre-flight check and returns an error
	// if neither field is set on any message in the batch.
	//
	// TemplateID uses json:"TemplateId,omitempty". The zero value (0) is correctly
	// omitted: 0 is not a valid Postmark template ID, so an absent TemplateID in
	// JSON causes no unintended side-effects. When a non-zero TemplateID is set it
	// takes precedence; TemplateAlias is ignored by Postmark if TemplateID is set.
	//
	// InlineCss is a *bool so that an explicit false is serialised correctly
	// (see SendEmailReq.TrackOpens for the full rationale).
	//
	// TrackLinks must be one of the TrackLinksValue constants (see SendEmailReq.TrackLinks).
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
		TrackOpens    *bool             `json:"TrackOpens,omitempty"`
		TrackLinks    TrackLinksValue   `json:"TrackLinks,omitempty"`
		MessageStream string            `json:"MessageStream,omitempty"`
		// Template identification — supply either TemplateID or TemplateAlias.
		TemplateID    int64          `json:"TemplateId,omitempty"`
		TemplateAlias string         `json:"TemplateAlias,omitempty"`
		TemplateModel map[string]any `json:"TemplateModel,omitempty"`
		InlineCss     *bool          `json:"InlineCss,omitempty"`
	}

	// SendBatchWithTemplatesReq is the request body for POST /email/batchWithTemplates.
	SendBatchWithTemplatesReq struct {
		Messages []*TemplateMessage `json:"Messages"`
	}

	// SendBatchWithTemplatesResp is the response envelope for POST /email/batchWithTemplates.
	// The Postmark API returns {"TotalCount":N,"Messages":[...]} where each message
	// carries its own ErrorCode and Message fields for per-message failures. There is
	// no separate top-level errors array; all error information is embedded within the
	// individual Messages entries.
	SendBatchWithTemplatesResp struct {
		TotalCount int              `json:"TotalCount"`
		Messages   []*SendEmailResp `json:"Messages"`
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
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
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
//
// An error is returned if:
//   - the batch is empty (Postmark requires at least one message), or
//   - the batch exceeds the Postmark limit of 500 messages.
func (a *API) SendEmailBatch(reqs []*SendEmailReq) ([]*SendEmailResp, error) {
	if len(reqs) == 0 {
		return nil, fmt.Errorf("batch must contain at least one message")
	}
	if len(reqs) > maxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds the maximum of %d", len(reqs), maxBatchSize)
	}
	req, err := a.newServerRequest(http.MethodPost, "email/batch", reqs)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
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
//
// An error is returned if:
//   - batchReq is nil,
//   - the batch is empty,
//   - the batch exceeds the Postmark limit of 500 messages, or
//   - any message omits both TemplateID and TemplateAlias.
func (a *API) SendEmailBatchWithTemplates(batchReq *SendBatchWithTemplatesReq) (*SendBatchWithTemplatesResp, error) {
	if batchReq == nil {
		return nil, fmt.Errorf("batchReq must not be nil")
	}
	if len(batchReq.Messages) == 0 {
		return nil, fmt.Errorf("batch must contain at least one message")
	}
	if len(batchReq.Messages) > maxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds the maximum of %d", len(batchReq.Messages), maxBatchSize)
	}
	for i, msg := range batchReq.Messages {
		if msg.TemplateID == 0 && msg.TemplateAlias == "" {
			return nil, fmt.Errorf("message at index %d must set either TemplateID or TemplateAlias", i)
		}
	}

	req, err := a.newServerRequest(http.MethodPost, "email/batchWithTemplates", batchReq)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data SendBatchWithTemplatesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
