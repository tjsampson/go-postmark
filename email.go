package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type (
	// EmailHeader represents a custom email header with a name/value pair.
	EmailHeader struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}

	// Attachment represents a file attachment to include with an email.
	// Content must be base64-encoded.
	Attachment struct {
		Name        string `json:"Name"`
		Content     string `json:"Content"`
		ContentType string `json:"ContentType"`
		ContentID   string `json:"ContentID,omitempty"`
	}

	// SendEmailReq is the request body for sending a single email via Postmark.
	//
	// At least one of HtmlBody or TextBody must be non-empty; Postmark will
	// reject the request with a 422 if both are absent.
	SendEmailReq struct {
		From          string            `json:"From"`
		To            string            `json:"To"`
		Cc            string            `json:"Cc,omitempty"`
		Bcc           string            `json:"Bcc,omitempty"`
		Subject       string            `json:"Subject"`
		// HtmlBody is the HTML body of the email. At least one of HtmlBody or
		// TextBody must be set; Postmark requires a non-empty body.
		HtmlBody      string            `json:"HtmlBody,omitempty"`
		// TextBody is the plain-text body of the email. At least one of HtmlBody
		// or TextBody must be set; Postmark requires a non-empty body.
		TextBody      string            `json:"TextBody,omitempty"`
		ReplyTo       string            `json:"ReplyTo,omitempty"`
		Headers       []EmailHeader     `json:"Headers,omitempty"`
		// TrackOpens controls whether open tracking is enabled for this message.
		// A nil value omits the field (server default applies); false explicitly
		// disables tracking even on a stream where it is enabled by default.
		TrackOpens    *bool             `json:"TrackOpens,omitempty"`
		TrackLinks    string            `json:"TrackLinks,omitempty"`
		MessageStream string            `json:"MessageStream,omitempty"`
		Attachments   []Attachment      `json:"Attachments,omitempty"`
		Tag           string            `json:"Tag,omitempty"`
		Metadata      map[string]string `json:"Metadata,omitempty"`
	}

	// SendEmailResp is the response returned by the Postmark send-email endpoint.
	SendEmailResp struct {
		To          string    `json:"To"`
		SubmittedAt time.Time `json:"SubmittedAt"`
		MessageID   string    `json:"MessageID"`
		ErrorCode   int       `json:"ErrorCode"`
		Message     string    `json:"Message"`
	}
)

// SendEmail sends a single email via the Postmark API (POST /email).
// It uses the server token (X-Postmark-Server-Token) for authentication.
// req must not be nil.
func (a *API) SendEmail(req *SendEmailReq) (*SendEmailResp, error) {
	if req == nil {
		return nil, fmt.Errorf("postmark: SendEmail called with nil request")
	}
	httpReq, err := a.newServerRequest(http.MethodPost, "email", req)
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

// SendEmailBatch sends multiple emails in a single request via the Postmark
// API (POST /email/batch). It uses the server token (X-Postmark-Server-Token)
// for authentication.
func (a *API) SendEmailBatch(reqs []SendEmailReq) ([]SendEmailResp, error) {
	httpReq, err := a.newServerRequest(http.MethodPost, "email/batch", reqs)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data []SendEmailResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return data, nil
}
