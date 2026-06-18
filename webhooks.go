package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type (
	// WebhookHttpAuth holds the HTTP Basic Authentication credentials for a webhook.
	WebhookHttpAuth struct {
		Username string `json:"Username"`
		Password string `json:"Password"`
	}

	// WebhookHeader represents a custom HTTP header sent with each webhook request.
	WebhookHeader struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}

	// WebhookTriggerOpen configures the Open event trigger for a webhook.
	WebhookTriggerOpen struct {
		Enabled            bool `json:"Enabled"`
		PostFirstOpenOnly  bool `json:"PostFirstOpenOnly"`
	}

	// WebhookTriggerClick configures the Click event trigger for a webhook.
	WebhookTriggerClick struct {
		Enabled bool `json:"Enabled"`
	}

	// WebhookTriggerDelivery configures the Delivery event trigger for a webhook.
	WebhookTriggerDelivery struct {
		Enabled bool `json:"Enabled"`
	}

	// WebhookTriggerBounce configures the Bounce event trigger for a webhook.
	WebhookTriggerBounce struct {
		Enabled        bool `json:"Enabled"`
		IncludeContent bool `json:"IncludeContent"`
	}

	// WebhookTriggerSpamComplaint configures the SpamComplaint event trigger for a webhook.
	WebhookTriggerSpamComplaint struct {
		Enabled        bool `json:"Enabled"`
		IncludeContent bool `json:"IncludeContent"`
	}

	// WebhookTriggerSubscriptionChange configures the SubscriptionChange event trigger for a webhook.
	WebhookTriggerSubscriptionChange struct {
		Enabled bool `json:"Enabled"`
	}

	// WebhookTriggers groups all the event triggers for a webhook.
	WebhookTriggers struct {
		Open               WebhookTriggerOpen               `json:"Open"`
		Click              WebhookTriggerClick              `json:"Click"`
		Delivery           WebhookTriggerDelivery           `json:"Delivery"`
		Bounce             WebhookTriggerBounce             `json:"Bounce"`
		SpamComplaint      WebhookTriggerSpamComplaint      `json:"SpamComplaint"`
		SubscriptionChange WebhookTriggerSubscriptionChange `json:"SubscriptionChange"`
	}

	// WebhookReq is the request body for creating or updating a webhook.
	WebhookReq struct {
		Url           string           `json:"Url"`
		MessageStream string           `json:"MessageStream"`
		HttpAuth      *WebhookHttpAuth `json:"HttpAuth,omitempty"`
		Headers       []WebhookHeader  `json:"Headers,omitempty"`
		Triggers      WebhookTriggers  `json:"Triggers"`
	}

	// WebhookResp represents a Postmark webhook as returned by the API.
	WebhookResp struct {
		ID            int              `json:"ID"`
		Url           string           `json:"Url"`
		MessageStream string           `json:"MessageStream"`
		HttpAuth      *WebhookHttpAuth `json:"HttpAuth,omitempty"`
		Headers       []WebhookHeader  `json:"Headers,omitempty"`
		Triggers      WebhookTriggers  `json:"Triggers"`
	}

	// ListWebhooksResp is the response envelope returned by the list-webhooks endpoint.
	ListWebhooksResp struct {
		Webhooks []WebhookResp `json:"Webhooks"`
	}
)

// newServerRequest builds an *http.Request for the given HTTP method and API
// path, using the X-Postmark-Server-Token header required by the Webhooks API.
func (a *API) newServerRequest(method, path string, body interface{}) (*http.Request, error) {
	req, err := a.newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	// Webhooks API requires the server token, not the account token.
	req.Header.Del("X-Postmark-Account-Token")
	req.Header.Set("X-Postmark-Server-Token", a.token)
	return req, nil
}

// ListWebhooks returns all webhooks configured for the given message stream.
// Pass an empty string to retrieve webhooks across all message streams.
func (a *API) ListWebhooks(messageStream string) (*ListWebhooksResp, error) {
	params := url.Values{}
	if messageStream != "" {
		params.Set("MessageStream", messageStream)
	}
	path := "webhooks"
	if len(params) > 0 {
		path = "webhooks?" + params.Encode()
	}

	req, err := a.newServerRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data ListWebhooksResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetWebhook fetches the webhook identified by id.
func (a *API) GetWebhook(id int) (*WebhookResp, error) {
	req, err := a.newServerRequest(http.MethodGet, fmt.Sprintf("webhooks/%d", id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data WebhookResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateWebhook creates a new webhook with the settings in webhookReq.
// It returns the full WebhookResp on success.
func (a *API) CreateWebhook(webhookReq *WebhookReq) (*WebhookResp, error) {
	req, err := a.newServerRequest(http.MethodPost, "webhooks", webhookReq)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data WebhookResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateWebhook applies the changes in webhookReq to the webhook identified by
// id and returns the updated WebhookResp.
func (a *API) UpdateWebhook(id int, webhookReq *WebhookReq) (*WebhookResp, error) {
	req, err := a.newServerRequest(http.MethodPut, fmt.Sprintf("webhooks/%d", id), webhookReq)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data WebhookResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteWebhook deletes the webhook identified by id.
// It returns a DeleteResp containing the outcome message from the API.
func (a *API) DeleteWebhook(id int) (*DeleteResp, error) {
	req, err := a.newServerRequest(http.MethodDelete, fmt.Sprintf("webhooks/%d", id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data DeleteResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
