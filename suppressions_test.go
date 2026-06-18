package postmark

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ---- ListSuppressions ----------------------------------------------------------

func TestListSuppressions_Success(t *testing.T) {
	wantCreatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	want := ListSuppressionsResp{
		Suppressions: []SuppressionResp{
			{
				EmailAddress:      "test@example.com",
				SuppressionReason: "HardBounce",
				Origin:            "Recipient",
				CreatedAt:         wantCreatedAt,
			},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/message-streams/outbound/suppressions/dump") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListSuppressions("outbound", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Suppressions) != 1 {
		t.Fatalf("expected 1 suppression, got %d", len(got.Suppressions))
	}
	s := got.Suppressions[0]
	if s.EmailAddress != "test@example.com" {
		t.Errorf("EmailAddress = %q, want test@example.com", s.EmailAddress)
	}
	if s.SuppressionReason != "HardBounce" {
		t.Errorf("SuppressionReason = %q, want HardBounce", s.SuppressionReason)
	}
	if !s.CreatedAt.Equal(wantCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", s.CreatedAt, wantCreatedAt)
	}
}

func TestListSuppressions_WithParams(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.Query()
		if q.Get("SuppressionReason") != "HardBounce" {
			t.Errorf("expected SuppressionReason=HardBounce in query, got: %s", req.URL.RawQuery)
		}
		if q.Get("Origin") != "Recipient" {
			t.Errorf("expected Origin=Recipient in query, got: %s", req.URL.RawQuery)
		}
		// Verify that special characters in EmailAddress are properly URL-encoded.
		if q.Get("EmailAddress") != "user+tag@example.com" {
			t.Errorf("expected EmailAddress=user+tag@example.com (decoded), got: %q", q.Get("EmailAddress"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, ListSuppressionsResp{}),
		}, nil
	})))

	params := &ListSuppressionsParams{
		SuppressionReason: "HardBounce",
		Origin:            "Recipient",
		EmailAddress:      "user+tag@example.com",
	}
	_, err := api.ListSuppressions("outbound", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListSuppressions_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.ListSuppressions("outbound", nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- CreateSuppressions --------------------------------------------------------

func TestCreateSuppressions_Success(t *testing.T) {
	want := CreateSuppressionsResp{
		Suppressions: []SuppressionResult{
			{EmailAddress: "suppress@example.com", Status: "Suppressed"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound/suppressions") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		// Assert that the correct email address was serialised in the request body.
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var sentReq CreateSuppressionsReq
		if err := json.Unmarshal(bodyBytes, &sentReq); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if len(sentReq.Suppressions) != 1 || sentReq.Suppressions[0].EmailAddress != "suppress@example.com" {
			t.Errorf("unexpected request body suppressions: %+v", sentReq.Suppressions)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	body := &CreateSuppressionsReq{
		Suppressions: []SuppressionEmail{
			{EmailAddress: "suppress@example.com"},
		},
	}
	got, err := api.CreateSuppressions("outbound", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Suppressions) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got.Suppressions))
	}
	if got.Suppressions[0].Status != "Suppressed" {
		t.Errorf("Status = %q, want Suppressed", got.Suppressions[0].Status)
	}
}

func TestCreateSuppressions_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 400, Message: "bad request"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateSuppressions("outbound", &CreateSuppressionsReq{})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- DeleteSuppressions --------------------------------------------------------

func TestDeleteSuppressions_Success(t *testing.T) {
	want := DeleteSuppressionsResp{
		Suppressions: []SuppressionResult{
			{EmailAddress: "suppress@example.com", Status: "Deleted"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound/suppressions/delete") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		// Assert that the correct email address was serialised in the request body.
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var sentReq DeleteSuppressionsReq
		if err := json.Unmarshal(bodyBytes, &sentReq); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if len(sentReq.Suppressions) != 1 || sentReq.Suppressions[0].EmailAddress != "suppress@example.com" {
			t.Errorf("unexpected request body suppressions: %+v", sentReq.Suppressions)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	body := &DeleteSuppressionsReq{
		Suppressions: []SuppressionEmail{
			{EmailAddress: "suppress@example.com"},
		},
	}
	got, err := api.DeleteSuppressions("outbound", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Suppressions) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got.Suppressions))
	}
	if got.Suppressions[0].Status != "Deleted" {
		t.Errorf("Status = %q, want Deleted", got.Suppressions[0].Status)
	}
}

func TestDeleteSuppressions_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "stream not found"}),
		}, nil
	})))

	_, err := api.DeleteSuppressions("nonexistent", &DeleteSuppressionsReq{})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestDeleteSuppressions_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.DeleteSuppressions("outbound", &DeleteSuppressionsReq{})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
