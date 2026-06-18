package postmark

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestWebhooksCRUD exercises ListWebhooks, CreateWebhook, GetWebhook,
// UpdateWebhook, and DeleteWebhook using table-driven success and error sub-cases.
func TestWebhooksCRUD(t *testing.T) {
	webhookResp := WebhookResp{
		ID:            77,
		Url:           "https://example.com/hook77",
		MessageStream: "outbound",
	}
	listResp := ListWebhooksResp{
		Webhooks: []WebhookResp{
			{ID: 1, Url: "https://example.com/hook1", MessageStream: "outbound"},
			{ID: 2, Url: "https://example.com/hook2", MessageStream: "outbound"},
		},
	}
	deleteResp := DeleteResp{Message: "Webhook removed."}

	tests := []struct {
		name           string
		wantMethod     string
		wantPathSuffix string
		statusCode     int
		responseBody   interface{}
		call           func(api *API) (interface{}, error)
		checkOK        func(t *testing.T, got interface{})
	}{
		{
			name:           "ListWebhooks/success_with_stream",
			wantMethod:     http.MethodGet,
			wantPathSuffix: "/webhooks",
			statusCode:     http.StatusOK,
			responseBody:   listResp,
			call:           func(api *API) (interface{}, error) { return api.ListWebhooks("outbound") },
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*ListWebhooksResp)
				if len(r.Webhooks) != 2 {
					t.Errorf("len(Webhooks) = %d, want 2", len(r.Webhooks))
				}
			},
		},
		{
			name:         "ListWebhooks/success_no_filter",
			wantMethod:   http.MethodGet,
			statusCode:   http.StatusOK,
			responseBody: ListWebhooksResp{Webhooks: []WebhookResp{{ID: 1, Url: "https://example.com/hook1"}}},
			call:         func(api *API) (interface{}, error) { return api.ListWebhooks("") },
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*ListWebhooksResp)
				if len(r.Webhooks) != 1 {
					t.Errorf("len(Webhooks) = %d, want 1", len(r.Webhooks))
				}
			},
		},
		{
			name:         "ListWebhooks/error",
			statusCode:   http.StatusInternalServerError,
			responseBody: PostmarkErr{ErrorCode: 500, Message: "internal error"},
			call:         func(api *API) (interface{}, error) { return api.ListWebhooks("") },
		},
		{
			name:           "CreateWebhook/success",
			wantMethod:     http.MethodPost,
			wantPathSuffix: "/webhooks",
			statusCode:     http.StatusOK,
			responseBody:   WebhookResp{ID: 100, Url: "https://example.com/webhook", MessageStream: "outbound"},
			call: func(api *API) (interface{}, error) {
				return api.CreateWebhook(&CreateWebhookReq{
					Url:           "https://example.com/webhook",
					MessageStream: "outbound",
				})
			},
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*WebhookResp)
				if r.ID != 100 || r.Url != "https://example.com/webhook" {
					t.Errorf("got ID=%d Url=%s", r.ID, r.Url)
				}
			},
		},
		{
			name:         "CreateWebhook/error",
			statusCode:   http.StatusUnprocessableEntity,
			responseBody: PostmarkErr{ErrorCode: 422, Message: "invalid url"},
			call: func(api *API) (interface{}, error) {
				return api.CreateWebhook(&CreateWebhookReq{Url: "bad"})
			},
		},
		{
			name:           "GetWebhook/success",
			wantMethod:     http.MethodGet,
			wantPathSuffix: "/webhooks/77",
			statusCode:     http.StatusOK,
			responseBody: WebhookResp{
				ID:  77,
				Url: "https://example.com/hook77",
				Headers: []NameValue{
					{Name: "X-Custom", Value: "header-value"},
				},
			},
			call: func(api *API) (interface{}, error) { return api.GetWebhook(77) },
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*WebhookResp)
				if r.ID != 77 {
					t.Errorf("ID = %d, want 77", r.ID)
				}
				if len(r.Headers) != 1 || r.Headers[0].Name != "X-Custom" {
					t.Errorf("unexpected headers: %+v", r.Headers)
				}
			},
		},
		{
			name:         "GetWebhook/not_found",
			statusCode:   http.StatusNotFound,
			responseBody: PostmarkErr{ErrorCode: 404, Message: "Webhook not found"},
			call:         func(api *API) (interface{}, error) { return api.GetWebhook(9999) },
		},
		{
			name:           "UpdateWebhook/success",
			wantMethod:     http.MethodPut,
			wantPathSuffix: "/webhooks/77",
			statusCode:     http.StatusOK,
			responseBody:   WebhookResp{ID: 77, Url: "https://example.com/hook-updated"},
			call: func(api *API) (interface{}, error) {
				return api.UpdateWebhook(77, &UpdateWebhookReq{Url: "https://example.com/hook-updated"})
			},
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*WebhookResp)
				if r.Url != "https://example.com/hook-updated" {
					t.Errorf("Url = %q", r.Url)
				}
			},
		},
		{
			name:         "UpdateWebhook/not_found",
			statusCode:   http.StatusNotFound,
			responseBody: PostmarkErr{ErrorCode: 404, Message: "Webhook not found"},
			call: func(api *API) (interface{}, error) {
				return api.UpdateWebhook(9999, &UpdateWebhookReq{Url: "https://example.com/hook"})
			},
		},
		{
			name:           "DeleteWebhook/success",
			wantMethod:     http.MethodDelete,
			wantPathSuffix: "/webhooks/22",
			statusCode:     http.StatusOK,
			responseBody:   deleteResp,
			call:           func(api *API) (interface{}, error) { return api.DeleteWebhook(22) },
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*DeleteResp)
				if r.Message != "Webhook removed." {
					t.Errorf("Message = %q", r.Message)
				}
			},
		},
		{
			name:         "DeleteWebhook/not_found",
			statusCode:   http.StatusNotFound,
			responseBody: PostmarkErr{ErrorCode: 404, Message: "Webhook not found"},
			call:         func(api *API) (interface{}, error) { return api.DeleteWebhook(9999) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isError := tc.statusCode >= http.StatusBadRequest

			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				if tc.wantMethod != "" && req.Method != tc.wantMethod {
					t.Errorf("method = %s, want %s", req.Method, tc.wantMethod)
				}
				if tc.wantPathSuffix != "" && !strings.HasSuffix(req.URL.Path, tc.wantPathSuffix) {
					t.Errorf("path = %s, want suffix %s", req.URL.Path, tc.wantPathSuffix)
				}
				return &http.Response{
					StatusCode: tc.statusCode,
					Body:       jsonBody(t, tc.responseBody),
				}, nil
			})))

			got, err := tc.call(api)
			if isError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.statusCode == http.StatusNotFound && !errors.Is(err, ErrNotFound) {
					t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.checkOK != nil {
					tc.checkOK(t, got)
				}
			}
		})
	}

	// Extra assertion: ListWebhooks with a stream filter must send MessageStream query param.
	t.Run("ListWebhooks/stream_query_param", func(t *testing.T) {
		api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.RawQuery, "MessageStream=outbound") {
				t.Errorf("expected MessageStream param, query=%s", req.URL.RawQuery)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, listResp),
			}, nil
		})))
		if _, err := api.ListWebhooks("outbound"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Extra assertion: ListWebhooks with no stream must send no query params.
	t.Run("ListWebhooks/no_query_params", func(t *testing.T) {
		api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.URL.RawQuery != "" {
				t.Errorf("expected no query params, got %s", req.URL.RawQuery)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, webhookResp),
			}, nil
		})))
		if _, err := api.ListWebhooks(""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestWebhookResp_WithTriggers verifies that trigger and HttpAuth fields are
// correctly deserialized from an API response.
func TestWebhookResp_WithTriggers(t *testing.T) {
	want := WebhookResp{
		ID:  200,
		Url: "https://example.com/hook",
		Triggers: WebhookTriggers{
			Open:   WebhookTriggerOpen{Enabled: true, PostFirstOpenOnly: false},
			Click:  WebhookTriggerClick{Enabled: true},
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

// TestWebhooks_UnmarshalError verifies that a malformed JSON response body
// causes webhook methods to return a non-nil error.
func TestWebhooks_UnmarshalError(t *testing.T) {
	tests := []struct {
		name string
		call func(api *API) (interface{}, error)
	}{
		{
			name: "ListWebhooks",
			call: func(api *API) (interface{}, error) { return api.ListWebhooks("") },
		},
		{
			name: "CreateWebhook",
			call: func(api *API) (interface{}, error) {
				return api.CreateWebhook(&CreateWebhookReq{Url: "https://example.com/hook"})
			},
		},
		{
			name: "GetWebhook",
			call: func(api *API) (interface{}, error) { return api.GetWebhook(1) },
		},
		{
			name: "UpdateWebhook",
			call: func(api *API) (interface{}, error) {
				return api.UpdateWebhook(1, &UpdateWebhookReq{Url: "https://example.com/hook"})
			},
		},
		{
			name: "DeleteWebhook",
			call: func(api *API) (interface{}, error) { return api.DeleteWebhook(1) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{not valid json`)),
				}, nil
			})))

			_, err := tc.call(api)
			if err == nil {
				t.Fatal("expected unmarshal error, got nil")
			}
		})
	}
}
