package postmark

import (
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
			{ID: "outbound", ServerID: 1, Name: "Outbound", MessageStreamType: "Transactional", CreatedAt: now},
			{ID: "broadcasts", ServerID: 1, Name: "Broadcasts", MessageStreamType: "Broadcasts", CreatedAt: now},
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
}

func TestListMessageStreams_WithStreamTypeFilter(t *testing.T) {
	want := ListMessageStreamsResp{
		TotalCount:     1,
		MessageStreams: []MessageStreamResp{{ID: "outbound", MessageStreamType: "Transactional"}},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.RawQuery, "MessageStreamType=Transactional") {
			t.Errorf("expected MessageStreamType query param, query=%s", req.URL.RawQuery)
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
		if !strings.Contains(req.URL.RawQuery, "IncludeArchivedStreams=true") {
			t.Errorf("expected IncludeArchivedStreams param, query=%s", req.URL.RawQuery)
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
}

// ---- GetMessageStream ----------------------------------------------------------

func TestGetMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := MessageStreamResp{
		ID:                "outbound",
		ServerID:          42,
		Name:              "Outbound",
		MessageStreamType: "Transactional",
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
}

// ---- CreateMessageStream -------------------------------------------------------

func TestCreateMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := MessageStreamResp{
		ID:                "my-stream",
		ServerID:          1,
		Name:              "My Stream",
		MessageStreamType: "Transactional",
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
		MessageStreamType: "Transactional",
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
		MessageStreamType: "Transactional",
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
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
		t.Error("expected ExpungeAt to be non-nil")
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
		t.Errorf("EmailAddress = %q", got.Suppressions[0].EmailAddress)
	}
}

func TestListSuppressions_WithParams(t *testing.T) {
	want := SuppressionsResp{Suppressions: []SuppressionEntry{}}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.RawQuery
		if !strings.Contains(q, "SuppressionReason=HardBounce") {
			t.Errorf("expected SuppressionReason param, query=%s", q)
		}
		if !strings.Contains(q, "Origin=Recipient") {
			t.Errorf("expected Origin param, query=%s", q)
		}
		if !strings.Contains(q, "EmailAddress=test%40example.com") {
			t.Errorf("expected EmailAddress param, query=%s", q)
		}
		if !strings.Contains(q, "FromDate=2024-01-01") {
			t.Errorf("expected FromDate param, query=%s", q)
		}
		if !strings.Contains(q, "ToDate=2024-12-31") {
			t.Errorf("expected ToDate param, query=%s", q)
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
}
