package postmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// ---- ListWebhooks --------------------------------------------------------------

func TestListWebhooks_Success(t *testing.T) {
	want := ListWebhooksResp{
		Webhooks: []WebhookResp{
			{ID: 1, Url: "https://example.com/hook1", MessageStream: "outbound"},
			{ID: 2, Url: "https://example.com/hook2", MessageStream: "outbound"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/webhooks") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListWebhooks("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Webhooks) != 2 {
		t.Fatalf("expected 2 webhooks, got %d", len(got.Webhooks))
	}
	if got.Webhooks[0].ID != 1 {
		t.Errorf("Webhooks[0].ID = %d, want 1", got.Webhooks[0].ID)
	}
}

func TestListWebhooks_WithMessageStream(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("MessageStream") != "outbound" {
			t.Errorf("expected MessageStream=outbound query param, got: %s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, ListWebhooksResp{}),
		}, nil
	})))

	_, err := api.ListWebhooks("outbound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListWebhooks_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.ListWebhooks("")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetWebhook ----------------------------------------------------------------

func TestGetWebhook_Success(t *testing.T) {
	want := WebhookResp{
		ID:            42,
		Url:           "https://example.com/hook",
		MessageStream: "outbound",
		Triggers: WebhookTriggers{
			Open:   WebhookTriggerOpen{Enabled: true, PostFirstOpenOnly: false},
			Bounce: WebhookTriggerBounce{Enabled: true, IncludeContent: false},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/webhooks/42") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetWebhook(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("ID = %d, want 42", got.ID)
	}
	if got.Url != "https://example.com/hook" {
		t.Errorf("Url = %q, want https://example.com/hook", got.Url)
	}
}

func TestGetWebhook_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Webhook not found"}),
		}, nil
	})))

	_, err := api.GetWebhook(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- CreateWebhook -------------------------------------------------------------

func TestCreateWebhook_Success(t *testing.T) {
	want := WebhookResp{
		ID:            10,
		Url:           "https://example.com/new-hook",
		MessageStream: "outbound",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/webhooks") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		// Assert the correct URL and MessageStream were serialised in the request body.
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var sentReq CreateWebhookReq
		if err := json.Unmarshal(bodyBytes, &sentReq); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if sentReq.Url != "https://example.com/new-hook" {
			t.Errorf("request body Url = %q, want https://example.com/new-hook", sentReq.Url)
		}
		if sentReq.MessageStream != "outbound" {
			t.Errorf("request body MessageStream = %q, want outbound", sentReq.MessageStream)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	body := &CreateWebhookReq{
		Url:           "https://example.com/new-hook",
		MessageStream: "outbound",
		Triggers: WebhookTriggers{
			Open: WebhookTriggerOpen{Enabled: true},
		},
	}
	got, err := api.CreateWebhook(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 10 {
		t.Errorf("ID = %d, want 10", got.ID)
	}
	if got.Url != "https://example.com/new-hook" {
		t.Errorf("Url = %q", got.Url)
	}
}

func TestCreateWebhook_WithHTTPAuth(t *testing.T) {
	want := WebhookResp{
		ID:            11,
		Url:           "https://example.com/secure-hook",
		MessageStream: "outbound",
		HttpAuth:      &WebhookHTTPAuth{Username: "user", Password: "pass"},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	body := &CreateWebhookReq{
		Url:           "https://example.com/secure-hook",
		MessageStream: "outbound",
		HttpAuth:      &WebhookHTTPAuth{Username: "user", Password: "pass"},
	}
	got, err := api.CreateWebhook(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.HttpAuth == nil {
		t.Fatal("expected HttpAuth to be set")
	}
	if got.HttpAuth.Username != "user" {
		t.Errorf("Username = %q, want user", got.HttpAuth.Username)
	}
}

func TestCreateWebhook_WithHTTPHeaders(t *testing.T) {
	want := WebhookResp{
		ID:  12,
		Url: "https://example.com/hook",
		HttpHeaders: []WebhookHTTPHeader{
			{Name: "X-Custom-Header", Value: "custom-value"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	body := &CreateWebhookReq{
		Url: "https://example.com/hook",
		HttpHeaders: []WebhookHTTPHeader{
			{Name: "X-Custom-Header", Value: "custom-value"},
		},
	}
	got, err := api.CreateWebhook(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.HttpHeaders) != 1 {
		t.Fatalf("expected 1 header, got %d", len(got.HttpHeaders))
	}
	if got.HttpHeaders[0].Name != "X-Custom-Header" {
		t.Errorf("Header Name = %q, want X-Custom-Header", got.HttpHeaders[0].Name)
	}
}

func TestCreateWebhook_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 400, Message: "invalid url"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateWebhook(&CreateWebhookReq{Url: "not-a-url"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- EditWebhook ---------------------------------------------------------------

func TestEditWebhook_Success(t *testing.T) {
	want := WebhookResp{
		ID:  42,
		Url: "https://example.com/updated-hook",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/webhooks/42") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		// Assert the correct URL was serialised in the request body.
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var sentReq EditWebhookReq
		if err := json.Unmarshal(bodyBytes, &sentReq); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if sentReq.Url != "https://example.com/updated-hook" {
			t.Errorf("request body Url = %q, want https://example.com/updated-hook", sentReq.Url)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	body := &EditWebhookReq{
		Url: "https://example.com/updated-hook",
		Triggers: WebhookTriggers{
			Click: WebhookTriggerClick{Enabled: true},
		},
	}
	got, err := api.EditWebhook(42, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Url != "https://example.com/updated-hook" {
		t.Errorf("Url = %q, want https://example.com/updated-hook", got.Url)
	}
}

func TestEditWebhook_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Webhook not found"}),
		}, nil
	})))

	_, err := api.EditWebhook(9999, &EditWebhookReq{})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestEditWebhook_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 400, Message: "invalid webhook data"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.EditWebhook(42, &EditWebhookReq{Url: "not-a-url"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- DeleteWebhook -------------------------------------------------------------

func TestDeleteWebhook_Success(t *testing.T) {
	want := DeleteResp{Message: "Webhook deleted."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, fmt.Sprintf("/webhooks/%d", 55)) {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteWebhook(55)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Webhook deleted." {
		t.Errorf("Message = %q, want 'Webhook deleted.'", got.Message)
	}
}

func TestDeleteWebhook_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Webhook not found"}),
		}, nil
	})))

	_, err := api.DeleteWebhook(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestDeleteWebhook_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.DeleteWebhook(42)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- WebhookTriggers coverage --------------------------------------------------

func TestWebhookTriggers_AllFields(t *testing.T) {
	want := WebhookResp{
		ID:  99,
		Url: "https://example.com/all-triggers",
		Triggers: WebhookTriggers{
			Open:               WebhookTriggerOpen{Enabled: true, PostFirstOpenOnly: true},
			Click:              WebhookTriggerClick{Enabled: true},
			Delivery:           WebhookTriggerDelivery{Enabled: true},
			Bounce:             WebhookTriggerBounce{Enabled: true, IncludeContent: true},
			SpamComplaint:      WebhookTriggerSpamComplaint{Enabled: true, IncludeContent: true},
			SubscriptionChange: WebhookTriggerSubscriptionChange{Enabled: true},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetWebhook(99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Triggers.Open.Enabled {
		t.Error("expected Open.Enabled to be true")
	}
	if !got.Triggers.Open.PostFirstOpenOnly {
		t.Error("expected Open.PostFirstOpenOnly to be true")
	}
	if !got.Triggers.Click.Enabled {
		t.Error("expected Click.Enabled to be true")
	}
	if !got.Triggers.Delivery.Enabled {
		t.Error("expected Delivery.Enabled to be true")
	}
	if !got.Triggers.Bounce.Enabled {
		t.Error("expected Bounce.Enabled to be true")
	}
	if !got.Triggers.Bounce.IncludeContent {
		t.Error("expected Bounce.IncludeContent to be true")
	}
	if !got.Triggers.SpamComplaint.Enabled {
		t.Error("expected SpamComplaint.Enabled to be true")
	}
	if !got.Triggers.SpamComplaint.IncludeContent {
		t.Error("expected SpamComplaint.IncludeContent to be true")
	}
	if !got.Triggers.SubscriptionChange.Enabled {
		t.Error("expected SubscriptionChange.Enabled to be true")
	}
}
