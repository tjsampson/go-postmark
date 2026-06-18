package postmark

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
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

// TestListSuppressions_WithParams verifies that query parameters are forwarded
// correctly, including proper URL-encoding of special characters (e.g. '+' in
// an email address that would otherwise be interpreted as a space).
func TestListSuppressions_WithParams(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.Query()
		if q.Get("SuppressionReason") != "HardBounce" {
			t.Errorf("expected SuppressionReason=HardBounce in query, got: %s", req.URL.RawQuery)
		}
		if q.Get("Origin") != "Recipient" {
			t.Errorf("expected Origin=Recipient in query, got: %s", req.URL.RawQuery)
		}
		// Verify that special characters in EmailAddress survive the round-trip.
		// Use Query().Get() which percent-decodes the value, so the comparison is
		// independent of how the Go runtime encodes '+' or '@'.
		if q.Get("EmailAddress") != "user+tag@example.com" {
			t.Errorf("expected EmailAddress=user+tag@example.com (decoded), got: %q", q.Get("EmailAddress"))
		}
		// Confirm that '+' is sent as %2B (not a literal '+', which would be
		// misinterpreted as a space on decode). Decode the raw query ourselves
		// so the assertion is not sensitive to whether '@' is encoded or not.
		rawEmail := ""
		for _, part := range strings.Split(req.URL.RawQuery, "&") {
			if strings.HasPrefix(part, "EmailAddress=") {
				rawEmail = strings.TrimPrefix(part, "EmailAddress=")
				break
			}
		}
		decoded, err := url.QueryUnescape(rawEmail)
		if err != nil {
			t.Fatalf("could not unescape raw EmailAddress %q: %v", rawEmail, err)
		}
		if decoded != "user+tag@example.com" {
			t.Errorf("round-trip decoded EmailAddress = %q, want user+tag@example.com", decoded)
		}
		if strings.Contains(rawEmail, "+") {
			t.Errorf("'+' must be percent-encoded as %%2B in wire form, got: %s", rawEmail)
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

// TestListSuppressions_EmptyStreamID asserts that an empty streamID is rejected
// before any HTTP request is made, preventing silent URL construction bugs.
func TestListSuppressions_EmptyStreamID(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not have been made for empty streamID")
		return nil, nil
	})))

	_, err := api.ListSuppressions("", nil)
	if err == nil {
		t.Fatal("expected an error for empty streamID, got nil")
	}
	if !errors.Is(err, errEmptyStreamID) {
		t.Errorf("expected errEmptyStreamID, got: %v", err)
	}
}

// TestListSuppressions_StreamIDPathEscape verifies that a streamID containing
// characters that are special in URL paths (e.g. spaces) are properly
// percent-encoded by url.PathEscape, so the path segment remains well-formed.
// Note: http.NewRequest decodes percent-encoded sequences in URL.Path; the
// encoded form is preserved in URL.EscapedPath() and URL.RequestURI().
func TestListSuppressions_StreamIDPathEscape(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		// EscapedPath() returns the path with percent-encoding preserved, unlike
		// URL.Path which Go decodes after parsing. A space must appear as %20 in
		// the wire path, not as a literal space or as '+' (query-string encoding).
		escapedPath := req.URL.EscapedPath()
		if !strings.Contains(escapedPath, "/message-streams/my%20stream/suppressions/dump") {
			t.Errorf("expected percent-encoded streamID in EscapedPath, got: %s", escapedPath)
		}
		// URL.Path will contain the decoded form; verify the streamID round-trips correctly.
		if !strings.Contains(req.URL.Path, "/message-streams/my stream/suppressions/dump") {
			t.Errorf("expected decoded streamID in Path, got: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, ListSuppressionsResp{}),
		}, nil
	})))

	_, err := api.ListSuppressions("my stream", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestListSuppressions_APIError asserts that a non-2xx response returns a
// PostmarkErr so callers can inspect the error code.
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
	// PostmarkErr implements error with a value receiver and readResponse returns
	// it as a value, so errors.As with a value target is the correct pattern here.
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected error to be (or wrap) PostmarkErr, got: %T %v", err, err)
	}
	if pe.ErrorCode != pmErr.ErrorCode {
		t.Errorf("ErrorCode = %d, want %d", pe.ErrorCode, pmErr.ErrorCode)
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

// TestCreateSuppressions_EmptyStreamID asserts that an empty streamID is
// rejected before any HTTP request is made.
func TestCreateSuppressions_EmptyStreamID(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not have been made for empty streamID")
		return nil, nil
	})))

	_, err := api.CreateSuppressions("", &CreateSuppressionsReq{})
	if err == nil {
		t.Fatal("expected an error for empty streamID, got nil")
	}
	if !errors.Is(err, errEmptyStreamID) {
		t.Errorf("expected errEmptyStreamID, got: %v", err)
	}
}

// TestCreateSuppressions_APIError asserts that a non-2xx response returns a
// PostmarkErr so callers can inspect the error code.
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
	// PostmarkErr implements error with a value receiver and readResponse returns
	// it as a value, so errors.As with a value target is the correct pattern here.
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected error to be (or wrap) PostmarkErr, got: %T %v", err, err)
	}
	if pe.ErrorCode != pmErr.ErrorCode {
		t.Errorf("ErrorCode = %d, want %d", pe.ErrorCode, pmErr.ErrorCode)
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

// TestDeleteSuppressions_EmptyStreamID asserts that an empty streamID is
// rejected before any HTTP request is made.
func TestDeleteSuppressions_EmptyStreamID(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not have been made for empty streamID")
		return nil, nil
	})))

	_, err := api.DeleteSuppressions("", &DeleteSuppressionsReq{})
	if err == nil {
		t.Fatal("expected an error for empty streamID, got nil")
	}
	if !errors.Is(err, errEmptyStreamID) {
		t.Errorf("expected errEmptyStreamID, got: %v", err)
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

// TestDeleteSuppressions_APIError asserts that a non-2xx response returns a
// PostmarkErr so callers can inspect the error code.
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
	// PostmarkErr implements error with a value receiver and readResponse returns
	// it as a value, so errors.As with a value target is the correct pattern here.
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected error to be (or wrap) PostmarkErr, got: %T %v", err, err)
	}
	if pe.ErrorCode != pmErr.ErrorCode {
		t.Errorf("ErrorCode = %d, want %d", pe.ErrorCode, pmErr.ErrorCode)
	}
}
