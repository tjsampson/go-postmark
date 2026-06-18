package postmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// boolPtr is a helper that returns a pointer to the given bool value,
// needed for SendEmailReq.TrackOpens which is *bool.
func boolPtr(b bool) *bool { return &b }

// ---- ServerTokenOpt ------------------------------------------------------------

func TestNew_WithServerTokenOpt(t *testing.T) {
	api := New(ServerTokenOpt("srv-tok-abc"))
	if api.serverToken != "srv-tok-abc" {
		t.Errorf("expected serverToken srv-tok-abc, got %s", api.serverToken)
	}
}

// ---- SendEmail -----------------------------------------------------------------

func TestSendEmail_Success(t *testing.T) {
	submittedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	want := SendEmailResp{
		To:          "recipient@example.com",
		SubmittedAt: submittedAt,
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
		From:     "sender@example.com",
		To:       "recipient@example.com",
		Subject:  "Hello",
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
	if !got.SubmittedAt.Equal(want.SubmittedAt) {
		t.Errorf("SubmittedAt = %v, want %v", got.SubmittedAt, want.SubmittedAt)
	}
	if got.ErrorCode != 0 {
		t.Errorf("ErrorCode = %d, want 0", got.ErrorCode)
	}
}

func TestSendEmail_AllFields(t *testing.T) {
	want := SendEmailResp{
		To:          "to@example.com",
		SubmittedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
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
			// TrackOpens must be present and true.
			if parsed.TrackOpens == nil || !*parsed.TrackOpens {
				t.Errorf("expected TrackOpens=true, got %v", parsed.TrackOpens)
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
		TrackOpens:    boolPtr(true),
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

// TestSendEmail_TrackOpensFalse verifies that explicitly setting TrackOpens to
// false sends the field as false in the JSON body (not omitted), so callers can
// disable open tracking on streams where it is enabled by default.
func TestSendEmail_TrackOpensFalse(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			// Unmarshal into a raw map to distinguish absent from false.
			var raw map[string]interface{}
			if err := json.Unmarshal(body, &raw); err != nil {
				t.Fatalf("invalid request body: %v", err)
			}
			val, present := raw["TrackOpens"]
			if !present {
				t.Error("expected TrackOpens to be present in JSON body, but it was omitted")
			}
			if present && val != false {
				t.Errorf("expected TrackOpens=false, got %v", val)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{Message: "OK"}),
			}, nil
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{
		From:       "a@b.com",
		To:         "c@d.com",
		Subject:    "track off",
		TextBody:   "hi",
		TrackOpens: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSendEmail_TrackOpensNil verifies that leaving TrackOpens nil omits the
// field from the JSON body entirely (server default applies).
func TestSendEmail_TrackOpensNil(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			var raw map[string]interface{}
			if err := json.Unmarshal(body, &raw); err != nil {
				t.Fatalf("invalid request body: %v", err)
			}
			if _, present := raw["TrackOpens"]; present {
				t.Error("expected TrackOpens to be absent from JSON body when nil, but it was present")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{Message: "OK"}),
			}, nil
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{
		From:     "a@b.com",
		To:       "c@d.com",
		Subject:  "no track field",
		TextBody: "hi",
		// TrackOpens is nil — omitempty should suppress the field.
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSendEmail_MessageStream verifies that setting MessageStream to a
// non-default value sends it in the request body so Postmark routes the
// message through the correct stream.
func TestSendEmail_MessageStream(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			var parsed SendEmailReq
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("invalid request body: %v", err)
			}
			if parsed.MessageStream != "broadcasts" {
				t.Errorf("MessageStream = %q, want broadcasts", parsed.MessageStream)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{Message: "OK"}),
			}, nil
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{
		From:          "a@b.com",
		To:            "c@d.com",
		Subject:       "broadcast",
		TextBody:      "hi",
		MessageStream: "broadcasts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
	// Assert error is a PostmarkErr with the correct code.
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Fatalf("expected error to be PostmarkErr, got %T: %v", err, err)
	}
	if pe.ErrorCode != 300 {
		t.Errorf("ErrorCode = %d, want 300", pe.ErrorCode)
	}
	if pe.Message != "Invalid email request" {
		t.Errorf("Message = %q, want %q", pe.Message, "Invalid email request")
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

// TestSendEmail_NilRequest verifies that passing a nil *SendEmailReq to
// SendEmail returns an error rather than sending a null JSON body.
func TestSendEmail_NilRequest(t *testing.T) {
	api := New(ServerTokenOpt("srv-tok"))

	_, err := api.SendEmail(nil)
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
}

// TestSendEmail_EmptyServerToken verifies that calling SendEmail when no server
// token is configured (neither ServerTokenOpt nor POSTMARK_SERVER_TOKEN) returns
// an error rather than sending an unauthenticated request.
func TestSendEmail_EmptyServerToken(t *testing.T) {
	// Ensure the env var is unset for this test.
	t.Setenv("POSTMARK_SERVER_TOKEN", "")

	api := New(
		// Deliberately omit ServerTokenOpt.
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP request should not be made when server token is empty")
			return nil, fmt.Errorf("should not reach transport")
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{From: "a@b.com", To: "c@d.com", Subject: "no token"})
	if err == nil {
		t.Fatal("expected error for empty server token, got nil")
	}
	if !strings.Contains(err.Error(), "server token") {
		t.Errorf("expected error message to mention server token, got: %v", err)
	}
}

// TestSendEmailBatch_EmptyServerToken verifies the same guard for SendEmailBatch.
func TestSendEmailBatch_EmptyServerToken(t *testing.T) {
	t.Setenv("POSTMARK_SERVER_TOKEN", "")

	api := New(
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP request should not be made when server token is empty")
			return nil, fmt.Errorf("should not reach transport")
		})),
	)

	_, err := api.SendEmailBatch([]SendEmailReq{{From: "a@b.com"}})
	if err == nil {
		t.Fatal("expected error for empty server token, got nil")
	}
	if !strings.Contains(err.Error(), "server token") {
		t.Errorf("expected error message to mention server token, got: %v", err)
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
	// Assert the error type and code.
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Fatalf("expected PostmarkErr, got %T: %v", err, err)
	}
	if pe.ErrorCode != 500 {
		t.Errorf("ErrorCode = %d, want 500", pe.ErrorCode)
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
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestSendEmailBatch_MalformedResponse verifies that a non-JSON response body
// is reported as an unmarshal error, not silently ignored.
func TestSendEmailBatch_MalformedResponse(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("not valid json")),
			}, nil
		})),
	)

	_, err := api.SendEmailBatch([]SendEmailReq{{From: "a@b.com"}})
	if err == nil {
		t.Fatal("expected unmarshal error for malformed response, got nil")
	}
}

// TestSendEmailBatch_Empty verifies that an empty (non-nil) slice is treated
// identically to a nil slice: it returns ([]SendEmailResp{}, nil) immediately,
// without making a network request. Postmark rejects an empty batch array with
// a 422 Unprocessable Entity, so there is nothing to send.
func TestSendEmailBatch_Empty(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP request should not be made for an empty slice")
			return nil, fmt.Errorf("should not reach transport")
		})),
	)

	got, err := api.SendEmailBatch([]SendEmailReq{})
	if err != nil {
		t.Fatalf("unexpected error for empty slice: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("expected empty non-nil slice, got %v", got)
	}
}

// TestSendEmailBatch_Nil verifies that a nil slice is treated as empty —
// it returns ([]SendEmailResp{}, nil) without making a network request.
func TestSendEmailBatch_Nil(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP request should not be made for nil slice")
			return nil, fmt.Errorf("should not reach transport")
		})),
	)

	got, err := api.SendEmailBatch(nil)
	if err != nil {
		t.Fatalf("unexpected error for nil slice: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("expected empty non-nil slice, got %v", got)
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

// TestSendEmailBatch_UsesServerTokenEnvFallback verifies that SendEmailBatch
// also resolves the server token from the POSTMARK_SERVER_TOKEN env var when
// no ServerTokenOpt is provided.
func TestSendEmailBatch_UsesServerTokenEnvFallback(t *testing.T) {
	t.Setenv("POSTMARK_SERVER_TOKEN", "env-batch-token")

	api := New(
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("X-Postmark-Server-Token"); got != "env-batch-token" {
				t.Errorf("expected X-Postmark-Server-Token=env-batch-token, got %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, []SendEmailResp{{Message: "OK"}}),
			}, nil
		})),
	)

	_, err := api.SendEmailBatch([]SendEmailReq{{From: "a@b.com", To: "c@d.com", Subject: "batch env test"}})
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

// TestCreateServer_UsesAccountToken verifies that account-token endpoints
// (e.g. CreateServer) still send X-Postmark-Account-Token and do NOT send
// X-Postmark-Server-Token, ensuring the two auth paths don't regress.
func TestCreateServer_UsesAccountToken(t *testing.T) {
	api := New(
		APITokenOpt("my-account-token"),
		ServerTokenOpt("my-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			acct := req.Header.Get("X-Postmark-Account-Token")
			srv := req.Header.Get("X-Postmark-Server-Token")
			if acct != "my-account-token" {
				t.Errorf("X-Postmark-Account-Token = %q, want my-account-token", acct)
			}
			if srv != "" {
				t.Errorf("X-Postmark-Server-Token should not be set on server endpoint, got %q", srv)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, ServerResp{ID: 1, Name: "Regression"}),
			}, nil
		})),
	)

	_, err := api.CreateServer(&CreateServerReq{Name: "Regression"})
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

// TestSendEmail_SubmittedAtParsed verifies that SubmittedAt is parsed as a
// time.Time from the ISO-8601 string returned by Postmark.
func TestSendEmail_SubmittedAtParsed(t *testing.T) {
	wantTime := time.Date(2024, 3, 10, 14, 30, 0, 0, time.UTC)

	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: jsonBody(t, SendEmailResp{
					To:          "x@y.com",
					SubmittedAt: wantTime,
					MessageID:   "ts-check",
					Message:     "OK",
				}),
			}, nil
		})),
	)

	got, err := api.SendEmail(&SendEmailReq{From: "a@b.com", To: "x@y.com", Subject: "ts"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.SubmittedAt.Equal(wantTime) {
		t.Errorf("SubmittedAt = %v, want %v", got.SubmittedAt, wantTime)
	}
}

// Ensure the time import is used (compile-time check).
var _ = time.Time{}
