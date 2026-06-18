package postmark

import (
	"errors"
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

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/webhooks") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if got := req.Header.Get("X-Postmark-Server-Token"); got != "test-server-token" {
				t.Errorf("X-Postmark-Server-Token = %q, want %q", got, "test-server-token")
			}
			if req.Header.Get("X-Postmark-Account-Token") != "" {
				t.Error("X-Postmark-Account-Token header must not be set for webhooks")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.ListWebhooks("outbound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Webhooks) != 2 {
		t.Errorf("expected 2 webhooks, got %d", len(got.Webhooks))
	}
	if got.Webhooks[0].ID != 1 {
		t.Errorf("expected webhook ID 1, got %d", got.Webhooks[0].ID)
	}
}

func TestListWebhooks_MessageStreamQueryParam(t *testing.T) {
	want := ListWebhooksResp{
		Webhooks: []WebhookResp{
			{ID: 1, Url: "https://example.com/hook1", MessageStream: "outbound"},
		},
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.RawQuery, "MessageStream=outbound") {
				t.Errorf("expected MessageStream query param, query=%s", req.URL.RawQuery)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.ListWebhooks("outbound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Webhooks) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(got.Webhooks))
	}
}

func TestListWebhooks_NoStream(t *testing.T) {
	want := ListWebhooksResp{
		Webhooks: []WebhookResp{
			{ID: 10, Url: "https://example.com/hookA"},
		},
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			// When messageStream is empty, url.Values.Encode() returns "" so
			// len(params) == 0 and no "?" suffix is appended to the path.
			// We verify that the query string is therefore absent on the wire.
			if req.URL.RawQuery != "" {
				t.Errorf("expected empty query string when messageStream is empty, got %q", req.URL.RawQuery)
			}
			if got := req.Header.Get("X-Postmark-Server-Token"); got != "test-server-token" {
				t.Errorf("X-Postmark-Server-Token = %q, want %q", got, "test-server-token")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.ListWebhooks("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Webhooks) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(got.Webhooks))
	}
}

func TestListWebhooks_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.ListWebhooks("outbound")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestListWebhooks_NoServerToken verifies that newServerRequest fails fast with
// an error when no server token has been configured, so the request is never
// sent rather than going out with an empty authentication header.
func TestListWebhooks_NoServerToken(t *testing.T) {
	api := New(
		// Intentionally omit ServerTokenOpt — serverToken remains "".
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP client must not be called when serverToken is empty")
			return nil, nil
		})),
	)

	_, err := api.ListWebhooks("")
	if err == nil {
		t.Fatal("expected an error when serverToken is empty, got nil")
	}
	if !strings.Contains(err.Error(), "serverToken") {
		t.Errorf("expected error to mention serverToken, got: %v", err)
	}
}

// ---- GetWebhook ----------------------------------------------------------------

func TestGetWebhook_Success(t *testing.T) {
	want := WebhookResp{
		ID:            42,
		Url:           "https://example.com/hook",
		MessageStream: "outbound",
		HTTPAuth: &WebhookHTTPAuth{
			Username: "user",
			Password: "pass",
		},
		Headers: []WebhookHeader{
			{Name: "X-Custom", Value: "value"},
		},
		Triggers: WebhookTriggers{
			Open:               &WebhookTriggerOpen{Enabled: true, PostFirstOpenOnly: true},
			Click:              &WebhookTriggerClick{Enabled: true},
			Delivery:           &WebhookTriggerDelivery{Enabled: true},
			Bounce:             &WebhookTriggerBounce{Enabled: true, IncludeContent: false},
			SpamComplaint:      &WebhookTriggerSpamComplaint{Enabled: true, IncludeContent: true},
			SubscriptionChange: &WebhookTriggerSubscriptionChange{Enabled: true},
		},
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/webhooks/42") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if got := req.Header.Get("X-Postmark-Server-Token"); got != "test-server-token" {
				t.Errorf("X-Postmark-Server-Token = %q, want %q", got, "test-server-token")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.GetWebhook(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("expected ID 42, got %d", got.ID)
	}
	if got.Url != want.Url {
		t.Errorf("Url = %q, want %q", got.Url, want.Url)
	}
	if got.HTTPAuth == nil || got.HTTPAuth.Username != "user" {
		t.Errorf("unexpected HTTPAuth: %+v", got.HTTPAuth)
	}
	if len(got.Headers) != 1 || got.Headers[0].Name != "X-Custom" {
		t.Errorf("unexpected Headers: %+v", got.Headers)
	}
	if got.Triggers.Open == nil || !got.Triggers.Open.Enabled {
		t.Error("expected Open trigger to be enabled")
	}
}

func TestGetWebhook_NotFound(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "webhook not found"}),
			}, nil
		})),
	)

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
		ID:            100,
		Url:           "https://example.com/new-hook",
		MessageStream: "outbound",
		Triggers: WebhookTriggers{
			Delivery: &WebhookTriggerDelivery{Enabled: true},
		},
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/webhooks") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if got := req.Header.Get("X-Postmark-Server-Token"); got != "test-server-token" {
				t.Errorf("X-Postmark-Server-Token = %q, want %q", got, "test-server-token")
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.CreateWebhook(&WebhookReq{
		Url:           "https://example.com/new-hook",
		MessageStream: "outbound",
		Triggers: &WebhookTriggers{
			Delivery: &WebhookTriggerDelivery{Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 100 {
		t.Errorf("expected ID 100, got %d", got.ID)
	}
	if got.Url != want.Url {
		t.Errorf("Url = %q, want %q", got.Url, want.Url)
	}
	if got.Triggers.Delivery == nil || !got.Triggers.Delivery.Enabled {
		t.Error("expected Delivery trigger to be enabled")
	}
}

func TestCreateWebhook_NilReq(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP client must not be called when req is nil")
			return nil, nil
		})),
	)

	_, err := api.CreateWebhook(nil)
	if err == nil {
		t.Fatal("expected an error when req is nil, got nil")
	}
}

func TestCreateWebhook_WithAuthAndHeaders(t *testing.T) {
	want := WebhookResp{
		ID:  200,
		Url: "https://example.com/secure-hook",
		HTTPAuth: &WebhookHTTPAuth{
			Username: "admin",
			Password: "secret",
		},
		Headers: []WebhookHeader{
			{Name: "Authorization", Value: "Bearer token"},
		},
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.CreateWebhook(&WebhookReq{
		Url: "https://example.com/secure-hook",
		HTTPAuth: &WebhookHTTPAuth{
			Username: "admin",
			Password: "secret",
		},
		Headers: []WebhookHeader{
			{Name: "Authorization", Value: "Bearer token"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 200 {
		t.Errorf("expected ID 200, got %d", got.ID)
	}
	if got.HTTPAuth == nil || got.HTTPAuth.Username != "admin" {
		t.Errorf("unexpected HTTPAuth: %+v", got.HTTPAuth)
	}
	if len(got.Headers) != 1 || got.Headers[0].Name != "Authorization" {
		t.Errorf("unexpected Headers: %+v", got.Headers)
	}
}

func TestCreateWebhook_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 300, Message: "invalid webhook URL"}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.CreateWebhook(&WebhookReq{Url: "not-a-url"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- UpdateWebhook -------------------------------------------------------------

func TestUpdateWebhook_Success(t *testing.T) {
	want := WebhookResp{
		ID:  42,
		Url: "https://example.com/updated-hook",
		Triggers: WebhookTriggers{
			Open:  &WebhookTriggerOpen{Enabled: true, PostFirstOpenOnly: false},
			Click: &WebhookTriggerClick{Enabled: true},
		},
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPut {
				t.Errorf("expected PUT, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/webhooks/42") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if got := req.Header.Get("X-Postmark-Server-Token"); got != "test-server-token" {
				t.Errorf("X-Postmark-Server-Token = %q, want %q", got, "test-server-token")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.UpdateWebhook(42, &WebhookReq{
		Url: "https://example.com/updated-hook",
		Triggers: &WebhookTriggers{
			Open:  &WebhookTriggerOpen{Enabled: true},
			Click: &WebhookTriggerClick{Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("expected ID 42, got %d", got.ID)
	}
	if got.Url != want.Url {
		t.Errorf("Url = %q, want %q", got.Url, want.Url)
	}
	if got.Triggers.Open == nil || !got.Triggers.Open.Enabled {
		t.Error("expected Open trigger to be enabled")
	}
	if got.Triggers.Click == nil || !got.Triggers.Click.Enabled {
		t.Error("expected Click trigger to be enabled")
	}
}

func TestUpdateWebhook_NilReq(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP client must not be called when req is nil")
			return nil, nil
		})),
	)

	_, err := api.UpdateWebhook(42, nil)
	if err == nil {
		t.Fatal("expected an error when req is nil, got nil")
	}
}

func TestUpdateWebhook_NotFound(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "webhook not found"}),
			}, nil
		})),
	)

	_, err := api.UpdateWebhook(9999, &WebhookReq{Url: "https://example.com/hook"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- DeleteWebhook -------------------------------------------------------------

func TestDeleteWebhook_Success(t *testing.T) {
	want := DeleteResp{Message: "Webhook deleted."}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/webhooks/77") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if got := req.Header.Get("X-Postmark-Server-Token"); got != "test-server-token" {
				t.Errorf("X-Postmark-Server-Token = %q, want %q", got, "test-server-token")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.DeleteWebhook(77)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Webhook deleted." {
		t.Errorf("Message = %q, want %q", got.Message, "Webhook deleted.")
	}
}

func TestDeleteWebhook_NotFound(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "webhook not found"}),
			}, nil
		})),
	)

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

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.DeleteWebhook(1)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- Trigger sub-struct coverage -----------------------------------------------

func TestWebhookTriggers_AllFields(t *testing.T) {
	want := WebhookResp{
		ID:  300,
		Url: "https://example.com/full-trigger-hook",
		Triggers: WebhookTriggers{
			Open:               &WebhookTriggerOpen{Enabled: true, PostFirstOpenOnly: true},
			Click:              &WebhookTriggerClick{Enabled: true},
			Delivery:           &WebhookTriggerDelivery{Enabled: true},
			Bounce:             &WebhookTriggerBounce{Enabled: true, IncludeContent: true},
			SpamComplaint:      &WebhookTriggerSpamComplaint{Enabled: true, IncludeContent: true},
			SubscriptionChange: &WebhookTriggerSubscriptionChange{Enabled: true},
		},
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.CreateWebhook(&WebhookReq{
		Url: "https://example.com/full-trigger-hook",
		Triggers: &WebhookTriggers{
			Open:               &WebhookTriggerOpen{Enabled: true, PostFirstOpenOnly: true},
			Click:              &WebhookTriggerClick{Enabled: true},
			Delivery:           &WebhookTriggerDelivery{Enabled: true},
			Bounce:             &WebhookTriggerBounce{Enabled: true, IncludeContent: true},
			SpamComplaint:      &WebhookTriggerSpamComplaint{Enabled: true, IncludeContent: true},
			SubscriptionChange: &WebhookTriggerSubscriptionChange{Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 300 {
		t.Errorf("expected ID 300, got %d", got.ID)
	}
	if got.Triggers.Open == nil || !got.Triggers.Open.PostFirstOpenOnly {
		t.Error("expected Open.PostFirstOpenOnly to be true")
	}
	if got.Triggers.Bounce == nil || !got.Triggers.Bounce.IncludeContent {
		t.Error("expected Bounce.IncludeContent to be true")
	}
	if got.Triggers.SpamComplaint == nil || !got.Triggers.SpamComplaint.IncludeContent {
		t.Error("expected SpamComplaint.IncludeContent to be true")
	}
	if got.Triggers.SubscriptionChange == nil || !got.Triggers.SubscriptionChange.Enabled {
		t.Error("expected SubscriptionChange.Enabled to be true")
	}
}
