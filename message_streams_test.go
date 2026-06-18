package postmark

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ---- ListMessageStreams ---------------------------------------------------------

func TestListMessageStreams_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := ListMessageStreamsResp{
		TotalCount: 2,
		MessageStreams: []MessageStreamResp{
			{ID: "outbound", ServerID: 1, Name: "Outbound", MessageStreamType: MessageStreamTypeTransactional, CreatedAt: now},
			{ID: "broadcasts", ServerID: 1, Name: "Broadcasts", MessageStreamType: MessageStreamTypeBroadcasts, CreatedAt: now},
		},
	}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/message-streams") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			// Verify server token header is present, not account token.
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			if req.Header.Get("X-Postmark-Account-Token") != "" {
				t.Error("X-Postmark-Account-Token header must not be set for message-streams")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})))

	got, err := api.ListMessageStreams("", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
	if len(got.MessageStreams) != 2 {
		t.Errorf("len(MessageStreams) = %d, want 2", len(got.MessageStreams))
	}
	if got.MessageStreams[0].ID != "outbound" {
		t.Errorf("MessageStreams[0].ID = %q, want outbound", got.MessageStreams[0].ID)
	}
	// Use .Equal() to compare time.Time values: avoids location metadata mismatches.
	if !got.MessageStreams[0].CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got.MessageStreams[0].CreatedAt, now)
	}
}

func TestListMessageStreams_WithStreamTypeFilter(t *testing.T) {
	want := ListMessageStreamsResp{
		TotalCount:     1,
		MessageStreams: []MessageStreamResp{{ID: "outbound", MessageStreamType: MessageStreamTypeTransactional}},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		// Assert the MessageStreamType query parameter is actually present.
		if req.URL.Query().Get("MessageStreamType") != "Transactional" {
			t.Errorf("expected MessageStreamType=Transactional, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListMessageStreams("Transactional", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.MessageStreams) != 1 {
		t.Errorf("expected 1 stream, got %d", len(got.MessageStreams))
	}
}

func TestListMessageStreams_IncludeArchived(t *testing.T) {
	want := ListMessageStreamsResp{TotalCount: 0, MessageStreams: []MessageStreamResp{}}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("IncludeArchivedStreams") != "true" {
			t.Errorf("expected IncludeArchivedStreams=true param, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	_, err := api.ListMessageStreams("", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListMessageStreams_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.ListMessageStreams("", false)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected error to be PostmarkErr, got %T: %v", err, err)
	}
	if pe.ErrorCode != 500 {
		t.Errorf("ErrorCode = %d, want 500", pe.ErrorCode)
	}
}

// ---- GetMessageStream ----------------------------------------------------------

func TestGetMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := MessageStreamResp{
		ID:                "outbound",
		ServerID:          42,
		Name:              "Outbound",
		MessageStreamType: MessageStreamTypeTransactional,
		CreatedAt:         now,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetMessageStream("outbound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "outbound" {
		t.Errorf("ID = %q, want outbound", got.ID)
	}
	if got.ServerID != 42 {
		t.Errorf("ServerID = %d, want 42", got.ServerID)
	}
	if got.Name != "Outbound" {
		t.Errorf("Name = %q, want Outbound", got.Name)
	}
	if got.MessageStreamType != MessageStreamTypeTransactional {
		t.Errorf("MessageStreamType = %q, want Transactional", got.MessageStreamType)
	}
	// Use .Equal() to compare time.Time values: avoids location metadata mismatches.
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}
}

func TestGetMessageStream_EmptyID(t *testing.T) {
	api := New()
	_, err := api.GetMessageStream("")
	if !errors.Is(err, errEmptyStreamID) {
		t.Errorf("expected errEmptyStreamID, got %v", err)
	}
}

func TestGetMessageStream_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Stream not found"}),
		}, nil
	})))

	_, err := api.GetMessageStream("nonexistent")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

// ---- CreateMessageStream -------------------------------------------------------

func TestCreateMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := MessageStreamResp{
		ID:                "my-stream",
		ServerID:          1,
		Name:              "My Stream",
		MessageStreamType: MessageStreamTypeTransactional,
		CreatedAt:         now,
	}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/message-streams") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})))

	got, err := api.CreateMessageStream(&CreateMessageStreamReq{
		ID:                "my-stream",
		Name:              "My Stream",
		MessageStreamType: MessageStreamTypeTransactional,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "my-stream" {
		t.Errorf("ID = %q, want my-stream", got.ID)
	}
	if got.Name != "My Stream" {
		t.Errorf("Name = %q, want My Stream", got.Name)
	}
	// Use .Equal() to compare time.Time values: avoids location metadata mismatches.
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}
}

// TestCreateMessageStream_NilReq verifies that passing a nil request returns
// errNilCreateReq immediately without making a network call.
func TestCreateMessageStream_NilReq(t *testing.T) {
	api := New()
	_, err := api.CreateMessageStream(nil)
	if !errors.Is(err, errNilCreateReq) {
		t.Errorf("expected errNilCreateReq, got %v", err)
	}
}

// TestCreateMessageStream_WithSubscriptionConfig verifies that a non-nil
// SubscriptionManagementConfiguration pointer is serialised and the response
// is decoded correctly.
func TestCreateMessageStream_WithSubscriptionConfig(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	cfg := &SubscriptionManagementConfiguration{UnsubscribeHandlingType: "Custom"}
	want := MessageStreamResp{
		ID:                                  "broadcasts",
		ServerID:                            1,
		Name:                                "Broadcasts",
		MessageStreamType:                   MessageStreamTypeBroadcasts,
		CreatedAt:                           now,
		SubscriptionManagementConfiguration: cfg,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.CreateMessageStream(&CreateMessageStreamReq{
		ID:                                  "broadcasts",
		Name:                                "Broadcasts",
		MessageStreamType:                   MessageStreamTypeBroadcasts,
		SubscriptionManagementConfiguration: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SubscriptionManagementConfiguration == nil {
		t.Fatal("expected non-nil SubscriptionManagementConfiguration")
	}
	if got.SubscriptionManagementConfiguration.UnsubscribeHandlingType != "Custom" {
		t.Errorf("UnsubscribeHandlingType = %q, want Custom",
			got.SubscriptionManagementConfiguration.UnsubscribeHandlingType)
	}
}

func TestCreateMessageStream_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateMessageStream(&CreateMessageStreamReq{
		ID:                "bad",
		Name:              "Bad",
		MessageStreamType: MessageStreamTypeTransactional,
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected PostmarkErr, got %T: %v", err, err)
	}
}

// ---- UpdateMessageStream -------------------------------------------------------

func TestUpdateMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := MessageStreamResp{
		ID:        "outbound",
		ServerID:  1,
		Name:      "Updated Name",
		CreatedAt: now,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.UpdateMessageStream("outbound", &UpdateMessageStreamReq{
		Name: "Updated Name",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Updated Name" {
		t.Errorf("Name = %q, want Updated Name", got.Name)
	}
}

func TestUpdateMessageStream_EmptyID(t *testing.T) {
	api := New()
	_, err := api.UpdateMessageStream("", &UpdateMessageStreamReq{Name: "X"})
	if !errors.Is(err, errEmptyStreamID) {
		t.Errorf("expected errEmptyStreamID, got %v", err)
	}
}

// TestUpdateMessageStream_NilReq verifies that passing a nil request returns
// errNilUpdateReq immediately without making a network call.
func TestUpdateMessageStream_NilReq(t *testing.T) {
	api := New()
	_, err := api.UpdateMessageStream("outbound", nil)
	if !errors.Is(err, errNilUpdateReq) {
		t.Errorf("expected errNilUpdateReq, got %v", err)
	}
}

func TestUpdateMessageStream_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Stream not found"}),
		}, nil
	})))

	_, err := api.UpdateMessageStream("ghost", &UpdateMessageStreamReq{Name: "Ghost"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

// ---- ArchiveMessageStream ------------------------------------------------------

func TestArchiveMessageStream_Success(t *testing.T) {
	expungeAt := time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Second)
	want := MessageStreamArchiveResp{
		ID:        "outbound",
		ServerID:  1,
		ExpungeAt: &expungeAt,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound/archive") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		// Verify Content-Type is set (non-nil body `{}` was sent).
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", req.Header.Get("Content-Type"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ArchiveMessageStream("outbound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "outbound" {
		t.Errorf("ID = %q, want outbound", got.ID)
	}
	if got.ExpungeAt == nil {
		t.Fatal("expected ExpungeAt to be non-nil")
	}
	// Use .Equal() to compare time.Time values: avoids location metadata mismatches.
	if !got.ExpungeAt.Equal(expungeAt) {
		t.Errorf("ExpungeAt = %v, want %v", got.ExpungeAt, expungeAt)
	}
}

func TestArchiveMessageStream_EmptyID(t *testing.T) {
	api := New()
	_, err := api.ArchiveMessageStream("")
	if !errors.Is(err, errEmptyStreamID) {
		t.Errorf("expected errEmptyStreamID, got %v", err)
	}
}

func TestArchiveMessageStream_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Stream not found"}),
		}, nil
	})))

	_, err := api.ArchiveMessageStream("nonexistent")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

// ---- UnarchiveMessageStream ----------------------------------------------------

func TestUnarchiveMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := MessageStreamResp{
		ID:        "outbound",
		ServerID:  1,
		Name:      "Outbound",
		CreatedAt: now,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound/unarchive") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		// Verify Content-Type is set (non-nil body `{}` was sent).
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", req.Header.Get("Content-Type"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.UnarchiveMessageStream("outbound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "outbound" {
		t.Errorf("ID = %q, want outbound", got.ID)
	}
	// Use .Equal() to compare time.Time values: avoids location metadata mismatches.
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}
}

func TestUnarchiveMessageStream_EmptyID(t *testing.T) {
	api := New()
	_, err := api.UnarchiveMessageStream("")
	if !errors.Is(err, errEmptyStreamID) {
		t.Errorf("expected errEmptyStreamID, got %v", err)
	}
}

func TestUnarchiveMessageStream_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Stream not found"}),
		}, nil
	})))

	_, err := api.UnarchiveMessageStream("nonexistent")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

// ---- ListSuppressions ----------------------------------------------------------

func TestListSuppressions_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := SuppressionsResp{
		Suppressions: []SuppressionEntry{
			{
				EmailAddress:      "suppressed@example.com",
				SuppressionReason: "HardBounce",
				Origin:            "Recipient",
				CreatedAt:         now,
			},
		},
	}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound/suppressions/dump") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})))

	got, err := api.ListSuppressions("outbound", SuppressionsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Suppressions) != 1 {
		t.Errorf("expected 1 suppression, got %d", len(got.Suppressions))
	}
	if got.Suppressions[0].EmailAddress != "suppressed@example.com" {
		t.Errorf("EmailAddress = %q, want suppressed@example.com", got.Suppressions[0].EmailAddress)
	}
	if got.Suppressions[0].SuppressionReason != "HardBounce" {
		t.Errorf("SuppressionReason = %q, want HardBounce", got.Suppressions[0].SuppressionReason)
	}
	if got.Suppressions[0].Origin != "Recipient" {
		t.Errorf("Origin = %q, want Recipient", got.Suppressions[0].Origin)
	}
	// Use .Equal() to compare time.Time values: avoids location metadata mismatches.
	if !got.Suppressions[0].CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got.Suppressions[0].CreatedAt, now)
	}
}

func TestListSuppressions_EmptyID(t *testing.T) {
	api := New()
	_, err := api.ListSuppressions("", SuppressionsParams{})
	if !errors.Is(err, errEmptyStreamID) {
		t.Errorf("expected errEmptyStreamID, got %v", err)
	}
}

func TestListSuppressions_WithParams(t *testing.T) {
	want := SuppressionsResp{Suppressions: []SuppressionEntry{}}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.Query()
		if q.Get("SuppressionReason") != "HardBounce" {
			t.Errorf("expected SuppressionReason=HardBounce, query=%s", req.URL.RawQuery)
		}
		if q.Get("Origin") != "Recipient" {
			t.Errorf("expected Origin=Recipient, query=%s", req.URL.RawQuery)
		}
		if q.Get("EmailAddress") != "test@example.com" {
			t.Errorf("expected EmailAddress=test@example.com, query=%s", req.URL.RawQuery)
		}
		if q.Get("FromDate") != "2024-01-01" {
			t.Errorf("expected FromDate=2024-01-01, query=%s", req.URL.RawQuery)
		}
		if q.Get("ToDate") != "2024-12-31" {
			t.Errorf("expected ToDate=2024-12-31, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	_, err := api.ListSuppressions("outbound", SuppressionsParams{
		SuppressionReason: "HardBounce",
		Origin:            "Recipient",
		EmailAddress:      "test@example.com",
		FromDate:          "2024-01-01",
		ToDate:            "2024-12-31",
	})
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

	_, err := api.ListSuppressions("outbound", SuppressionsParams{})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected PostmarkErr, got %T: %v", err, err)
	}
	if pe.ErrorCode != 500 {
		t.Errorf("ErrorCode = %d, want 500", pe.ErrorCode)
	}
}

// ---- CreateSuppression ---------------------------------------------------------

func TestCreateSuppression_Success(t *testing.T) {
	want := SuppressionResp{
		Suppressions: []SuppressionResult{
			{EmailAddress: "new@example.com", Status: "Suppressed"},
		},
	}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound/suppressions") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})))

	got, err := api.CreateSuppression("outbound", &CreateSuppressionReq{
		Suppressions: []SuppressionAddress{{EmailAddress: "new@example.com"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Suppressions) != 1 {
		t.Errorf("expected 1 result, got %d", len(got.Suppressions))
	}
	if got.Suppressions[0].Status != "Suppressed" {
		t.Errorf("Status = %q, want Suppressed", got.Suppressions[0].Status)
	}
	if got.Suppressions[0].EmailAddress != "new@example.com" {
		t.Errorf("EmailAddress = %q, want new@example.com", got.Suppressions[0].EmailAddress)
	}
}

func TestCreateSuppression_EmptyID(t *testing.T) {
	api := New()
	_, err := api.CreateSuppression("", &CreateSuppressionReq{
		Suppressions: []SuppressionAddress{{EmailAddress: "x@example.com"}},
	})
	if !errors.Is(err, errEmptyStreamID) {
		t.Errorf("expected errEmptyStreamID, got %v", err)
	}
}

func TestCreateSuppression_Multiple(t *testing.T) {
	want := SuppressionResp{
		Suppressions: []SuppressionResult{
			{EmailAddress: "a@example.com", Status: "Suppressed"},
			{EmailAddress: "b@example.com", Status: "Suppressed"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.CreateSuppression("outbound", &CreateSuppressionReq{
		Suppressions: []SuppressionAddress{
			{EmailAddress: "a@example.com"},
			{EmailAddress: "b@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Suppressions) != 2 {
		t.Errorf("expected 2 results, got %d", len(got.Suppressions))
	}
}

// ---- DeleteSuppression ---------------------------------------------------------

func TestDeleteSuppression_Success(t *testing.T) {
	want := SuppressionResp{
		Suppressions: []SuppressionResult{
			{EmailAddress: "old@example.com", Status: "Deleted"},
		},
	}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound/suppressions/delete") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})))

	got, err := api.DeleteSuppression("outbound", &DeleteSuppressionReq{
		Suppressions: []SuppressionAddress{{EmailAddress: "old@example.com"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Suppressions) != 1 {
		t.Errorf("expected 1 result, got %d", len(got.Suppressions))
	}
	if got.Suppressions[0].Status != "Deleted" {
		t.Errorf("Status = %q, want Deleted", got.Suppressions[0].Status)
	}
	if got.Suppressions[0].EmailAddress != "old@example.com" {
		t.Errorf("EmailAddress = %q, want old@example.com", got.Suppressions[0].EmailAddress)
	}
}

func TestDeleteSuppression_EmptyID(t *testing.T) {
	api := New()
	_, err := api.DeleteSuppression("", &DeleteSuppressionReq{
		Suppressions: []SuppressionAddress{{EmailAddress: "x@example.com"}},
	})
	if !errors.Is(err, errEmptyStreamID) {
		t.Errorf("expected errEmptyStreamID, got %v", err)
	}
}

func TestDeleteSuppression_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Stream not found"}),
		}, nil
	})))

	_, err := api.DeleteSuppression("nonexistent", &DeleteSuppressionReq{
		Suppressions: []SuppressionAddress{{EmailAddress: "old@example.com"}},
	})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

func TestDeleteSuppression_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.DeleteSuppression("outbound", &DeleteSuppressionReq{
		Suppressions: []SuppressionAddress{{EmailAddress: "bad@example.com"}},
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected PostmarkErr, got %T: %v", err, err)
	}
	if pe.ErrorCode != 500 {
		t.Errorf("ErrorCode = %d, want 500", pe.ErrorCode)
	}
}
