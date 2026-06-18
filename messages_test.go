package postmark

import (
	"net/http"
	"strings"
	"testing"
)

// ---- SearchOutboundMessages ---------------------------------------------------

func TestSearchOutboundMessages_Success(t *testing.T) {
	want := ListOutboundMessagesResp{
		TotalCount: 2,
		Messages: []OutboundMessageSummary{
			{MessageID: "msg-1", Subject: "Hello"},
			{MessageID: "msg-2", Subject: "World"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "messages/outbound") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if !strings.Contains(req.URL.RawQuery, "count=10") {
			t.Errorf("expected count param, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.SearchOutboundMessages(OutboundMessageSearchParams{Count: 10, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
	if len(got.Messages) != 2 {
		t.Errorf("len(Messages) = %d, want 2", len(got.Messages))
	}
}

func TestSearchOutboundMessages_WithFilters(t *testing.T) {
	tests := []struct {
		name   string
		params OutboundMessageSearchParams
		check  func(t *testing.T, query string)
	}{
		{
			name: "with recipient filter",
			params: OutboundMessageSearchParams{Count: 5, Offset: 0, Recipient: "user@example.com"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "recipient=user%40example.com") {
					t.Errorf("expected recipient param, query=%s", query)
				}
			},
		},
		{
			name: "with status filter",
			params: OutboundMessageSearchParams{Count: 5, Offset: 0, Status: "sent"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "status=sent") {
					t.Errorf("expected status param, query=%s", query)
				}
			},
		},
		{
			name: "with date range",
			params: OutboundMessageSearchParams{Count: 5, Offset: 0, FromDate: "2024-01-01", ToDate: "2024-01-31"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "fromdate=2024-01-01") {
					t.Errorf("expected fromdate param, query=%s", query)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				tc.check(t, req.URL.RawQuery)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, ListOutboundMessagesResp{}),
				}, nil
			})))
			_, err := api.SearchOutboundMessages(tc.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSearchOutboundMessages_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.SearchOutboundMessages(OutboundMessageSearchParams{Count: 10})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- GetOutboundMessageDetails -----------------------------------------------

func TestGetOutboundMessageDetails_Success(t *testing.T) {
	want := OutboundMessageDetailsResp{
		MessageID: "abc-123",
		Subject:   "Test Subject",
		Status:    "sent",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "abc-123/details") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundMessageDetails("abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MessageID != "abc-123" {
		t.Errorf("MessageID = %q, want abc-123", got.MessageID)
	}
	if got.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want Test Subject", got.Subject)
	}
}

func TestGetOutboundMessageDetails_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "message not found"}),
		}, nil
	})))

	_, err := api.GetOutboundMessageDetails("unknown")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- GetOutboundMessageDump --------------------------------------------------

func TestGetOutboundMessageDump_Success(t *testing.T) {
	want := MessageDumpResp{Body: "MIME-Version: 1.0\r\nFrom: sender@example.com\r\n..."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "abc-123/dump") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundMessageDump("abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Body == "" {
		t.Error("expected non-empty Body")
	}
}

// ---- GetOutboundMessageOpens -------------------------------------------------

func TestGetOutboundMessageOpens_Success(t *testing.T) {
	want := ListOutboundOpensResp{
		TotalCount: 1,
		Opens: []OpenEvent{
			{MessageID: "msg-1", Recipient: "user@example.com"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "messages/outbound/opens") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if !strings.Contains(req.URL.RawQuery, "count=5") {
			t.Errorf("expected count param, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundMessageOpens(OutboundOpensParams{Count: 5, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
}

func TestGetOutboundMessageOpens_WithFilters(t *testing.T) {
	tests := []struct {
		name   string
		params OutboundOpensParams
		check  func(t *testing.T, query string)
	}{
		{
			name:   "with platform filter",
			params: OutboundOpensParams{Count: 5, Platform: "Desktop"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "platform=Desktop") {
					t.Errorf("expected platform param, query=%s", query)
				}
			},
		},
		{
			name:   "with country filter",
			params: OutboundOpensParams{Count: 5, Country: "US"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "country=US") {
					t.Errorf("expected country param, query=%s", query)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				tc.check(t, req.URL.RawQuery)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, ListOutboundOpensResp{}),
				}, nil
			})))
			_, err := api.GetOutboundMessageOpens(tc.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---- GetOutboundMessageOpensByMessageID -------------------------------------

func TestGetOutboundMessageOpensByMessageID_Success(t *testing.T) {
	want := ListOutboundOpensResp{
		TotalCount: 1,
		Opens: []OpenEvent{
			{MessageID: "msg-42", Recipient: "reader@example.com"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "opens/msg-42") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if !strings.Contains(req.URL.RawQuery, "count=10") {
			t.Errorf("expected count param, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundMessageOpensByMessageID("msg-42", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
}

// ---- GetOutboundMessageClicks -----------------------------------------------

func TestGetOutboundMessageClicks_Success(t *testing.T) {
	want := ListOutboundClicksResp{
		TotalCount: 3,
		Clicks: []ClickEvent{
			{MessageID: "msg-1", Recipient: "clicker@example.com"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "messages/outbound/clicks") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundMessageClicks(OutboundClicksParams{Count: 10, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", got.TotalCount)
	}
}

func TestGetOutboundMessageClicks_WithFilters(t *testing.T) {
	tests := []struct {
		name   string
		params OutboundClicksParams
		check  func(t *testing.T, query string)
	}{
		{
			name:   "with tag filter",
			params: OutboundClicksParams{Count: 5, Tag: "newsletter"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "tag=newsletter") {
					t.Errorf("expected tag param, query=%s", query)
				}
			},
		},
		{
			name:   "with city filter",
			params: OutboundClicksParams{Count: 5, City: "New York"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "city=New+York") && !strings.Contains(query, "city=New%20York") {
					t.Errorf("expected city param, query=%s", query)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				tc.check(t, req.URL.RawQuery)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, ListOutboundClicksResp{}),
				}, nil
			})))
			_, err := api.GetOutboundMessageClicks(tc.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---- GetOutboundMessageClicksByMessageID ------------------------------------

func TestGetOutboundMessageClicksByMessageID_Success(t *testing.T) {
	want := ListOutboundClicksResp{
		TotalCount: 2,
		Clicks: []ClickEvent{
			{MessageID: "msg-99", OriginalLink: "https://example.com"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "clicks/msg-99") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if !strings.Contains(req.URL.RawQuery, "count=20") {
			t.Errorf("expected count param, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundMessageClicksByMessageID("msg-99", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
}

// ---- SearchInboundMessages --------------------------------------------------

func TestSearchInboundMessages_Success(t *testing.T) {
	want := ListInboundMessagesResp{
		TotalCount: 1,
		InboundMessages: []InboundMessageSummary{
			{MessageID: "inbound-1", Subject: "Re: Hello"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "messages/inbound") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if !strings.Contains(req.URL.RawQuery, "count=10") {
			t.Errorf("expected count param, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.SearchInboundMessages(InboundMessageSearchParams{Count: 10, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
}

func TestSearchInboundMessages_WithFilters(t *testing.T) {
	tests := []struct {
		name   string
		params InboundMessageSearchParams
		check  func(t *testing.T, query string)
	}{
		{
			name:   "with mailboxhash filter",
			params: InboundMessageSearchParams{Count: 5, MailboxHash: "hash123"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "mailboxhash=hash123") {
					t.Errorf("expected mailboxhash param, query=%s", query)
				}
			},
		},
		{
			name:   "with status filter",
			params: InboundMessageSearchParams{Count: 5, Status: "processed"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "status=processed") {
					t.Errorf("expected status param, query=%s", query)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				tc.check(t, req.URL.RawQuery)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, ListInboundMessagesResp{}),
				}, nil
			})))
			_, err := api.SearchInboundMessages(tc.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---- GetInboundMessageDetails -----------------------------------------------

func TestGetInboundMessageDetails_Success(t *testing.T) {
	want := InboundMessageDetailsResp{
		MessageID: "inbound-42",
		Subject:   "Inbound Test",
		Status:    "processed",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "inbound-42/details") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetInboundMessageDetails("inbound-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MessageID != "inbound-42" {
		t.Errorf("MessageID = %q, want inbound-42", got.MessageID)
	}
}

func TestGetInboundMessageDetails_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "not found"}),
		}, nil
	})))

	_, err := api.GetInboundMessageDetails("missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- BypassInboundMessageRules ----------------------------------------------

func TestBypassInboundMessageRules_Success(t *testing.T) {
	want := InboundBypassResp{ErrorCode: 0, Message: "OK"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "msg-bypass/bypass") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.BypassInboundMessageRules("msg-bypass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "OK" {
		t.Errorf("Message = %q, want OK", got.Message)
	}
}

func TestBypassInboundMessageRules_Error(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.BypassInboundMessageRules("bad-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- RetryInboundMessage ----------------------------------------------------

func TestRetryInboundMessage_Success(t *testing.T) {
	want := InboundRetryResp{ErrorCode: 0, Message: "OK"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "msg-retry/retry") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.RetryInboundMessage("msg-retry")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "OK" {
		t.Errorf("Message = %q, want OK", got.Message)
	}
}

func TestRetryInboundMessage_Error(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "message not found"}),
		}, nil
	})))

	_, err := api.RetryInboundMessage("missing-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
