package postmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type (
	// WebhookHTTPAuth holds the HTTP Basic Authentication credentials for a webhook.
	//
	// Security note: the Password field may contain a plaintext credential as
	// returned by the Postmark API. Callers must not log, serialise, or expose
	// WebhookHTTPAuth (or any struct that embeds it) in debug endpoints or
	// structured log output without first redacting this field.
	WebhookHTTPAuth struct {
		Username string `json:"Username"`
		// Password is omitted from JSON output when empty so that a zero-value
		// WebhookHTTPAuth is not accidentally re-serialised with an empty password
		// field. Callers must treat this value as a secret.
		Password string `json:"Password,omitempty"`
	}

	// WebhookHeader represents a custom HTTP header sent with each webhook request.
	WebhookHeader struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}

	// WebhookTriggerOpen configures the Open event trigger for a webhook.
	WebhookTriggerOpen struct {
		Enabled           bool `json:"Enabled"`
		PostFirstOpenOnly bool `json:"PostFirstOpenOnly"`
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
	// Pointer fields with omitempty allow partial-update semantics on PUT: only
	// triggers explicitly provided by the caller are serialised; triggers left as
	// nil are omitted from the JSON body and therefore left unchanged by the API.
	WebhookTriggers struct {
		Open               *WebhookTriggerOpen               `json:"Open,omitempty"`
		Click              *WebhookTriggerClick              `json:"Click,omitempty"`
		Delivery           *WebhookTriggerDelivery           `json:"Delivery,omitempty"`
		Bounce             *WebhookTriggerBounce             `json:"Bounce,omitempty"`
		SpamComplaint      *WebhookTriggerSpamComplaint      `json:"SpamComplaint,omitempty"`
		SubscriptionChange *WebhookTriggerSubscriptionChange `json:"SubscriptionChange,omitempty"`
	}

	// WebhookReq is the request body for creating or updating a webhook.
	// Triggers is a pointer so that a caller creating a bare URL-only webhook can
	// omit the Triggers object entirely (nil pointer + omitempty → not serialised).
	WebhookReq struct {
		Url           string           `json:"Url"`
		MessageStream string           `json:"MessageStream"`
		HTTPAuth      *WebhookHTTPAuth `json:"HttpAuth,omitempty"`
		Headers       []WebhookHeader  `json:"Headers,omitempty"`
		Triggers      *WebhookTriggers `json:"Triggers,omitempty"`
	}

	// WebhookResp represents a Postmark webhook as returned by the API.
	WebhookResp struct {
		ID            int              `json:"ID"`
		Url           string           `json:"Url"`
		MessageStream string           `json:"MessageStream"`
		HTTPAuth      *WebhookHTTPAuth `json:"HttpAuth,omitempty"`
		Headers       []WebhookHeader  `json:"Headers,omitempty"`
		Triggers      WebhookTriggers  `json:"Triggers"`
	}

	// ListWebhooksResp is the response envelope returned by the list-webhooks endpoint.
	ListWebhooksResp struct {
		Webhooks []WebhookResp `json:"Webhooks"`
	}
)

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

// CreateWebhook creates a new webhook with the settings in req.
// It returns the full WebhookResp on success.
// req must not be nil; if it is, an error is returned immediately.
func (a *API) CreateWebhook(req *WebhookReq) (*WebhookResp, error) {
	if req == nil {
		return nil, errors.New("req must not be nil")
	}
	httpReq, err := a.newServerRequest(http.MethodPost, "webhooks", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data WebhookResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateWebhook applies the changes in req to the webhook identified by id
// and returns the updated WebhookResp.
// req must not be nil; if it is, an error is returned immediately.
func (a *API) UpdateWebhook(id int, req *WebhookReq) (*WebhookResp, error) {
	if req == nil {
		return nil, errors.New("req must not be nil")
	}
	httpReq, err := a.newServerRequest(http.MethodPut, fmt.Sprintf("webhooks/%d", id), req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
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
