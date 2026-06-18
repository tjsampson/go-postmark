package postmark

import (
	"encoding/json"
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
		From:    "sender@example.com",
		To:      "receiver@example.com",
		Subject: "Hello",
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
		From:    "from@example.com",
		To:      "to@example.com",
		Subject: "Metadata test",
		TextBody: "Hello",
		Metadata: map[string]string{"customer-id": "abc123"},
		Headers:  []MailHeader{{Name: "X-Custom", Value: "my-value"}},
		Attachments: []Attachment{
			{Name: "file.txt", Content: "aGVsbG8=", ContentType: "text/plain"},
		},
		TrackOpens: true,
		TrackLinks: "None",
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
				TemplateModel: map[string]interface{}{"name": "Alice"},
			},
			{
				From:          "sender@example.com",
				To:            "b@example.com",
				TemplateAlias: "welcome-email",
				TemplateModel: map[string]interface{}{"name": "Bob"},
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
				TemplateModel: map[string]interface{}{"key": "value"},
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
