package postmark

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ---- ListMessageStreams --------------------------------------------------------

func TestListMessageStreams_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := ListMessageStreamsResp{
		TotalCount: 2,
		MessageStreams: []MessageStream{
			{ID: "outbound", Name: "Outbound", MessageStreamType: "Transactional", ServerID: 1, CreatedAt: now, UpdatedAt: now},
			{ID: "broadcasts", Name: "Broadcasts", MessageStreamType: "Broadcasts", ServerID: 1, CreatedAt: now, UpdatedAt: now},
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
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			if req.Header.Get("X-Postmark-Account-Token") != "" {
				t.Error("expected X-Postmark-Account-Token header to NOT be set")
			}
			if !strings.Contains(req.URL.RawQuery, "IncludeArchivedStreams=true") {
				t.Errorf("expected IncludeArchivedStreams=true in query, got %s", req.URL.RawQuery)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.ListMessageStreams("", true)
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

// TestListMessageStreams_StreamTypeFilter exercises the streamType != "" branch,
// verifying that MessageStreamType is included in the query string.
func TestListMessageStreams_StreamTypeFilter(t *testing.T) {
	want := ListMessageStreamsResp{
		TotalCount:     1,
		MessageStreams: []MessageStream{{ID: "outbound", MessageStreamType: "Transactional", ServerID: 1}},
	}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.RawQuery, "MessageStreamType=Transactional") {
				t.Errorf("expected MessageStreamType=Transactional in query, got %s", req.URL.RawQuery)
			}
			if !strings.Contains(req.URL.RawQuery, "IncludeArchivedStreams=false") {
				t.Errorf("expected IncludeArchivedStreams=false in query, got %s", req.URL.RawQuery)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.ListMessageStreams("Transactional", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
}

func TestListMessageStreams_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal server error"}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.ListMessageStreams("Transactional", false)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected a PostmarkErr, got %T: %v", err, err)
	}
}

// ---- GetMessageStream ---------------------------------------------------------

func TestGetMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := MessageStream{
		ID:                "outbound",
		Name:              "Outbound",
		MessageStreamType: "Transactional",
		ServerID:          42,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.GetMessageStream("outbound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "outbound" {
		t.Errorf("ID = %q, want %q", got.ID, "outbound")
	}
	if got.ServerID != 42 {
		t.Errorf("ServerID = %d, want 42", got.ServerID)
	}
}

// TestGetMessageStream_NotFound asserts that a 404 response causes GetMessageStream
// to return ErrNotFound (the sentinel defined in servers.go), detectable via
// errors.Is. readResponse in api.go maps HTTP 404 → ErrNotFound directly.
func TestGetMessageStream_NotFound(t *testing.T) {
	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: http.StatusNotFound, Message: "Message stream not found"}),
			}, nil
		})),
	)

	_, err := api.GetMessageStream("nonexistent")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- CreateMessageStream ------------------------------------------------------

func TestCreateMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := MessageStream{
		ID:                "my-stream",
		Name:              "My Stream",
		Description:       "A test stream",
		MessageStreamType: "Broadcasts",
		ServerID:          10,
		CreatedAt:         now,
		UpdatedAt:         now,
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
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.CreateMessageStream(&CreateMessageStreamReq{
		ID:                "my-stream",
		Name:              "My Stream",
		Description:       "A test stream",
		MessageStreamType: "Broadcasts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "my-stream" {
		t.Errorf("ID = %q, want %q", got.ID, "my-stream")
	}
	if got.Name != "My Stream" {
		t.Errorf("Name = %q, want %q", got.Name, "My Stream")
	}
}

func TestCreateMessageStream_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 409, Message: "stream already exists"}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusConflict,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.CreateMessageStream(&CreateMessageStreamReq{ID: "outbound", Name: "Outbound"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, ErrExists) {
		t.Errorf("expected errors.Is(err, ErrExists) to be true, got err=%v", err)
	}
}

// ---- UpdateMessageStream ------------------------------------------------------

func TestUpdateMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := MessageStream{
		ID:                "outbound",
		Name:              "Updated Name",
		Description:       "Updated description",
		MessageStreamType: "Transactional",
		ServerID:          5,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPut {
				t.Errorf("expected PUT, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.UpdateMessageStream("outbound", &UpdateMessageStreamReq{
		Name:        "Updated Name",
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated Name")
	}
}

// TestUpdateMessageStream_NotFound asserts that a 404 response causes
// UpdateMessageStream to return ErrNotFound, detectable via errors.Is.
func TestUpdateMessageStream_NotFound(t *testing.T) {
	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: http.StatusNotFound, Message: "Message stream not found"}),
			}, nil
		})),
	)

	_, err := api.UpdateMessageStream("nonexistent", &UpdateMessageStreamReq{Name: "Ghost"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- ArchiveMessageStream -----------------------------------------------------

func TestArchiveMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	purgeDate := now.Add(30 * 24 * time.Hour)
	want := ArchiveMessageStreamResp{
		ID:                "outbound",
		ServerID:          7,
		ArchivedAt:        &now,
		ExpectedPurgeDate: &purgeDate,
	}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound/archive") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.ArchiveMessageStream("outbound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "outbound" {
		t.Errorf("ID = %q, want %q", got.ID, "outbound")
	}
	if got.ServerID != 7 {
		t.Errorf("ServerID = %d, want 7", got.ServerID)
	}
	if got.ArchivedAt == nil {
		t.Error("expected ArchivedAt to be non-nil")
	}
	if got.ExpectedPurgeDate == nil {
		t.Error("expected ExpectedPurgeDate to be non-nil")
	}
}

// TestArchiveMessageStream_NotFound asserts that a 404 response causes
// ArchiveMessageStream to return ErrNotFound, detectable via errors.Is.
func TestArchiveMessageStream_NotFound(t *testing.T) {
	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: http.StatusNotFound, Message: "Message stream not found"}),
			}, nil
		})),
	)

	_, err := api.ArchiveMessageStream("nonexistent")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- UnarchiveMessageStream ---------------------------------------------------

func TestUnarchiveMessageStream_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	want := MessageStream{
		ID:                "outbound",
		Name:              "Outbound",
		MessageStreamType: "Transactional",
		ServerID:          7,
		CreatedAt:         now,
		UpdatedAt:         now,
		ArchivedAt:        nil,
		ExpectedPurgeDate: nil,
	}

	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/message-streams/outbound/unarchive") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.UnarchiveMessageStream("outbound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "outbound" {
		t.Errorf("ID = %q, want %q", got.ID, "outbound")
	}
	if got.ArchivedAt != nil {
		t.Error("expected ArchivedAt to be nil after unarchive")
	}
}

// TestUnarchiveMessageStream_NotFound asserts that a 404 response causes
// UnarchiveMessageStream to return ErrNotFound, detectable via errors.Is.
func TestUnarchiveMessageStream_NotFound(t *testing.T) {
	api := New(
		APITokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: http.StatusNotFound, Message: "Message stream not found"}),
			}, nil
		})),
	)

	_, err := api.UnarchiveMessageStream("nonexistent")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}
