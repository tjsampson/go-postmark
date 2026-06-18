package postmark

import (
	"encoding/json"
	"net/http"
)

type (
	// Attachment represents a file attachment to include in an email.
	Attachment struct {
		// Name is the filename of the attachment.
		Name string `json:"Name"`
		// Content is the base64-encoded content of the attachment.
		Content string `json:"Content"`
		// ContentType is the MIME type of the attachment (e.g. "application/pdf").
		ContentType string `json:"ContentType"`
		// ContentID is used for inline attachments (e.g. "cid:image1").
		ContentID string `json:"ContentID,omitempty"`
	}

	// Header represents a custom email header.
	Header struct {
		// Name is the header field name (e.g. "X-Custom-Header").
		Name string `json:"Name"`
		// Value is the header field value.
		Value string `json:"Value"`
	}

	// EmailReq is the request body for sending a single email via Postmark.
	EmailReq struct {
		// From is the sender email address.
		From string `json:"From"`
		// To is the recipient email address(es), comma-separated.
		To string `json:"To"`
		// Cc is the CC recipient email address(es), comma-separated.
		Cc string `json:"Cc,omitempty"`
		// Bcc is the BCC recipient email address(es), comma-separated.
		Bcc string `json:"Bcc,omitempty"`
		// ReplyTo is the reply-to email address.
		ReplyTo string `json:"ReplyTo,omitempty"`
		// Subject is the email subject line.
		Subject string `json:"Subject,omitempty"`
		// TextBody is the plain-text body of the email.
		TextBody string `json:"TextBody,omitempty"`
		// HtmlBody is the HTML body of the email.
		HtmlBody string `json:"HtmlBody,omitempty"`
		// Tag is an optional tag for categorising the email in Postmark.
		Tag string `json:"Tag,omitempty"`
		// TrackOpens enables open tracking for the email.
		TrackOpens bool `json:"TrackOpens,omitempty"`
		// TrackLinks controls link tracking ("None", "HtmlAndText", "HtmlOnly", "TextOnly").
		TrackLinks string `json:"TrackLinks,omitempty"`
		// MessageStream is the message stream to use (e.g. "outbound").
		MessageStream string `json:"MessageStream,omitempty"`
		// Attachments is the list of file attachments to include.
		Attachments []Attachment `json:"Attachments,omitempty"`
		// Headers is the list of custom email headers to include.
		Headers []Header `json:"Headers,omitempty"`
		// Metadata is a map of custom metadata key-value pairs.
		Metadata map[string]string `json:"Metadata,omitempty"`
	}

	// EmailResp is the response returned by Postmark after sending a single email.
	EmailResp struct {
		// To is the recipient email address the message was sent to.
		To string `json:"To"`
		// SubmittedAt is the timestamp when the message was submitted.
		SubmittedAt string `json:"SubmittedAt"`
		// MessageID is the unique identifier assigned by Postmark for the message.
		MessageID string `json:"MessageID"`
		// ErrorCode is the Postmark error code (0 indicates success).
		ErrorCode int `json:"ErrorCode"`
		// Message is a human-readable description of the result.
		Message string `json:"Message"`
	}

	// BatchEmailResp is the response returned by Postmark after sending a batch of emails.
	// Each element corresponds to the result for one email in the batch request.
	BatchEmailResp []EmailResp
)

// SendEmail sends a single email via the Postmark API (POST /email).
// It uses the server token (X-Postmark-Server-Token) for authentication.
func (a *API) SendEmail(req *EmailReq) (*EmailResp, error) {
	httpReq, err := a.newServerRequest(http.MethodPost, "email", req)
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
	return &data, nil
}

// SendEmailBatch sends a batch of up to 500 emails via the Postmark API (POST /email/batch).
// It uses the server token (X-Postmark-Server-Token) for authentication.
// Each element in the returned BatchEmailResp corresponds to one email in the request slice.
func (a *API) SendEmailBatch(reqs []*EmailReq) (BatchEmailResp, error) {
	httpReq, err := a.newServerRequest(http.MethodPost, "email/batch", reqs)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data BatchEmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return data, nil
}
