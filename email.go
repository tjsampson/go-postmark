package postmark

import (
	"encoding/json"
	"net/http"
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
	SendEmailReq struct {
		From          string            `json:"From"`
		To            string            `json:"To"`
		Cc            string            `json:"Cc,omitempty"`
		Bcc           string            `json:"Bcc,omitempty"`
		Subject       string            `json:"Subject"`
		HtmlBody      string            `json:"HtmlBody,omitempty"`
		TextBody      string            `json:"TextBody,omitempty"`
		ReplyTo       string            `json:"ReplyTo,omitempty"`
		Headers       []EmailHeader     `json:"Headers,omitempty"`
		TrackOpens    bool              `json:"TrackOpens,omitempty"`
		TrackLinks    string            `json:"TrackLinks,omitempty"`
		MessageStream string            `json:"MessageStream,omitempty"`
		Attachments   []Attachment      `json:"Attachments,omitempty"`
		Tag           string            `json:"Tag,omitempty"`
		Metadata      map[string]string `json:"Metadata,omitempty"`
	}

	// SendEmailResp is the response returned by the Postmark send-email endpoint.
	SendEmailResp struct {
		To          string `json:"To"`
		SubmittedAt string `json:"SubmittedAt"`
		MessageID   string `json:"MessageID"`
		ErrorCode   int    `json:"ErrorCode"`
		Message     string `json:"Message"`
	}
)

// SendEmail sends a single email via the Postmark API (POST /email).
// It uses the server token (X-Postmark-Server-Token) for authentication.
func (a *API) SendEmail(req *SendEmailReq) (*SendEmailResp, error) {
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
