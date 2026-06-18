package postmark

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

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
		if !strings.Contains(req.URL.RawQuery, "MessageStream=outbound") {
			t.Errorf("expected MessageStream param, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListWebhooks("outbound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Webhooks) != 2 {
		t.Errorf("len(Webhooks) = %d, want 2", len(got.Webhooks))
	}
}

func TestListWebhooks_NoFilter(t *testing.T) {
	want := ListWebhooksResp{
		Webhooks: []WebhookResp{
			{ID: 1, Url: "https://example.com/hook1"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %s", req.URL.RawQuery)
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
	if len(got.Webhooks) != 1 {
		t.Errorf("len(Webhooks) = %d, want 1", len(got.Webhooks))
	}
}

func TestListWebhooks_Error(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

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

func TestCreateWebhook_Success(t *testing.T) {
	want := WebhookResp{
		ID:            100,
		Url:           "https://example.com/webhook",
		MessageStream: "outbound",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/webhooks") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.CreateWebhook(&CreateWebhookReq{
		Url:           "https://example.com/webhook",
		MessageStream: "outbound",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 100 || got.Url != "https://example.com/webhook" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCreateWebhook_Error(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 422, Message: "invalid url"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateWebhook(&CreateWebhookReq{Url: "bad"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestGetWebhook_Success(t *testing.T) {
	want := WebhookResp{
		ID:  77,
		Url: "https://example.com/hook77",
		Headers: []NameValue{
			{Name: "X-Custom", Value: "header-value"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/webhooks/77") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetWebhook(77)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 77 {
		t.Errorf("ID = %d, want 77", got.ID)
	}
	if len(got.Headers) != 1 || got.Headers[0].Name != "X-Custom" {
		t.Errorf("unexpected headers: %+v", got.Headers)
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
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

func TestUpdateWebhook_Success(t *testing.T) {
	want := WebhookResp{
		ID:  77,
		Url: "https://example.com/hook-updated",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/webhooks/77") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.UpdateWebhook(77, &CreateWebhookReq{Url: "https://example.com/hook-updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Url != "https://example.com/hook-updated" {
		t.Errorf("Url = %q", got.Url)
	}
}

func TestUpdateWebhook_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Webhook not found"}),
		}, nil
	})))

	_, err := api.UpdateWebhook(9999, &CreateWebhookReq{Url: "https://example.com/hook"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

func TestDeleteWebhook_Success(t *testing.T) {
	want := DeleteResp{Message: "Webhook removed."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/webhooks/22") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteWebhook(22)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Webhook removed." {
		t.Errorf("Message = %q", got.Message)
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
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

// TestWebhookResp_WithTriggers verifies that trigger fields are correctly
// serialized and deserialized.
func TestWebhookResp_WithTriggers(t *testing.T) {
	want := WebhookResp{
		ID:  200,
		Url: "https://example.com/hook",
		Triggers: WebhookTriggers{
			Open:  WebhookTriggerOpen{Enabled: true, PostFirstOpenOnly: false},
			Click: WebhookTriggerClick{Enabled: true},
			Bounce: WebhookTriggerBounce{Enabled: true, IncludeContent: true},
		},
		HttpAuth: WebhookHttpAuth{Username: "user", Password: "pass"},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetWebhook(200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Triggers.Open.Enabled {
		t.Error("expected Triggers.Open.Enabled to be true")
	}
	if !got.Triggers.Click.Enabled {
		t.Error("expected Triggers.Click.Enabled to be true")
	}
	if !got.Triggers.Bounce.IncludeContent {
		t.Error("expected Triggers.Bounce.IncludeContent to be true")
	}
	if got.HttpAuth.Username != "user" {
		t.Errorf("HttpAuth.Username = %q, want user", got.HttpAuth.Username)
	}
}
