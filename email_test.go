package postmark

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// boolPtr is a test helper that returns a pointer to the given bool value,
// which is needed when setting *bool fields such as TrackOpens or InlineCss.
func boolPtr(b bool) *bool { return &b }

// ---- ServerTokenOpt ------------------------------------------------------------

func TestNew_WithServerTokenOpt(t *testing.T) {
	api := New(ServerTokenOpt("srv-tok-abc"))
	if api.serverToken != "srv-tok-abc" {
		t.Errorf("expected serverToken srv-tok-abc, got %s", api.serverToken)
	}
}

// ---- newServerRequest ----------------------------------------------------------

func TestNewServerRequest_SetsServerTokenHeader(t *testing.T) {
	api := New(ServerTokenOpt("my-server-token"))
	req, err := api.newServerRequest(http.MethodPost, "email", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("X-Postmark-Server-Token"); got != "my-server-token" {
		t.Errorf("X-Postmark-Server-Token = %q, want %q", got, "my-server-token")
	}
	// Must NOT set the account-token header.
	if got := req.Header.Get("X-Postmark-Account-Token"); got != "" {
		t.Errorf("X-Postmark-Account-Token should be empty, got %q", got)
	}
}

// TestNewServerRequest_EmptyTokenReturnsError verifies that newServerRequest
// returns an error immediately when no server token has been configured, rather
// than silently sending an unauthenticated request to Postmark.
func TestNewServerRequest_EmptyTokenReturnsError(t *testing.T) {
	api := New() // no ServerTokenOpt
	_, err := api.newServerRequest(http.MethodPost, "email", nil)
	if err == nil {
		t.Fatal("expected an error when serverToken is empty, got nil")
	}
	if !strings.Contains(err.Error(), "serverToken is not set") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestNewRequest_SetsAccountTokenHeader is the inverse of the above: newRequest
// must set X-Postmark-Account-Token and must NOT set X-Postmark-Server-Token.
// This guards against regressions from the buildRequest refactor.
func TestNewRequest_SetsAccountTokenHeader(t *testing.T) {
	api := New(APITokenOpt("my-account-token"))
	req, err := api.newRequest(http.MethodGet, "servers/1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("X-Postmark-Account-Token"); got != "my-account-token" {
		t.Errorf("X-Postmark-Account-Token = %q, want %q", got, "my-account-token")
	}
	// Must NOT set the server-token header.
	if got := req.Header.Get("X-Postmark-Server-Token"); got != "" {
		t.Errorf("X-Postmark-Server-Token should be empty, got %q", got)
	}
}

// ---- SendEmail -----------------------------------------------------------------

func TestSendEmail_Success(t *testing.T) {
	want := SendEmailResp{
		To:          "receiver@example.com",
		SubmittedAt: "2023-01-01T00:00:00Z",
		MessageID:   "msg-id-1",
		ErrorCode:   0,
		Message:     "OK",
	}

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/email") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") != "srv-token" {
				t.Errorf("missing or wrong X-Postmark-Server-Token header")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.SendEmail(&SendEmailReq{
		From:     "sender@example.com",
		To:       "receiver@example.com",
		Subject:  "Hello",
		HtmlBody: "<p>Hi</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.To != want.To {
		t.Errorf("To = %q, want %q", got.To, want.To)
	}
	if got.MessageID != want.MessageID {
		t.Errorf("MessageID = %q, want %q", got.MessageID, want.MessageID)
	}
}

func TestSendEmail_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 300, Message: "Invalid email request"}

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{From: "bad", To: "also-bad"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestSendEmail_EmptyServerToken verifies that SendEmail returns an error (not
// a silent 401 from Postmark) when no server token has been configured.
func TestSendEmail_EmptyServerToken(t *testing.T) {
	api := New() // no ServerTokenOpt
	_, err := api.SendEmail(&SendEmailReq{From: "f@example.com", To: "t@example.com"})
	if err == nil {
		t.Fatal("expected an error when serverToken is empty, got nil")
	}
	if !strings.Contains(err.Error(), "serverToken is not set") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSendEmail_WithMetadataAndHeaders(t *testing.T) {
	want := SendEmailResp{
		To:        "to@example.com",
		MessageID: "msg-meta-1",
		ErrorCode: 0,
		Message:   "OK",
	}

	var capturedBody SendEmailReq

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&capturedBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	emailReq := &SendEmailReq{
		From:     "from@example.com",
		To:       "to@example.com",
		Subject:  "Metadata test",
		TextBody: "Hello",
		Metadata: map[string]string{"customer-id": "abc123"},
		Headers:  []MailHeader{{Name: "X-Custom", Value: "my-value"}},
		Attachments: []Attachment{
			{Name: "file.txt", Content: "aGVsbG8=", ContentType: "text/plain"},
		},
		TrackOpens: boolPtr(true),
		TrackLinks: TrackLinksNone,
	}

	_, err := api.SendEmail(emailReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedBody.Metadata["customer-id"] != "abc123" {
		t.Errorf("Metadata customer-id = %q, want abc123", capturedBody.Metadata["customer-id"])
	}
	if len(capturedBody.Headers) != 1 || capturedBody.Headers[0].Name != "X-Custom" {
		t.Errorf("unexpected headers: %+v", capturedBody.Headers)
	}
	if len(capturedBody.Attachments) != 1 || capturedBody.Attachments[0].Name != "file.txt" {
		t.Errorf("unexpected attachments: %+v", capturedBody.Attachments)
	}
}

// TestSendEmail_TrackOpensFalseIsSerialised verifies that explicitly setting
// TrackOpens to false results in the field being present in the JSON payload
// (not silently dropped by omitempty), which was the bug with the old bool type.
func TestSendEmail_TrackOpensFalseIsSerialised(t *testing.T) {
	var capturedBody map[string]any

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&capturedBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{}),
			}, nil
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{
		From:       "f@example.com",
		To:         "t@example.com",
		TrackOpens: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v, ok := capturedBody["TrackOpens"]
	if !ok {
		t.Fatal("TrackOpens key is missing from JSON payload; false *bool must not be omitted")
	}
	if v != false {
		t.Errorf("TrackOpens = %v, want false", v)
	}
}

// TestSendEmail_TrackLinksValue verifies that the TrackLinksValue typed constant
// is correctly serialised in the JSON payload.
func TestSendEmail_TrackLinksValue(t *testing.T) {
	tests := []struct {
		name       string
		trackLinks TrackLinksValue
		wantJSON   string
	}{
		{"None", TrackLinksNone, "None"},
		{"HtmlAndText", TrackLinksHtmlAndText, "HtmlAndText"},
		{"HtmlOnly", TrackLinksHtmlOnly, "HtmlOnly"},
		{"TextOnly", TrackLinksTextOnly, "TextOnly"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var capturedBody map[string]any

			api := New(
				ServerTokenOpt("srv-token"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					if err := json.NewDecoder(req.Body).Decode(&capturedBody); err != nil {
						t.Fatalf("failed to decode request body: %v", err)
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, SendEmailResp{}),
					}, nil
				})),
			)

			_, err := api.SendEmail(&SendEmailReq{
				From:       "f@example.com",
				To:         "t@example.com",
				TrackLinks: tc.trackLinks,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			v, ok := capturedBody["TrackLinks"]
			if !ok {
				t.Fatalf("TrackLinks key is missing from JSON payload")
			}
			if v != tc.wantJSON {
				t.Errorf("TrackLinks = %q, want %q", v, tc.wantJSON)
			}
		})
	}
}

// TestSendEmail_TrackLinksOmitted verifies that leaving TrackLinks as the zero
// value ("") causes the field to be omitted from the JSON payload entirely,
// which defers to the message-stream default — distinct from setting
// TrackLinksNone, which explicitly disables link tracking.
func TestSendEmail_TrackLinksOmitted(t *testing.T) {
	var capturedBody map[string]any

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&capturedBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{}),
			}, nil
		})),
	)

	// TrackLinks is intentionally left at its zero value.
	_, err := api.SendEmail(&SendEmailReq{
		From: "f@example.com",
		To:   "t@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, present := capturedBody["TrackLinks"]; present {
		t.Error("TrackLinks should be absent from JSON when zero value; omitting defers to stream default")
	}
}

// ---- SendEmailBatch ------------------------------------------------------------

func TestSendEmailBatch_Success(t *testing.T) {
	wantResp := []*SendEmailResp{
		{To: "a@example.com", MessageID: "id-1", ErrorCode: 0, Message: "OK"},
		{To: "b@example.com", MessageID: "id-2", ErrorCode: 0, Message: "OK"},
	}

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/email/batch") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") != "srv-token" {
				t.Errorf("missing X-Postmark-Server-Token header")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, wantResp),
			}, nil
		})),
	)

	reqs := []*SendEmailReq{
		{From: "sender@example.com", To: "a@example.com", Subject: "Hi A"},
		{From: "sender@example.com", To: "b@example.com", Subject: "Hi B"},
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

func TestSendEmailBatch_EmptySlice(t *testing.T) {
	api := New(ServerTokenOpt("srv-token"))
	_, err := api.SendEmailBatch([]*SendEmailReq{})
	if err == nil {
		t.Fatal("expected an error for empty batch, got nil")
	}
	if !strings.Contains(err.Error(), "at least one message") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSendEmailBatch_ExceedsMaxSize(t *testing.T) {
	reqs := make([]*SendEmailReq, 501)
	for i := range reqs {
		reqs[i] = &SendEmailReq{From: "f@example.com", To: "t@example.com"}
	}

	api := New(ServerTokenOpt("srv-token"))
	_, err := api.SendEmailBatch(reqs)
	if err == nil {
		t.Fatal("expected an error for batch size > 500, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds the maximum") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSendEmailBatch_ExactlyMaxSize(t *testing.T) {
	// 500 messages is the limit — should not return an error before the HTTP call.
	reqs := make([]*SendEmailReq, 500)
	for i := range reqs {
		reqs[i] = &SendEmailReq{From: "f@example.com", To: "t@example.com"}
	}

	wantResp := make([]*SendEmailResp, 500)
	for i := range wantResp {
		wantResp[i] = &SendEmailResp{To: "t@example.com", MessageID: "id", ErrorCode: 0, Message: "OK"}
	}

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, wantResp),
			}, nil
		})),
	)

	got, err := api.SendEmailBatch(reqs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 500 {
		t.Errorf("expected 500 responses, got %d", len(got))
	}
}

func TestSendEmailBatch_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 400, Message: "Bad request"}

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendEmailBatch([]*SendEmailReq{
		{From: "f@example.com", To: "t@example.com"},
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestSendEmailBatch_EmptyServerToken verifies that SendEmailBatch returns an
// error immediately (not a silent 401 from Postmark) when no server token has
// been configured. The empty-token guard fires after the batch pre-flight
// checks, so the batch must be non-empty for this test to reach the token check.
func TestSendEmailBatch_EmptyServerToken(t *testing.T) {
	api := New() // no ServerTokenOpt
	_, err := api.SendEmailBatch([]*SendEmailReq{
		{From: "f@example.com", To: "t@example.com"},
	})
	if err == nil {
		t.Fatal("expected an error when serverToken is empty, got nil")
	}
	if !strings.Contains(err.Error(), "serverToken is not set") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---- SendEmailBatchWithTemplates -----------------------------------------------

func TestSendEmailBatchWithTemplates_Success(t *testing.T) {
	wantResp := SendBatchWithTemplatesResp{
		TotalCount: 2,
		Messages: []*SendEmailResp{
			{To: "a@example.com", MessageID: "tmpl-id-1", ErrorCode: 0, Message: "OK"},
			{To: "b@example.com", MessageID: "tmpl-id-2", ErrorCode: 0, Message: "OK"},
		},
	}

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/email/batchWithTemplates") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") != "srv-token" {
				t.Errorf("missing X-Postmark-Server-Token header")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, wantResp),
			}, nil
		})),
	)

	batchReq := &SendBatchWithTemplatesReq{
		Messages: []*TemplateMessage{
			{
				From:          "sender@example.com",
				To:            "a@example.com",
				TemplateID:    1001,
				TemplateModel: map[string]any{"name": "Alice"},
			},
			{
				From:          "sender@example.com",
				To:            "b@example.com",
				TemplateAlias: "welcome-email",
				TemplateModel: map[string]any{"name": "Bob"},
			},
		},
	}

	got, err := api.SendEmailBatchWithTemplates(batchReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got.Messages))
	}
	if got.Messages[0].MessageID != "tmpl-id-1" {
		t.Errorf("Messages[0].MessageID = %q, want tmpl-id-1", got.Messages[0].MessageID)
	}
}

func TestSendEmailBatchWithTemplates_RequestBody(t *testing.T) {
	// Verify the request body is serialised correctly.
	var capturedBody SendBatchWithTemplatesReq

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&capturedBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendBatchWithTemplatesResp{TotalCount: 1}),
			}, nil
		})),
	)

	batchReq := &SendBatchWithTemplatesReq{
		Messages: []*TemplateMessage{
			{
				From:          "from@example.com",
				To:            "to@example.com",
				TemplateID:    42,
				TemplateModel: map[string]any{"key": "value"},
				MessageStream: "outbound",
			},
		},
	}

	_, err := api.SendEmailBatchWithTemplates(batchReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedBody.Messages) != 1 {
		t.Fatalf("expected 1 message in captured body, got %d", len(capturedBody.Messages))
	}
	msg := capturedBody.Messages[0]
	if msg.TemplateID != 42 {
		t.Errorf("TemplateID = %d, want 42", msg.TemplateID)
	}
	if msg.MessageStream != "outbound" {
		t.Errorf("MessageStream = %q, want outbound", msg.MessageStream)
	}
}

func TestSendEmailBatchWithTemplates_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(
		ServerTokenOpt("srv-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendEmailBatchWithTemplates(&SendBatchWithTemplatesReq{
		Messages: []*TemplateMessage{
			{From: "f@example.com", To: "t@example.com", TemplateID: 1},
		},
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSendEmailBatchWithTemplates_EmptyBatch(t *testing.T) {
	api := New(ServerTokenOpt("srv-token"))
	_, err := api.SendEmailBatchWithTemplates(&SendBatchWithTemplatesReq{
		Messages: []*TemplateMessage{},
	})
	if err == nil {
		t.Fatal("expected an error for empty batch, got nil")
	}
	if !strings.Contains(err.Error(), "at least one message") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSendEmailBatchWithTemplates_ExceedsMaxSize(t *testing.T) {
	msgs := make([]*TemplateMessage, 501)
	for i := range msgs {
		msgs[i] = &TemplateMessage{
			From:       "f@example.com",
			To:         "t@example.com",
			TemplateID: 1,
		}
	}

	api := New(ServerTokenOpt("srv-token"))
	_, err := api.SendEmailBatchWithTemplates(&SendBatchWithTemplatesReq{Messages: msgs})
	if err == nil {
		t.Fatal("expected an error for batch size > 500, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds the maximum") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSendEmailBatchWithTemplates_MissingTemplate(t *testing.T) {
	// A message with neither TemplateID nor TemplateAlias must be rejected
	// before making any HTTP request.
	api := New(ServerTokenOpt("srv-token"))
	_, err := api.SendEmailBatchWithTemplates(&SendBatchWithTemplatesReq{
		Messages: []*TemplateMessage{
			{From: "f@example.com", To: "t@example.com"}, // no TemplateID or TemplateAlias
		},
	})
	if err == nil {
		t.Fatal("expected an error for missing template, got nil")
	}
	if !strings.Contains(err.Error(), "TemplateID") || !strings.Contains(err.Error(), "TemplateAlias") {
		t.Errorf("error message should mention TemplateID and TemplateAlias, got: %v", err)
	}
}

// TestSendEmailBatchWithTemplates_NilRequest verifies that a nil batchReq is
// handled gracefully and returns an error rather than panicking.
func TestSendEmailBatchWithTemplates_NilRequest(t *testing.T) {
	api := New(ServerTokenOpt("srv-token"))
	_, err := api.SendEmailBatchWithTemplates(nil)
	if err == nil {
		t.Fatal("expected an error for nil batchReq, got nil")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error message should mention nil, got: %v", err)
	}
}

// TestSendEmailBatchWithTemplates_EmptyServerToken verifies that
// SendEmailBatchWithTemplates returns an error immediately (not a silent 401
// from Postmark) when no server token has been configured. The empty-token guard
// fires after all pre-flight validation, so the batch must be valid to reach it.
func TestSendEmailBatchWithTemplates_EmptyServerToken(t *testing.T) {
	api := New() // no ServerTokenOpt
	_, err := api.SendEmailBatchWithTemplates(&SendBatchWithTemplatesReq{
		Messages: []*TemplateMessage{
			{From: "f@example.com", To: "t@example.com", TemplateID: 1},
		},
	})
	if err == nil {
		t.Fatal("expected an error when serverToken is empty, got nil")
	}
	if !strings.Contains(err.Error(), "serverToken is not set") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestSendBatchWithTemplatesResp_Deserialisation verifies that the response
// envelope for POST /email/batchWithTemplates is correctly deserialised.
// The Postmark API returns {"TotalCount":N,"Messages":[...]} — there is no
// top-level Errors array; per-message errors are reported inside each Messages
// entry via ErrorCode and Message fields.
func TestSendBatchWithTemplatesResp_Deserialisation(t *testing.T) {
	raw := `{
		"TotalCount": 2,
		"Messages": [
			{"To": "a@example.com", "MessageID": "id-1", "ErrorCode": 0, "Message": "OK"},
			{"To": "b@example.com", "MessageID": "",     "ErrorCode": 406, "Message": "You tried to send to a recipient that has been marked as inactive."}
		]
	}`

	var resp SendBatchWithTemplatesResp
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if resp.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", resp.TotalCount)
	}
	if len(resp.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(resp.Messages))
	}
	if resp.Messages[0].ErrorCode != 0 {
		t.Errorf("Messages[0].ErrorCode = %d, want 0", resp.Messages[0].ErrorCode)
	}
	if resp.Messages[1].ErrorCode != 406 {
		t.Errorf("Messages[1].ErrorCode = %d, want 406", resp.Messages[1].ErrorCode)
	}
	if resp.Messages[1].Message != "You tried to send to a recipient that has been marked as inactive." {
		t.Errorf("Messages[1].Message = %q", resp.Messages[1].Message)
	}
}
