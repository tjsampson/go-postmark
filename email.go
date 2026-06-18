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
	//
	// Field notes:
	//   - HtmlBody: HTML body of the email. At least one of HtmlBody or TextBody must be set.
	//   - TextBody: Plain-text body of the email. At least one of HtmlBody or TextBody must be set.
	//   - TrackOpens: Controls open tracking. Nil omits the field (server default applies);
	//     false explicitly disables tracking even when enabled by default on the stream.
	//   - MessageStream: Routes this message through the named stream. If empty,
	//     Postmark uses the default outbound stream.
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
		TrackOpens    *bool             `json:"TrackOpens,omitempty"`
		TrackLinks    string            `json:"TrackLinks,omitempty"`
		MessageStream string            `json:"MessageStream,omitempty"`
		Attachments   []Attachment      `json:"Attachments,omitempty"`
		Tag           string            `json:"Tag,omitempty"`
		Metadata      map[string]string `json:"Metadata,omitempty"`
	}

	// SendEmailResp is the response returned by the Postmark send-email endpoint.
	//
	// On success ErrorCode will be 0 and Message will be "OK". When Postmark
	// returns a structured application-level error (even on a 2xx HTTP status),
	// ErrorCode will be non-zero and Message will describe the problem. Callers
	// should check ErrorCode != 0 in addition to the returned Go error.
	SendEmailResp struct {
		To          string    `json:"To"`
		SubmittedAt time.Time `json:"SubmittedAt"`
		MessageID   string    `json:"MessageID"`
		ErrorCode   int       `json:"ErrorCode"`
		Message     string    `json:"Message"`
	}
)

// SendEmail sends a single email via the Postmark API (POST /email).
// It uses the server token (X-Postmark-Server-Token) for authentication;
// supply ServerTokenOpt or set POSTMARK_SERVER_TOKEN in the environment.
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
// for authentication; supply ServerTokenOpt or set POSTMARK_SERVER_TOKEN in
// the environment.
//
// A nil or empty slice is returned as ([]SendEmailResp{}, nil) immediately,
// without making a network request. Postmark rejects an empty batch array
// with a 422 Unprocessable Entity, so there is nothing to send.
func (a *API) SendEmailBatch(reqs []SendEmailReq) ([]SendEmailResp, error) {
	if len(reqs) == 0 {
		return []SendEmailResp{}, nil
	}
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
