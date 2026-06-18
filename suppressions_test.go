package postmark

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// ---- ListSuppressions ----------------------------------------------------------

func TestListSuppressions_Success(t *testing.T) {
	want := ListSuppressionsResp{
		Suppressions: []SuppressionResp{
			{
				EmailAddress:      "test@example.com",
				SuppressionReason: "HardBounce",
				Origin:            "Recipient",
				CreatedAt:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
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
	if got.Suppressions[0].EmailAddress != "test@example.com" {
		t.Errorf("EmailAddress = %q, want test@example.com", got.Suppressions[0].EmailAddress)
	}
	if got.Suppressions[0].SuppressionReason != "HardBounce" {
		t.Errorf("SuppressionReason = %q, want HardBounce", got.Suppressions[0].SuppressionReason)
	}
}

func TestListSuppressions_WithParams(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.RawQuery
		if !strings.Contains(q, "SuppressionReason=HardBounce") {
			t.Errorf("expected SuppressionReason param in query: %s", q)
		}
		if !strings.Contains(q, "Origin=Recipient") {
			t.Errorf("expected Origin param in query: %s", q)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, ListSuppressionsResp{}),
		}, nil
	})))

	params := &ListSuppressionsParams{
		SuppressionReason: "HardBounce",
		Origin:            "Recipient",
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
		Suppressions: []SuppressionCreateResult{
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
		Suppressions: []SuppressionDeleteResult{
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
