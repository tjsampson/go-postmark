package postmark

import (
	"encoding/json"
	"errors"
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
	tests := []struct {
		name    string
		req     *SendEmailReq
		resp    SendEmailResp
		wantErr bool
	}{
		{
			name: "basic send",
			req: &SendEmailReq{
				From:    "sender@example.com",
				To:      "recipient@example.com",
				Subject: "Hello",
				TextBody: "Hello world",
			},
			resp: SendEmailResp{
				To:        "recipient@example.com",
				MessageID: "msg-001",
				ErrorCode: 0,
				Message:   "OK",
			},
		},
		{
			name: "html email with attachments",
			req: &SendEmailReq{
				From:     "sender@example.com",
				To:       "recipient@example.com",
				Subject:  "HTML Email",
				HtmlBody: "<h1>Hello</h1>",
				Attachments: []EmailAttachment{
					{Name: "file.pdf", Content: "base64data", ContentType: "application/pdf"},
				},
				Headers: []EmailHeader{
					{Name: "X-Custom", Value: "value"},
				},
				Metadata:      map[string]string{"key": "val"},
				TrackOpens:    true,
				TrackLinks:    "HtmlAndText",
				MessageStream: "outbound",
			},
			resp: SendEmailResp{
				To:        "recipient@example.com",
				MessageID: "msg-002",
				ErrorCode: 0,
				Message:   "OK",
			},
		},
		{
			name: "email with Cc/Bcc/ReplyTo/Tag",
			req: &SendEmailReq{
				From:     "sender@example.com",
				To:       "a@example.com",
				Cc:       "b@example.com",
				Bcc:      "c@example.com",
				ReplyTo:  "reply@example.com",
				Tag:      "welcome",
				Subject:  "Welcome",
				TextBody: "Hi",
			},
			resp: SendEmailResp{
				To:        "a@example.com",
				MessageID: "msg-003",
				ErrorCode: 0,
				Message:   "OK",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("srv-tok"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodPost {
						t.Errorf("expected POST, got %s", req.Method)
					}
					if !strings.HasSuffix(req.URL.Path, "/email") {
						t.Errorf("unexpected path: %s", req.URL.Path)
					}
					if req.Header.Get("X-Postmark-Server-Token") != "srv-tok" {
						t.Errorf("expected X-Postmark-Server-Token header, got %q", req.Header.Get("X-Postmark-Server-Token"))
					}
					// Ensure account token header is NOT set on email endpoints
					if req.Header.Get("X-Postmark-Account-Token") != "" {
						t.Errorf("X-Postmark-Account-Token should not be set for email endpoints")
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, tc.resp),
					}, nil
				})),
			)

			got, err := api.SendEmail(tc.req)
			if (err != nil) != tc.wantErr {
				t.Fatalf("SendEmail() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil {
				if got.MessageID != tc.resp.MessageID {
					t.Errorf("MessageID = %q, want %q", got.MessageID, tc.resp.MessageID)
				}
				if got.To != tc.resp.To {
					t.Errorf("To = %q, want %q", got.To, tc.resp.To)
				}
			}
		})
	}
}

func TestSendEmail_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 422, Message: "Invalid email address"}

	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{From: "bad", To: "bad"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSendEmail_NotFound(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "not found"}),
			}, nil
		})),
	)

	_, err := api.SendEmail(&SendEmailReq{From: "a@b.com", To: "c@d.com"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

// ---- SendBatch -----------------------------------------------------------------

func TestSendBatch_Success(t *testing.T) {
	tests := []struct {
		name  string
		reqs  []SendEmailReq
		resps []SendEmailResp
	}{
		{
			name: "single message batch",
			reqs: []SendEmailReq{
				{From: "a@a.com", To: "b@b.com", Subject: "Hi", TextBody: "Hello"},
			},
			resps: []SendEmailResp{
				{To: "b@b.com", MessageID: "batch-001", ErrorCode: 0, Message: "OK"},
			},
		},
		{
			name: "multi message batch",
			reqs: []SendEmailReq{
				{From: "a@a.com", To: "b@b.com", Subject: "Hi 1"},
				{From: "a@a.com", To: "c@c.com", Subject: "Hi 2"},
			},
			resps: []SendEmailResp{
				{To: "b@b.com", MessageID: "batch-002", ErrorCode: 0, Message: "OK"},
				{To: "c@c.com", MessageID: "batch-003", ErrorCode: 0, Message: "OK"},
			},
		},
		{
			name:  "empty batch",
			reqs:  []SendEmailReq{},
			resps: []SendEmailResp{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("srv-tok"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodPost {
						t.Errorf("expected POST, got %s", req.Method)
					}
					if !strings.HasSuffix(req.URL.Path, "/email/batch") {
						t.Errorf("unexpected path: %s", req.URL.Path)
					}
					if req.Header.Get("X-Postmark-Server-Token") != "srv-tok" {
						t.Errorf("expected X-Postmark-Server-Token header")
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, tc.resps),
					}, nil
				})),
			)

			got, err := api.SendBatch(tc.reqs)
			if err != nil {
				t.Fatalf("SendBatch() unexpected error: %v", err)
			}
			if len(got) != len(tc.resps) {
				t.Errorf("got %d responses, want %d", len(got), len(tc.resps))
			}
			for i, r := range got {
				if r.MessageID != tc.resps[i].MessageID {
					t.Errorf("[%d] MessageID = %q, want %q", i, r.MessageID, tc.resps[i].MessageID)
				}
			}
		})
	}
}

func TestSendBatch_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendBatch([]SendEmailReq{{From: "a@a.com", To: "b@b.com"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- SendWithTemplate ----------------------------------------------------------

func TestSendWithTemplate_Success(t *testing.T) {
	tests := []struct {
		name string
		req  *SendTemplateReq
		resp SendEmailResp
	}{
		{
			name: "by template ID",
			req: &SendTemplateReq{
				From:       "sender@example.com",
				To:         "recipient@example.com",
				TemplateId: 12345,
				TemplateModel: map[string]interface{}{
					"name":    "Alice",
					"product": "Postmark",
				},
			},
			resp: SendEmailResp{
				To:        "recipient@example.com",
				MessageID: "tmpl-001",
				ErrorCode: 0,
				Message:   "OK",
			},
		},
		{
			name: "by template alias",
			req: &SendTemplateReq{
				From:          "sender@example.com",
				To:            "recipient@example.com",
				TemplateAlias: "welcome-email",
				TemplateModel: map[string]interface{}{
					"name": "Bob",
				},
				TrackOpens:    true,
				TrackLinks:    "HtmlOnly",
				MessageStream: "outbound",
				Metadata:      map[string]string{"campaign": "summer"},
				Headers: []EmailHeader{
					{Name: "X-Campaign", Value: "summer"},
				},
				Attachments: []EmailAttachment{
					{Name: "info.pdf", Content: "data", ContentType: "application/pdf"},
				},
				InlineCss: true,
			},
			resp: SendEmailResp{
				To:        "recipient@example.com",
				MessageID: "tmpl-002",
				ErrorCode: 0,
				Message:   "OK",
			},
		},
		{
			name: "with Cc/Bcc/ReplyTo/Tag",
			req: &SendTemplateReq{
				From:          "sender@example.com",
				To:            "a@example.com",
				Cc:            "b@example.com",
				Bcc:           "c@example.com",
				ReplyTo:       "reply@example.com",
				Tag:           "transactional",
				TemplateAlias: "invoice",
			},
			resp: SendEmailResp{
				To:        "a@example.com",
				MessageID: "tmpl-003",
				ErrorCode: 0,
				Message:   "OK",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("srv-tok"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodPost {
						t.Errorf("expected POST, got %s", req.Method)
					}
					if !strings.HasSuffix(req.URL.Path, "/email/withTemplate") {
						t.Errorf("unexpected path: %s", req.URL.Path)
					}
					if req.Header.Get("X-Postmark-Server-Token") != "srv-tok" {
						t.Errorf("expected X-Postmark-Server-Token header")
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, tc.resp),
					}, nil
				})),
			)

			got, err := api.SendWithTemplate(tc.req)
			if err != nil {
				t.Fatalf("SendWithTemplate() unexpected error: %v", err)
			}
			if got.MessageID != tc.resp.MessageID {
				t.Errorf("MessageID = %q, want %q", got.MessageID, tc.resp.MessageID)
			}
		})
	}
}

func TestSendWithTemplate_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 1101, Message: "template not found"}

	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendWithTemplate(&SendTemplateReq{From: "a@a.com", To: "b@b.com", TemplateId: 9999})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- SendBatchWithTemplates ----------------------------------------------------

func TestSendBatchWithTemplates_Success(t *testing.T) {
	tests := []struct {
		name  string
		reqs  []SendTemplateReq
		resps []SendEmailResp
	}{
		{
			name: "single templated message",
			reqs: []SendTemplateReq{
				{From: "a@a.com", To: "b@b.com", TemplateId: 1},
			},
			resps: []SendEmailResp{
				{To: "b@b.com", MessageID: "btmpl-001", ErrorCode: 0, Message: "OK"},
			},
		},
		{
			name: "multiple templated messages",
			reqs: []SendTemplateReq{
				{From: "a@a.com", To: "b@b.com", TemplateAlias: "welcome"},
				{From: "a@a.com", To: "c@c.com", TemplateAlias: "confirmation"},
			},
			resps: []SendEmailResp{
				{To: "b@b.com", MessageID: "btmpl-002", ErrorCode: 0, Message: "OK"},
				{To: "c@c.com", MessageID: "btmpl-003", ErrorCode: 0, Message: "OK"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("srv-tok"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodPost {
						t.Errorf("expected POST, got %s", req.Method)
					}
					if !strings.HasSuffix(req.URL.Path, "/email/batchWithTemplates") {
						t.Errorf("unexpected path: %s", req.URL.Path)
					}
					if req.Header.Get("X-Postmark-Server-Token") != "srv-tok" {
						t.Errorf("expected X-Postmark-Server-Token header")
					}
					// Verify the request body wraps messages in a "Messages" key
					var body batchTemplateReqWrapper
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Errorf("failed to decode request body: %v", err)
					}
					if len(body.Messages) != len(tc.reqs) {
						t.Errorf("body.Messages length = %d, want %d", len(body.Messages), len(tc.reqs))
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, tc.resps),
					}, nil
				})),
			)

			got, err := api.SendBatchWithTemplates(tc.reqs)
			if err != nil {
				t.Fatalf("SendBatchWithTemplates() unexpected error: %v", err)
			}
			if len(got) != len(tc.resps) {
				t.Errorf("got %d responses, want %d", len(got), len(tc.resps))
			}
			for i, r := range got {
				if r.MessageID != tc.resps[i].MessageID {
					t.Errorf("[%d] MessageID = %q, want %q", i, r.MessageID, tc.resps[i].MessageID)
				}
			}
		})
	}
}

func TestSendBatchWithTemplates_WrapsMessages(t *testing.T) {
	// Verify that the Messages wrapper key is present in the request body.
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			var raw map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&raw); err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}
			if _, ok := raw["Messages"]; !ok {
				t.Error("expected 'Messages' key in request body for batchWithTemplates")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, []SendEmailResp{}),
			}, nil
		})),
	)
	_, err := api.SendBatchWithTemplates([]SendTemplateReq{
		{From: "a@a.com", To: "b@b.com", TemplateId: 1},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendBatchWithTemplates_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendBatchWithTemplates([]SendTemplateReq{{From: "a@a.com", To: "b@b.com", TemplateId: 1}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- CreateBulkJob -------------------------------------------------------------

func TestCreateBulkJob_Success(t *testing.T) {
	tests := []struct {
		name string
		reqs []SendEmailReq
		resp BulkJobResp
	}{
		{
			name: "single message bulk",
			reqs: []SendEmailReq{
				{From: "a@a.com", To: "b@b.com", Subject: "Bulk 1"},
			},
			resp: BulkJobResp{
				ID:         "bulk-job-001",
				Status:     "Queued",
				TotalCount: 1,
			},
		},
		{
			name: "multiple messages bulk",
			reqs: []SendEmailReq{
				{From: "a@a.com", To: "b@b.com", Subject: "Bulk A"},
				{From: "a@a.com", To: "c@c.com", Subject: "Bulk B"},
				{From: "a@a.com", To: "d@d.com", Subject: "Bulk C"},
			},
			resp: BulkJobResp{
				ID:           "bulk-job-002",
				Status:       "Processing",
				TotalCount:   3,
				SuccessCount: 2,
				ErrorCount:   1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("srv-tok"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodPost {
						t.Errorf("expected POST, got %s", req.Method)
					}
					if !strings.HasSuffix(req.URL.Path, "/email/bulk") {
						t.Errorf("unexpected path: %s", req.URL.Path)
					}
					if req.Header.Get("X-Postmark-Server-Token") != "srv-tok" {
						t.Errorf("expected X-Postmark-Server-Token header")
					}
					// Verify the request body wraps messages in a "Messages" key
					var body bulkEmailReqWrapper
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Errorf("failed to decode request body: %v", err)
					}
					if len(body.Messages) != len(tc.reqs) {
						t.Errorf("body.Messages length = %d, want %d", len(body.Messages), len(tc.reqs))
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, tc.resp),
					}, nil
				})),
			)

			got, err := api.CreateBulkJob(tc.reqs)
			if err != nil {
				t.Fatalf("CreateBulkJob() unexpected error: %v", err)
			}
			if got.ID != tc.resp.ID {
				t.Errorf("ID = %q, want %q", got.ID, tc.resp.ID)
			}
			if got.Status != tc.resp.Status {
				t.Errorf("Status = %q, want %q", got.Status, tc.resp.Status)
			}
			if got.TotalCount != tc.resp.TotalCount {
				t.Errorf("TotalCount = %d, want %d", got.TotalCount, tc.resp.TotalCount)
			}
		})
	}
}

func TestCreateBulkJob_WrapsMessages(t *testing.T) {
	// Verify the Messages wrapper key is present in the request body.
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			var raw map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&raw); err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}
			if _, ok := raw["Messages"]; !ok {
				t.Error("expected 'Messages' key in request body for bulk job")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, BulkJobResp{ID: "job-1", Status: "Queued"}),
			}, nil
		})),
	)
	_, err := api.CreateBulkJob([]SendEmailReq{
		{From: "a@a.com", To: "b@b.com", Subject: "Test"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateBulkJob_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "bulk error"}

	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.CreateBulkJob([]SendEmailReq{{From: "a@a.com", To: "b@b.com"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- GetBulkJob ----------------------------------------------------------------

func TestGetBulkJob_Success(t *testing.T) {
	tests := []struct {
		name   string
		jobID  string
		resp   BulkJobResp
	}{
		{
			name:  "queued job",
			jobID: "bulk-job-001",
			resp: BulkJobResp{
				ID:         "bulk-job-001",
				CreatedAt:  "2024-01-01T00:00:00Z",
				Status:     "Queued",
				TotalCount: 5,
			},
		},
		{
			name:  "completed job",
			jobID: "bulk-job-002",
			resp: BulkJobResp{
				ID:           "bulk-job-002",
				CreatedAt:    "2024-01-02T00:00:00Z",
				Status:       "Completed",
				TotalCount:   10,
				SuccessCount: 9,
				ErrorCount:   1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("srv-tok"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodGet {
						t.Errorf("expected GET, got %s", req.Method)
					}
					wantPath := "/email/bulk/" + tc.jobID
					if !strings.HasSuffix(req.URL.Path, wantPath) {
						t.Errorf("unexpected path: %s, want suffix %s", req.URL.Path, wantPath)
					}
					if req.Header.Get("X-Postmark-Server-Token") != "srv-tok" {
						t.Errorf("expected X-Postmark-Server-Token header")
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, tc.resp),
					}, nil
				})),
			)

			got, err := api.GetBulkJob(tc.jobID)
			if err != nil {
				t.Fatalf("GetBulkJob() unexpected error: %v", err)
			}
			if got.ID != tc.resp.ID {
				t.Errorf("ID = %q, want %q", got.ID, tc.resp.ID)
			}
			if got.Status != tc.resp.Status {
				t.Errorf("Status = %q, want %q", got.Status, tc.resp.Status)
			}
			if got.SuccessCount != tc.resp.SuccessCount {
				t.Errorf("SuccessCount = %d, want %d", got.SuccessCount, tc.resp.SuccessCount)
			}
			if got.ErrorCount != tc.resp.ErrorCount {
				t.Errorf("ErrorCount = %d, want %d", got.ErrorCount, tc.resp.ErrorCount)
			}
		})
	}
}

func TestGetBulkJob_NotFound(t *testing.T) {
	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "not found"}),
			}, nil
		})),
	)

	_, err := api.GetBulkJob("nonexistent-id")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

func TestGetBulkJob_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(
		ServerTokenOpt("srv-tok"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.GetBulkJob("job-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- Auth header isolation -----------------------------------------------------

// TestEmailEndpoints_UseServerToken verifies that all email endpoints send
// X-Postmark-Server-Token and do NOT send X-Postmark-Account-Token.
func TestEmailEndpoints_UseServerToken(t *testing.T) {
	const serverTok = "my-server-token"
	const accountTok = "my-account-token"

	checkHeaders := func(t *testing.T, req *http.Request) {
		t.Helper()
		if got := req.Header.Get("X-Postmark-Server-Token"); got != serverTok {
			t.Errorf("X-Postmark-Server-Token = %q, want %q", got, serverTok)
		}
		if got := req.Header.Get("X-Postmark-Account-Token"); got != "" {
			t.Errorf("X-Postmark-Account-Token should be empty for email endpoints, got %q", got)
		}
	}

	newAPI := func(handler roundTripFunc) *API {
		return New(
			ServerTokenOpt(serverTok),
			APITokenOpt(accountTok),
			HTTPClientOpt(newTestClient(handler)),
		)
	}

	t.Run("SendEmail", func(t *testing.T) {
		api := newAPI(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, SendEmailResp{})}, nil
		})
		_, _ = api.SendEmail(&SendEmailReq{From: "a@a.com", To: "b@b.com"})
	})

	t.Run("SendBatch", func(t *testing.T) {
		api := newAPI(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, []SendEmailResp{})}, nil
		})
		_, _ = api.SendBatch([]SendEmailReq{{From: "a@a.com", To: "b@b.com"}})
	})

	t.Run("SendWithTemplate", func(t *testing.T) {
		api := newAPI(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, SendEmailResp{})}, nil
		})
		_, _ = api.SendWithTemplate(&SendTemplateReq{From: "a@a.com", To: "b@b.com", TemplateId: 1})
	})

	t.Run("SendBatchWithTemplates", func(t *testing.T) {
		api := newAPI(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, []SendEmailResp{})}, nil
		})
		_, _ = api.SendBatchWithTemplates([]SendTemplateReq{{From: "a@a.com", To: "b@b.com", TemplateId: 1}})
	})

	t.Run("CreateBulkJob", func(t *testing.T) {
		api := newAPI(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, BulkJobResp{})}, nil
		})
		_, _ = api.CreateBulkJob([]SendEmailReq{{From: "a@a.com", To: "b@b.com"}})
	})

	t.Run("GetBulkJob", func(t *testing.T) {
		api := newAPI(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, BulkJobResp{})}, nil
		})
		_, _ = api.GetBulkJob("job-123")
	})
}
