package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type (
	// EmailHeader represents a custom email header (name/value pair).
	EmailHeader struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}

	// EmailAttachment represents a file attachment for an email.
	EmailAttachment struct {
		// Name is the file name of the attachment.
		Name string `json:"Name"`
		// Content is the Base64-encoded content of the attachment.
		Content string `json:"Content"`
		// ContentType is the MIME type of the attachment (e.g. "application/pdf").
		ContentType string `json:"ContentType"`
		// ContentID is optional and used for inline attachments (e.g. "cid:image1").
		ContentID string `json:"ContentID,omitempty"`
	}

	// SendEmailReq is the request body for sending a single email via POST /email.
	SendEmailReq struct {
		// From is the sender email address. Must be a verified Sender Signature.
		From string `json:"From"`
		// To is the recipient(s). Multiple addresses can be comma-separated.
		To string `json:"To"`
		// Cc is an optional comma-separated list of Cc recipients.
		Cc string `json:"Cc,omitempty"`
		// Bcc is an optional comma-separated list of Bcc recipients.
		Bcc string `json:"Bcc,omitempty"`
		// Subject is the email subject line.
		Subject string `json:"Subject,omitempty"`
		// Tag is used to categorise outbound email for use with Postmark's statistics.
		Tag string `json:"Tag,omitempty"`
		// HtmlBody is the HTML body of the email.
		HtmlBody string `json:"HtmlBody,omitempty"`
		// TextBody is the plain-text body of the email.
		TextBody string `json:"TextBody,omitempty"`
		// ReplyTo is the Reply-To address.
		ReplyTo string `json:"ReplyTo,omitempty"`
		// Headers are custom email headers to include.
		Headers []EmailHeader `json:"Headers,omitempty"`
		// TrackOpens enables or disables open tracking for this email.
		// Use a pointer so that an explicit false can be sent to override an
		// account-level default; a nil value omits the field entirely.
		TrackOpens *bool `json:"TrackOpens,omitempty"`
		// TrackLinks controls click tracking. Valid values: "None", "HtmlAndText",
		// "HtmlOnly", "TextOnly".
		TrackLinks string `json:"TrackLinks,omitempty"`
		// Attachments is a list of file attachments to include.
		Attachments []EmailAttachment `json:"Attachments,omitempty"`
		// Metadata holds key/value pairs to associate with the message.
		Metadata map[string]string `json:"Metadata,omitempty"`
		// MessageStream is the message stream to use. Defaults to "outbound".
		MessageStream string `json:"MessageStream,omitempty"`
	}

	// SendEmailResp is the response returned by the /email, /email/batch,
	// /email/withTemplate, and /email/batchWithTemplates endpoints.
	//
	// For single-send endpoints (SendEmail, SendWithTemplate) the library
	// automatically converts a non-zero ErrorCode into a Go error, so callers
	// do not need to inspect this field themselves.
	//
	// For batch endpoints (SendBatch, SendBatchWithTemplates) Postmark returns
	// HTTP 200 with a per-item ErrorCode for each message that failed. The
	// library does NOT aggregate these into a Go error because partial success
	// is a valid and common outcome. Callers MUST inspect ErrorCode on every
	// element of the returned slice (0 = success, non-zero = failure).
	SendEmailResp struct {
		// To is the recipient address.
		To string `json:"To"`
		// SubmittedAt is the UTC timestamp when the message was accepted.
		SubmittedAt string `json:"SubmittedAt"`
		// MessageID is the unique Postmark message ID.
		MessageID string `json:"MessageID"`
		// ErrorCode is the Postmark error code (0 = success).
		// A non-zero value indicates a per-message failure even within an HTTP 200
		// response. For batch endpoints callers must check this field on every
		// element; for single-send endpoints the library surfaces it as a Go error.
		ErrorCode int `json:"ErrorCode"`
		// Message describes the outcome (e.g. "OK").
		Message string `json:"Message"`
	}

	// SendTemplateReq is the request body for sending an email using a
	// Postmark template via POST /email/withTemplate.
	SendTemplateReq struct {
		// From is the sender email address. Must be a verified Sender Signature.
		From string `json:"From"`
		// To is the recipient(s). Multiple addresses can be comma-separated.
		To string `json:"To"`
		// Cc is an optional comma-separated list of Cc recipients.
		Cc string `json:"Cc,omitempty"`
		// Bcc is an optional comma-separated list of Bcc recipients.
		Bcc string `json:"Bcc,omitempty"`
		// ReplyTo is the Reply-To address.
		ReplyTo string `json:"ReplyTo,omitempty"`
		// Tag is used to categorise outbound email.
		Tag string `json:"Tag,omitempty"`
		// Subject overrides the template's subject line for this send.
		Subject string `json:"Subject,omitempty"`
		// TrackOpens enables or disables open tracking for this email.
		// Use a pointer so that an explicit false can be sent to override an
		// account-level default; a nil value omits the field entirely.
		TrackOpens *bool `json:"TrackOpens,omitempty"`
		// TrackLinks controls click tracking. Valid values: "None", "HtmlAndText",
		// "HtmlOnly", "TextOnly".
		TrackLinks string `json:"TrackLinks,omitempty"`
		// TemplateID is the numeric ID of the template to use.
		// Either TemplateID or TemplateAlias must be set. Because 0 is not a valid
		// Postmark template ID, omitempty is safe here and avoids sending a
		// spurious zero when only TemplateAlias is provided.
		TemplateID int64 `json:"TemplateId,omitempty"`
		// TemplateAlias is the string alias of the template to use.
		// Either TemplateID or TemplateAlias must be set.
		TemplateAlias string `json:"TemplateAlias,omitempty"`
		// TemplateModel is the data model passed to the template for rendering.
		TemplateModel map[string]interface{} `json:"TemplateModel,omitempty"`
		// Headers are custom email headers to include.
		Headers []EmailHeader `json:"Headers,omitempty"`
		// Attachments is a list of file attachments to include.
		Attachments []EmailAttachment `json:"Attachments,omitempty"`
		// Metadata holds key/value pairs to associate with the message.
		Metadata map[string]string `json:"Metadata,omitempty"`
		// MessageStream is the message stream to use. Defaults to "outbound".
		MessageStream string `json:"MessageStream,omitempty"`
		// InlineCss controls whether CSS in the <head> is inlined into HTML.
		// Use a pointer so that an explicit false can be sent to override an
		// account-level default; a nil value omits the field entirely.
		InlineCss *bool `json:"InlineCss,omitempty"`
	}

	// batchTemplateReqWrapper is the envelope Postmark expects for
	// POST /email/batchWithTemplates. The Postmark API for this endpoint
	// requires a JSON object with a top-level "Messages" key wrapping the array
	// (https://postmarkapp.com/developer/api/email-api#send-batch-with-templates).
	batchTemplateReqWrapper struct {
		Messages []SendTemplateReq `json:"Messages"`
	}

	// BulkJobBatch holds per-batch outcome information within a bulk job response.
	BulkJobBatch struct {
		// StartedAt is the UTC timestamp when this batch started processing.
		StartedAt string `json:"StartedAt"`
		// CompletedAt is the UTC timestamp when this batch finished processing.
		CompletedAt string `json:"CompletedAt"`
		// TotalCount is the total number of messages in this batch.
		TotalCount int `json:"TotalCount"`
		// SuccessCount is the number of messages sent successfully in this batch.
		SuccessCount int `json:"SuccessCount"`
		// ErrorCount is the number of messages that failed in this batch.
		ErrorCount int `json:"ErrorCount"`
	}

	// BulkJobResp is the response returned by the POST /email/bulk and
	// GET /email/bulk/{bulk-request-id} endpoints.
	//
	// Unlike the single-send and batch endpoints, the Postmark bulk API does not
	// embed per-message ErrorCode/Message fields directly in this response.
	// Overall job health is indicated by the Status, SuccessCount, and ErrorCount
	// fields. HTTP-level errors from the API (non-2xx) are surfaced as Go errors
	// by the underlying Do() call.
	BulkJobResp struct {
		// ID is the unique identifier for the bulk job.
		ID string `json:"ID"`
		// CreatedAt is the UTC timestamp when the bulk job was created.
		CreatedAt string `json:"CreatedAt"`
		// StartedProcessingAt is the UTC timestamp when the job began processing.
		StartedProcessingAt string `json:"StartedProcessingAt,omitempty"`
		// CompletedProcessingAt is the UTC timestamp when the job finished processing.
		CompletedProcessingAt string `json:"CompletedProcessingAt,omitempty"`
		// Status is the current status of the job (e.g. "Queued", "Processing",
		// "Completed").
		Status string `json:"Status"`
		// TotalCount is the total number of messages in the job.
		TotalCount int `json:"TotalCount"`
		// SuccessCount is the number of messages sent successfully.
		SuccessCount int `json:"SuccessCount"`
		// ErrorCount is the number of messages that failed.
		ErrorCount int `json:"ErrorCount"`
		// Batches contains per-batch outcome details. Present once the job has
		// started processing.
		Batches []BulkJobBatch `json:"Batches,omitempty"`
	}
)

// checkEmailRespError inspects a SendEmailResp for a non-zero ErrorCode and
// returns a PostmarkErr so callers receive a Go error rather than silently
// getting a response with a failed outcome embedded inside an HTTP 200.
// Postmark uses ErrorCode/Message for per-message failures even in 200 responses.
//
// This helper is intentionally NOT called for batch methods (SendBatch,
// SendBatchWithTemplates) because partial success — some messages delivered,
// others failed — is a valid and common outcome for those endpoints. Callers
// of batch methods must inspect each SendEmailResp.ErrorCode individually.
func checkEmailRespError(r *SendEmailResp) error {
	if r.ErrorCode != 0 {
		return PostmarkErr{ErrorCode: r.ErrorCode, Message: r.Message}
	}
	return nil
}

// SendEmail sends a single email message via POST /email.
// Authentication uses the X-Postmark-Server-Token header; configure the token
// with ServerTokenOpt when calling New().
//
// An error is returned for both transport failures and for Postmark
// per-message errors (non-zero ErrorCode in the response body), so callers
// do not need to inspect SendEmailResp.ErrorCode themselves.
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
	if err = checkEmailRespError(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

// SendBatch sends a batch of email messages via POST /email/batch.
// Authentication uses the X-Postmark-Server-Token header; configure the token
// with ServerTokenOpt when calling New().
//
// The Postmark /email/batch endpoint expects a bare JSON array as the request
// body (https://postmarkapp.com/developer/api/email-api#send-batch-emails).
//
// An error is returned for transport or HTTP-level failures (non-2xx status).
// Postmark returns HTTP 200 even when individual messages fail: per-message
// outcomes are represented in each SendEmailResp.ErrorCode (0 = success).
// Partial success — some messages delivered, others not — is a normal outcome.
// Callers MUST inspect ErrorCode on every element of the returned slice.
func (a *API) SendBatch(reqs []SendEmailReq) ([]SendEmailResp, error) {
	// /email/batch expects a bare JSON array, not a wrapped object.
	req, err := a.newServerRequest(http.MethodPost, "email/batch", reqs)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}
	var data []SendEmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SendWithTemplate sends an email using a Postmark template via
// POST /email/withTemplate.
// Authentication uses the X-Postmark-Server-Token header; configure the token
// with ServerTokenOpt when calling New().
//
// An error is returned for both transport failures and for Postmark
// per-message errors (non-zero ErrorCode in the response body).
func (a *API) SendWithTemplate(tmplReq *SendTemplateReq) (*SendEmailResp, error) {
	req, err := a.newServerRequest(http.MethodPost, "email/withTemplate", tmplReq)
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
	if err = checkEmailRespError(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

// SendBatchWithTemplates sends a batch of templated emails via
// POST /email/batchWithTemplates.
// Authentication uses the X-Postmark-Server-Token header; configure the token
// with ServerTokenOpt when calling New().
//
// The Postmark /email/batchWithTemplates endpoint requires a JSON object with a
// top-level "Messages" key wrapping the array
// (https://postmarkapp.com/developer/api/email-api#send-batch-with-templates);
// this wrapper is applied automatically.
//
// An error is returned for transport or HTTP-level failures (non-2xx status).
// Postmark returns HTTP 200 even when individual messages fail: per-message
// outcomes are represented in each SendEmailResp.ErrorCode (0 = success).
// Partial success — some messages delivered, others not — is a normal outcome.
// Callers MUST inspect ErrorCode on every element of the returned slice.
func (a *API) SendBatchWithTemplates(reqs []SendTemplateReq) ([]SendEmailResp, error) {
	payload := batchTemplateReqWrapper{Messages: reqs}
	req, err := a.newServerRequest(http.MethodPost, "email/batchWithTemplates", payload)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}
	var data []SendEmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// CreateBulkJob creates a bulk email job via POST /email/bulk.
// Authentication uses the X-Postmark-Server-Token header; configure the token
// with ServerTokenOpt when calling New().
//
// The Postmark /email/bulk endpoint expects a bare JSON array as the request
// body (https://postmarkapp.com/developer/api/email-api#send-bulk-email).
//
// Unlike the single-send and batch endpoints, BulkJobResp does not contain
// per-message ErrorCode fields. HTTP-level errors are returned as Go errors;
// per-message outcomes within a completed job are available via GetBulkJob
// (reflected in SuccessCount and ErrorCount).
func (a *API) CreateBulkJob(reqs []SendEmailReq) (*BulkJobResp, error) {
	// /email/bulk expects a bare JSON array, not a wrapped object.
	req, err := a.newServerRequest(http.MethodPost, "email/bulk", reqs)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}
	var data BulkJobResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetBulkJob retrieves the status of a bulk email job via
// GET /email/bulk/{bulk-request-id}.
// Authentication uses the X-Postmark-Server-Token header; configure the token
// with ServerTokenOpt when calling New().
//
// id is URL-path-escaped before use so that callers may pass arbitrary job IDs
// without risk of path injection.
func (a *API) GetBulkJob(id string) (*BulkJobResp, error) {
	if id == "" {
		return nil, fmt.Errorf("postmark: GetBulkJob: id must not be empty")
	}
	req, err := a.newServerRequest(http.MethodGet, "email/bulk/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}
	var data BulkJobResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
