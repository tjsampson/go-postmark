package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type (
	// NameValue is a simple name/value pair used in webhook headers.
	NameValue struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}

	// WebhookHttpAuth holds the HTTP Basic Auth credentials for a webhook.
	WebhookHttpAuth struct {
		Username string `json:"Username"`
		Password string `json:"Password"`
	}

	// WebhookTriggerOpen holds the configuration for open event triggers.
	WebhookTriggerOpen struct {
		Enabled     bool `json:"Enabled"`
		PostFirstOpenOnly bool `json:"PostFirstOpenOnly"`
	}

	// WebhookTriggerEnabled holds a simple enabled flag for a trigger type.
	WebhookTriggerEnabled struct {
		Enabled bool `json:"Enabled"`
	}

	// WebhookTriggerClick holds the configuration for click event triggers.
	WebhookTriggerClick struct {
		Enabled bool `json:"Enabled"`
	}

	// WebhookTriggerDelivery holds the configuration for delivery event triggers.
	WebhookTriggerDelivery struct {
		Enabled bool `json:"Enabled"`
	}

	// WebhookTriggerBounce holds the configuration for bounce event triggers.
	WebhookTriggerBounce struct {
		Enabled     bool `json:"Enabled"`
		IncludeContent bool `json:"IncludeContent"`
	}

	// WebhookTriggerSpamComplaint holds the configuration for spam complaint event triggers.
	WebhookTriggerSpamComplaint struct {
		Enabled        bool `json:"Enabled"`
		IncludeContent bool `json:"IncludeContent"`
	}

	// WebhookTriggerSubscriptionChange holds the configuration for subscription change event triggers.
	WebhookTriggerSubscriptionChange struct {
		Enabled bool `json:"Enabled"`
	}

	// WebhookTriggers holds all trigger configurations for a webhook.
	WebhookTriggers struct {
		Open               WebhookTriggerOpen               `json:"Open"`
		Click              WebhookTriggerClick              `json:"Click"`
		Delivery           WebhookTriggerDelivery           `json:"Delivery"`
		Bounce             WebhookTriggerBounce             `json:"Bounce"`
		SpamComplaint      WebhookTriggerSpamComplaint      `json:"SpamComplaint"`
		SubscriptionChange WebhookTriggerSubscriptionChange `json:"SubscriptionChange"`
	}

	// WebhookResp represents a Postmark Webhook as returned by the API.
	WebhookResp struct {
		ID            int             `json:"ID"`
		Url           string          `json:"Url"`
		MessageStream string          `json:"MessageStream"`
		HttpAuth      WebhookHttpAuth `json:"HttpAuth"`
		Headers       []NameValue     `json:"Headers"`
		Triggers      WebhookTriggers `json:"Triggers"`
	}

	// ListWebhooksResp is the response envelope returned by the list-webhooks endpoint.
	ListWebhooksResp struct {
		Webhooks []WebhookResp `json:"Webhooks"`
	}

	// CreateWebhookReq is the request body for creating or updating a webhook.
	CreateWebhookReq struct {
		Url           string          `json:"Url"`
		MessageStream string          `json:"MessageStream,omitempty"`
		HttpAuth      *WebhookHttpAuth `json:"HttpAuth,omitempty"`
		Headers       []NameValue     `json:"Headers,omitempty"`
		Triggers      *WebhookTriggers `json:"Triggers,omitempty"`
	}
)

// ListWebhooks returns a list of all webhooks on the account, optionally
// filtered by messageStream. Pass an empty string to list all webhooks.
func (a *API) ListWebhooks(messageStream string) (*ListWebhooksResp, error) {
	params := url.Values{}
	if messageStream != "" {
		params.Set("MessageStream", messageStream)
	}
	path := "webhooks"
	if len(params) > 0 {
		path = "webhooks?" + params.Encode()
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

// CreateWebhook creates a new webhook with the settings in req.
// It returns the full WebhookResp on success.
func (a *API) CreateWebhook(req *CreateWebhookReq) (*WebhookResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "webhooks", req)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}
	var data WebhookResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetWebhook fetches the webhook identified by webhookID.
func (a *API) GetWebhook(webhookID int64) (*WebhookResp, error) {
	httpReq, err := a.newRequest(http.MethodGet, fmt.Sprintf("webhooks/%d", webhookID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}
	var data WebhookResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateWebhook applies the changes in req to the webhook identified by
// webhookID and returns the updated WebhookResp.
func (a *API) UpdateWebhook(webhookID int64, req *CreateWebhookReq) (*WebhookResp, error) {
	httpReq, err := a.newRequest(http.MethodPut, fmt.Sprintf("webhooks/%d", webhookID), req)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
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
// It returns a DeleteResp containing the outcome message from the API.
func (a *API) DeleteWebhook(webhookID int64) (*DeleteResp, error) {
	httpReq, err := a.newRequest(http.MethodDelete, fmt.Sprintf("webhooks/%d", webhookID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}
	var data DeleteResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
