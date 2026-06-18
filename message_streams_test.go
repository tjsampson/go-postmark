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

// TestListMessageStreams_FalseNeverSendsParam verifies that passing any value
// other than "true" omits the includeArchived query parameter entirely (the API default is false).
func TestListMessageStreams_FalseNeverSendsParam(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.RawQuery != "" {
			t.Errorf("expected no query params when includeArchivedStr is not \"true\", got %s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, ListMessageStreamsResp{}),
		}, nil
	})))

	_, err := api.ListMessageStreams("false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected a PostmarkErr, got %T: %v", err, err)
	}
}

// ---- GetMessageStream ----------------------------------------------------------

func TestGetMessageStream_Success(t *testing.T) {
	createdAt := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	want := MessageStreamResp{
		ID:                "outbound",
		ServerID:          1,
		Name:              "Outbound",
		Description:       "Default transactional stream",
		MessageStreamType: "Transactional",
		CreatedAt:         createdAt,
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
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, want.CreatedAt)
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

func TestGetMessageStream_EmptyStreamID(t *testing.T) {
	api := New()
	_, err := api.GetMessageStream("")
	if err == nil {
		t.Fatal("expected error for empty streamID, got nil")
	}
}

// ---- CreateMessageStream -------------------------------------------------------

func TestCreateMessageStream_Success(t *testing.T) {
	createdAt := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	want := MessageStreamResp{
		ID:                "my-stream",
		ServerID:          1,
		Name:              "My Stream",
		Description:       "A custom stream",
		MessageStreamType: "Transactional",
		CreatedAt:         createdAt,
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
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected a PostmarkErr, got %T: %v", err, err)
	}
}

// TestCreateMessageStream_NilReq verifies that a nil request returns an error
// without making an HTTP call.
func TestCreateMessageStream_NilReq(t *testing.T) {
	api := New()
	_, err := api.CreateMessageStream(nil)
	if err == nil {
		t.Fatal("expected error for nil req, got nil")
	}
}

// TestCreateMessageStream_MissingRequiredFields verifies that empty required
// fields are rejected before an HTTP call is made.
func TestCreateMessageStream_MissingRequiredFields(t *testing.T) {
	api := New()

	tests := []struct {
		name string
		req  *CreateMessageStreamReq
	}{
		{"missing ID", &CreateMessageStreamReq{Name: "N", MessageStreamType: "Transactional"}},
		{"missing Name", &CreateMessageStreamReq{ID: "id", MessageStreamType: "Transactional"}},
		{"missing MessageStreamType", &CreateMessageStreamReq{ID: "id", Name: "N"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := api.CreateMessageStream(tc.req)
			if err == nil {
				t.Errorf("expected validation error for %q, got nil", tc.name)
			}
		})
	}
}

// ---- EditMessageStream ---------------------------------------------------------

func TestEditMessageStream_Success(t *testing.T) {
	createdAt := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	want := MessageStreamResp{
		ID:                "my-stream",
		ServerID:          1,
		Name:              "Renamed Stream",
		Description:       "Updated description",
		MessageStreamType: "Transactional",
		CreatedAt:         createdAt,
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

// TestEditMessageStream_EmptyReq verifies that an empty EditMessageStreamReq
// (all omitempty fields absent) still reaches the endpoint with an empty JSON
// body ({}) and does not silently become a no-op at the HTTP layer.
func TestEditMessageStream_EmptyReq(t *testing.T) {
	want := MessageStreamResp{
		ID:   "my-stream",
		Name: "My Stream",
	}
	requestReached := false

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		requestReached = true
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

	got, err := api.EditMessageStream("my-stream", &EditMessageStreamReq{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !requestReached {
		t.Error("expected an HTTP request to be made, but none was")
	}
	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
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

func TestEditMessageStream_EmptyStreamID(t *testing.T) {
	api := New()
	_, err := api.EditMessageStream("", &EditMessageStreamReq{Name: "Ghost"})
	if err == nil {
		t.Fatal("expected error for empty streamID, got nil")
	}
}

// ---- ArchiveMessageStream ------------------------------------------------------

func TestArchiveMessageStream_Success(t *testing.T) {
	archivedAt := time.Date(2021, 7, 1, 0, 0, 0, 0, time.UTC)
	expungeAt := time.Date(2021, 8, 1, 0, 0, 0, 0, time.UTC)
	want := ArchiveMessageStreamResp{
		ID:          "my-stream",
		ServerID:    1,
		Name:        "My Stream",
		Description: "A custom stream",
		ArchivedAt:  &archivedAt,
		ExpungeAt:   &expungeAt,
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
	if got.ArchivedAt == nil || !got.ArchivedAt.Equal(archivedAt) {
		t.Errorf("ArchivedAt = %v, want %v", got.ArchivedAt, archivedAt)
	}
	if got.ExpungeAt == nil || !got.ExpungeAt.Equal(expungeAt) {
		t.Errorf("ExpungeAt = %v, want %v", got.ExpungeAt, expungeAt)
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

func TestArchiveMessageStream_EmptyStreamID(t *testing.T) {
	api := New()
	_, err := api.ArchiveMessageStream("")
	if err == nil {
		t.Fatal("expected error for empty streamID, got nil")
	}
}

// ---- UnarchiveMessageStream ----------------------------------------------------

func TestUnarchiveMessageStream_Success(t *testing.T) {
	createdAt := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	want := MessageStreamResp{
		ID:                "my-stream",
		ServerID:          1,
		Name:              "My Stream",
		Description:       "A custom stream",
		MessageStreamType: "Transactional",
		CreatedAt:         createdAt,
		// ArchivedAt and ExpungeAt must be nil after a successful unarchive.
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
	// The distinguishing postcondition of a successful unarchive is that
	// ArchivedAt and ExpungeAt are nil.
	if got.ArchivedAt != nil {
		t.Errorf("ArchivedAt should be nil after unarchive, got %v", *got.ArchivedAt)
	}
	if got.ExpungeAt != nil {
		t.Errorf("ExpungeAt should be nil after unarchive, got %v", *got.ExpungeAt)
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

func TestUnarchiveMessageStream_EmptyStreamID(t *testing.T) {
	api := New()
	_, err := api.UnarchiveMessageStream("")
	if err == nil {
		t.Fatal("expected error for empty streamID, got nil")
	}
}

// ---- SubscriptionManagementConfiguration --------------------------------------

func TestGetMessageStream_WithSubscriptionManagement(t *testing.T) {
	smc := &SubscriptionManagementConfiguration{UnsubscribeHandlingType: UnsubscribeHandlingCustom}
	want := MessageStreamResp{
		ID:                                  "broadcasts",
		ServerID:                            1,
		Name:                                "Broadcasts",
		MessageStreamType:                   "Broadcasts",
		CreatedAt:                           time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
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
