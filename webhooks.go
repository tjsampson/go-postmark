package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type (
	// WebhookHTTPAuth holds the HTTP Basic Auth credentials for a webhook.
	WebhookHTTPAuth struct {
		Username string `json:"Username"`
		Password string `json:"Password"`
	}

	// WebhookHTTPHeader represents a single custom HTTP header sent with webhook requests.
	WebhookHTTPHeader struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}

	// WebhookTriggerOpen holds settings for the Open trigger.
	WebhookTriggerOpen struct {
		Enabled     bool `json:"Enabled"`
		PostFirstOpenOnly bool `json:"PostFirstOpenOnly"`
	}

	// WebhookTriggerClick holds settings for the Click trigger.
	WebhookTriggerClick struct {
		Enabled bool `json:"Enabled"`
	}

	// WebhookTriggerDelivery holds settings for the Delivery trigger.
	WebhookTriggerDelivery struct {
		Enabled bool `json:"Enabled"`
	}

	// WebhookTriggerBounce holds settings for the Bounce trigger.
	WebhookTriggerBounce struct {
		Enabled                bool `json:"Enabled"`
		IncludeContent         bool `json:"IncludeContent"`
	}

	// WebhookTriggerSpamComplaint holds settings for the SpamComplaint trigger.
	WebhookTriggerSpamComplaint struct {
		Enabled        bool `json:"Enabled"`
		IncludeContent bool `json:"IncludeContent"`
	}

	// WebhookTriggerSubscriptionChange holds settings for the SubscriptionChange trigger.
	WebhookTriggerSubscriptionChange struct {
		Enabled bool `json:"Enabled"`
	}

	// WebhookTriggers groups all available webhook trigger settings.
	WebhookTriggers struct {
		Open               WebhookTriggerOpen               `json:"Open"`
		Click              WebhookTriggerClick              `json:"Click"`
		Delivery           WebhookTriggerDelivery           `json:"Delivery"`
		Bounce             WebhookTriggerBounce             `json:"Bounce"`
		SpamComplaint      WebhookTriggerSpamComplaint      `json:"SpamComplaint"`
		SubscriptionChange WebhookTriggerSubscriptionChange `json:"SubscriptionChange"`
	}

	// WebhookResp represents a webhook as returned by the Postmark API.
	WebhookResp struct {
		ID            int                 `json:"ID"`
		Url           string              `json:"Url"`
		MessageStream string              `json:"MessageStream"`
		HttpAuth      *WebhookHTTPAuth    `json:"HttpAuth,omitempty"`
		HttpHeaders   []WebhookHTTPHeader `json:"HttpHeaders,omitempty"`
		Triggers      WebhookTriggers     `json:"Triggers"`
	}

	// ListWebhooksResp is the response envelope returned by the list webhooks endpoint.
	ListWebhooksResp struct {
		Webhooks []WebhookResp `json:"Webhooks"`
	}

	// CreateWebhookReq is the request body for creating a new webhook.
	CreateWebhookReq struct {
		Url           string              `json:"Url"`
		MessageStream string              `json:"MessageStream"`
		HttpAuth      *WebhookHTTPAuth    `json:"HttpAuth,omitempty"`
		HttpHeaders   []WebhookHTTPHeader `json:"HttpHeaders,omitempty"`
		Triggers      WebhookTriggers     `json:"Triggers"`
	}

	// EditWebhookReq is the request body for updating an existing webhook.
	EditWebhookReq struct {
		Url         string              `json:"Url,omitempty"`
		HttpAuth    *WebhookHTTPAuth    `json:"HttpAuth,omitempty"`
		HttpHeaders []WebhookHTTPHeader `json:"HttpHeaders,omitempty"`
		Triggers    WebhookTriggers     `json:"Triggers"`
	}
)

// ListWebhooks returns all webhooks for the server. If messageStream is non-empty,
// results are filtered to that stream.
// Corresponds to GET /webhooks (with optional ?MessageStream= query param).
func (a *API) ListWebhooks(messageStream string) (*ListWebhooksResp, error) {
	path := "webhooks"
	if messageStream != "" {
		path += "?MessageStream=" + messageStream
	}

	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data ListWebhooksResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetWebhook retrieves a single webhook by its ID.
// Corresponds to GET /webhooks/{webhookID}.
func (a *API) GetWebhook(webhookID int) (*WebhookResp, error) {
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("webhooks/%d", webhookID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data WebhookResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateWebhook creates a new webhook with the settings in body.
// Corresponds to POST /webhooks.
func (a *API) CreateWebhook(body *CreateWebhookReq) (*WebhookResp, error) {
	req, err := a.newRequest(http.MethodPost, "webhooks", body)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data WebhookResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// EditWebhook updates the webhook identified by webhookID with the settings in body.
// Corresponds to PUT /webhooks/{webhookID}.
func (a *API) EditWebhook(webhookID int, body *EditWebhookReq) (*WebhookResp, error) {
	req, err := a.newRequest(http.MethodPut, fmt.Sprintf("webhooks/%d", webhookID), body)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data WebhookResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteWebhook deletes the webhook identified by webhookID.
// Corresponds to DELETE /webhooks/{webhookID}.
func (a *API) DeleteWebhook(webhookID int) (*DeleteResp, error) {
	req, err := a.newRequest(http.MethodDelete, fmt.Sprintf("webhooks/%d", webhookID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data DeleteResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
