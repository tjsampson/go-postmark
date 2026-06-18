package postmark

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// ---- ListMessageStreams --------------------------------------------------------

func TestListMessageStreams_Success(t *testing.T) {
	want := ListMessageStreamsResp{
		TotalCount: 2,
		MessageStreams: []MessageStreamResp{
			{ID: "outbound", ServerID: 1, Name: "Outbound", MessageStreamType: "Transactional"},
			{ID: "broadcasts", ServerID: 1, Name: "Broadcasts", MessageStreamType: "Broadcasts"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if req.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListMessageStreams("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != want.TotalCount {
		t.Errorf("TotalCount = %d, want %d", got.TotalCount, want.TotalCount)
	}
	if len(got.MessageStreams) != len(want.MessageStreams) {
		t.Errorf("len(MessageStreams) = %d, want %d", len(got.MessageStreams), len(want.MessageStreams))
	}
}

func TestListMessageStreams_WithIncludeArchived(t *testing.T) {
	want := ListMessageStreamsResp{
		TotalCount:     1,
		MessageStreams: []MessageStreamResp{{ID: "archived-stream", ServerID: 1, Name: "Archived"}},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.RawQuery, "includeArchived=true") {
			t.Errorf("expected includeArchived=true in query, got %s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListMessageStreams("true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != want.TotalCount {
		t.Errorf("TotalCount = %d, want %d", got.TotalCount, want.TotalCount)
	}
}

func TestListMessageStreams_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.ListMessageStreams("")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetMessageStream ----------------------------------------------------------

func TestGetMessageStream_Success(t *testing.T) {
	want := MessageStreamResp{
		ID:                "outbound",
		ServerID:          1,
		Name:              "Outbound",
		Description:       "Default transactional stream",
		MessageStreamType: "Transactional",
		CreatedAt:         "2021-01-01T00:00:00Z",
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
	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.ServerID != want.ServerID {
		t.Errorf("ServerID = %d, want %d", got.ServerID, want.ServerID)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}

func TestGetMessageStream_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "server not found"}),
		}, nil
	})))

	_, err := api.GetMessageStream("nonexistent")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- CreateMessageStream -------------------------------------------------------

func TestCreateMessageStream_Success(t *testing.T) {
	want := MessageStreamResp{
		ID:                "my-stream",
		ServerID:          1,
		Name:              "My Stream",
		Description:       "A custom stream",
		MessageStreamType: "Transactional",
		CreatedAt:         "2021-06-01T00:00:00Z",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams") {
			t.Errorf("unexpected path: %s", req.URL.Path)
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
		Description:       "A custom stream",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.MessageStreamType != want.MessageStreamType {
		t.Errorf("MessageStreamType = %q, want %q", got.MessageStreamType, want.MessageStreamType)
	}
}

func TestCreateMessageStream_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 400, Message: "Bad request"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateMessageStream(&CreateMessageStreamReq{
		ID:                "bad",
		Name:              "Bad",
		MessageStreamType: "Invalid",
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- EditMessageStream ---------------------------------------------------------

func TestEditMessageStream_Success(t *testing.T) {
	want := MessageStreamResp{
		ID:                "my-stream",
		ServerID:          1,
		Name:              "Renamed Stream",
		Description:       "Updated description",
		MessageStreamType: "Transactional",
		CreatedAt:         "2021-06-01T00:00:00Z",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams/my-stream") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.EditMessageStream("my-stream", &EditMessageStreamReq{
		Name:        "Renamed Stream",
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Description != want.Description {
		t.Errorf("Description = %q, want %q", got.Description, want.Description)
	}
}

func TestEditMessageStream_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "server not found"}),
		}, nil
	})))

	_, err := api.EditMessageStream("nonexistent", &EditMessageStreamReq{Name: "Ghost"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- ArchiveMessageStream ------------------------------------------------------

func TestArchiveMessageStream_Success(t *testing.T) {
	want := ArchiveMessageStreamResp{
		ID:          "my-stream",
		ServerID:    1,
		Name:        "My Stream",
		Description: "A custom stream",
		ArchivedAt:  "2021-07-01T00:00:00Z",
		ExpungeAt:   "2021-08-01T00:00:00Z",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams/my-stream/archive") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ArchiveMessageStream("my-stream")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.ArchivedAt != want.ArchivedAt {
		t.Errorf("ArchivedAt = %q, want %q", got.ArchivedAt, want.ArchivedAt)
	}
	if got.ExpungeAt != want.ExpungeAt {
		t.Errorf("ExpungeAt = %q, want %q", got.ExpungeAt, want.ExpungeAt)
	}
}

func TestArchiveMessageStream_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "server not found"}),
		}, nil
	})))

	_, err := api.ArchiveMessageStream("nonexistent")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- UnarchiveMessageStream ----------------------------------------------------

func TestUnarchiveMessageStream_Success(t *testing.T) {
	want := MessageStreamResp{
		ID:                "my-stream",
		ServerID:          1,
		Name:              "My Stream",
		Description:       "A custom stream",
		MessageStreamType: "Transactional",
		CreatedAt:         "2021-06-01T00:00:00Z",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/message-streams/my-stream/unarchive") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.UnarchiveMessageStream("my-stream")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}

func TestUnarchiveMessageStream_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "server not found"}),
		}, nil
	})))

	_, err := api.UnarchiveMessageStream("nonexistent")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- SubscriptionManagementConfiguration --------------------------------------

func TestGetMessageStream_WithSubscriptionManagement(t *testing.T) {
	smc := &SubscriptionManagementConfiguration{UnsubscribeHandlingType: "Custom"}
	want := MessageStreamResp{
		ID:                                  "broadcasts",
		ServerID:                            1,
		Name:                                "Broadcasts",
		MessageStreamType:                   "Broadcasts",
		CreatedAt:                           "2021-01-01T00:00:00Z",
		SubscriptionManagementConfiguration: smc,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetMessageStream("broadcasts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SubscriptionManagementConfiguration == nil {
		t.Fatal("expected SubscriptionManagementConfiguration to be non-nil")
	}
	if got.SubscriptionManagementConfiguration.UnsubscribeHandlingType != smc.UnsubscribeHandlingType {
		t.Errorf("UnsubscribeHandlingType = %q, want %q",
			got.SubscriptionManagementConfiguration.UnsubscribeHandlingType,
			smc.UnsubscribeHandlingType)
	}
}
