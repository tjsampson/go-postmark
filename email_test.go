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

// ---- ServerTokenOpt ------------------------------------------------------------

func TestNew_WithServerTokenOpt(t *testing.T) {
	api := New(ServerTokenOpt("srv-tok-abc"))
	if api.serverToken != "srv-tok-abc" {
		t.Errorf("expected serverToken srv-tok-abc, got %s", api.serverToken)
	}
}

// ---- SendEmail -----------------------------------------------------------------

func TestSendEmail_Success(t *testing.T) {
	want := SendEmailResp{
		To:          "recipient@example.com",
		SubmittedAt: "2024-01-15T10:00:00Z",
		MessageID:   "msg-id-001",
		ErrorCode:   0,
		Message:     "OK",
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			// Verify HTTP method and path.
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/email") || strings.Contains(req.URL.Path, "batch") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			// Verify server token header.
			if got := req.Header.Get("X-Postmark-Server-Token"); got != "test-server-token" {
				t.Errorf("expected X-Postmark-Server-Token=test-server-token, got %s", got)
			}
			// Account token header must NOT be set for email endpoints.
			if got := req.Header.Get("X-Postmark-Account-Token"); got != "" {
				t.Errorf("X-Postmark-Account-Token should not be set, got %s", got)
			}
			// Verify request body is valid JSON containing the From field.
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			var parsed SendEmailReq
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("invalid request body JSON: %v", err)
			}
			if parsed.From != "sender@example.com" {
				t.Errorf("From = %q, want sender@example.com", parsed.From)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.SendEmail(&SendEmailReq{
		From:    "sender@example.com",
		To:      "recipient@example.com",
		Subject: "Hello",
		TextBody: "Hello, world!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MessageID != want.MessageID {
		t.Errorf("MessageID = %q, want %q", got.MessageID, want.MessageID)
	}
	if got.To != want.To {
		t.Errorf("To = %q, want %q", got.To, want.To)
	}
	if got.ErrorCode != 0 {
		t.Errorf("ErrorCode = %d, want 0", got.ErrorCode)
	}
}

func TestSendEmail_AllFields(t *testing.T) {
	want := SendEmailResp{
		To:          "to@example.com",
		SubmittedAt: "2024-06-01T00:00:00Z",
		MessageID:   "full-msg-id",
		ErrorCode:   0,
		Message:     "OK",
	}

	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			var parsed SendEmailReq
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("invalid request body: %v", err)
			}
			if parsed.Cc != "cc@example.com" {
				t.Errorf("Cc = %q, want cc@example.com", parsed.Cc)
			}
			if parsed.Bcc != "bcc@example.com" {
				t.Errorf("Bcc = %q, want bcc@example.com", parsed.Bcc)
			}
			if parsed.Tag != "welcome" {
				t.Errorf("Tag = %q, want welcome", parsed.Tag)
			}
			if len(parsed.Headers) != 1 || parsed.Headers[0].Name != "X-Custom" {
				t.Errorf("unexpected Headers: %+v", parsed.Headers)
			}
			if len(parsed.Attachments) != 1 || parsed.Attachments[0].Name != "doc.pdf" {
				t.Errorf("unexpected Attachments: %+v", parsed.Attachments)
			}
			if parsed.Metadata["key"] != "value" {
				t.Errorf("unexpected Metadata: %+v", parsed.Metadata)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.SendEmail(&SendEmailReq{
		From:          "sender@example.com",
		To:            "to@example.com",
		Cc:            "cc@example.com",
		Bcc:           "bcc@example.com",
		Subject:       "Test",
		HtmlBody:      "<b>Hello</b>",
		TextBody:      "Hello",
		ReplyTo:       "reply@example.com",
		Headers:       []EmailHeader{{Name: "X-Custom", Value: "val"}},
		TrackOpens:    true,
		TrackLinks:    "None",
		MessageStream: "outbound",
		Attachments:   []Attachment{{Name: "doc.pdf", Content: "dGVzdA==", ContentType: "application/pdf"}},
		Tag:           "welcome",
		Metadata:      map[string]string{"key": "value"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MessageID != "full-msg-id" {
		t.Errorf("MessageID = %q, want full-msg-id", got.MessageID)
	}
}

func TestSendEmail_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 300, Message: "Invalid email request"}

	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{From: "bad"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSendEmail_TransportError(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("connection refused")
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{From: "sender@example.com", To: "to@example.com"})
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- SendEmailBatch ------------------------------------------------------------

func TestSendEmailBatch_Success(t *testing.T) {
	wantResponses := []SendEmailResp{
		{To: "a@example.com", MessageID: "id-1", ErrorCode: 0, Message: "OK"},
		{To: "b@example.com", MessageID: "id-2", ErrorCode: 0, Message: "OK"},
	}

	api := New(
		ServerTokenOpt("batch-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			// Verify HTTP method and path.
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/email/batch") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			// Verify server token header.
			if got := req.Header.Get("X-Postmark-Server-Token"); got != "batch-server-token" {
				t.Errorf("expected X-Postmark-Server-Token=batch-server-token, got %s", got)
			}
			// Account token header must NOT be set.
			if got := req.Header.Get("X-Postmark-Account-Token"); got != "" {
				t.Errorf("X-Postmark-Account-Token should not be set, got %s", got)
			}
			// Verify request body is an array.
			body, _ := io.ReadAll(req.Body)
			var parsed []SendEmailReq
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("invalid batch request body: %v", err)
			}
			if len(parsed) != 2 {
				t.Errorf("expected 2 email requests, got %d", len(parsed))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, wantResponses),
			}, nil
		})),
	)

	reqs := []SendEmailReq{
		{From: "sender@example.com", To: "a@example.com", Subject: "Msg 1"},
		{From: "sender@example.com", To: "b@example.com", Subject: "Msg 2"},
	}
	got, err := api.SendEmailBatch(reqs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(got))
	}
	if got[0].MessageID != "id-1" {
		t.Errorf("got[0].MessageID = %q, want id-1", got[0].MessageID)
	}
	if got[1].MessageID != "id-2" {
		t.Errorf("got[1].MessageID = %q, want id-2", got[1].MessageID)
	}
}

func TestSendEmailBatch_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "Internal server error"}

	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendEmailBatch([]SendEmailReq{{From: "sender@example.com"}})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSendEmailBatch_TransportError(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("dial tcp: connection refused")
		})),
	)

	_, err := api.SendEmailBatch([]SendEmailReq{{From: "sender@example.com"}})
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
}

func TestSendEmailBatch_Empty(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, []SendEmailResp{}),
			}, nil
		})),
	)

	got, err := api.SendEmailBatch([]SendEmailReq{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 responses, got %d", len(got))
	}
}

// ---- ServerTokenOpt env fallback -----------------------------------------------

func TestSendEmail_UsesServerTokenEnvFallback(t *testing.T) {
	t.Setenv("POSTMARK_SERVER_TOKEN", "env-server-token")

	api := New(
		// No ServerTokenOpt — should fall back to env var.
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("X-Postmark-Server-Token"); got != "env-server-token" {
				t.Errorf("expected X-Postmark-Server-Token=env-server-token, got %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{Message: "OK"}),
			}, nil
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{From: "a@b.com", To: "c@d.com", Subject: "env test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- Verify account-token endpoints still work ---------------------------------

// TestSendEmail_DoesNotUseAccountToken confirms the email endpoint sets
// X-Postmark-Server-Token and not X-Postmark-Account-Token.
func TestSendEmail_DoesNotUseAccountToken(t *testing.T) {
	api := New(
		APITokenOpt("account-token-xyz"),
		ServerTokenOpt("server-token-xyz"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			acct := req.Header.Get("X-Postmark-Account-Token")
			srv := req.Header.Get("X-Postmark-Server-Token")
			if acct != "" {
				t.Errorf("X-Postmark-Account-Token should not be set on email endpoint, got %q", acct)
			}
			if srv != "server-token-xyz" {
				t.Errorf("X-Postmark-Server-Token = %q, want server-token-xyz", srv)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{Message: "OK"}),
			}, nil
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{From: "a@b.com", To: "c@d.com", Subject: "token check"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSendEmailBatch_NotFound asserts that a 404 response returns ErrNotFound.
func TestSendEmailBatch_NotFound(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Not found"}),
			}, nil
		})),
	)

	_, err := api.SendEmailBatch([]SendEmailReq{{From: "a@b.com"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
