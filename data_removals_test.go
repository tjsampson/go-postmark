package postmark

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// ---- RequestDataRemoval -----------------------------------------------------

func TestRequestDataRemoval_Success(t *testing.T) {
	want := DataRemovalResp{
		ID:           101,
		EmailAddress: "user@example.com",
		Status:       "Pending",
		RequestedAt:  "2024-01-15T12:00:00Z",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "data-removals") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		// Verify request body
		var body map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["EmailAddress"] != "user@example.com" {
			t.Errorf("expected EmailAddress=user@example.com, got %v", body["EmailAddress"])
		}
		if body["RequestedBy"] != "admin@example.com" {
			t.Errorf("expected RequestedBy=admin@example.com, got %v", body["RequestedBy"])
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.RequestDataRemoval(&DataRemovalReq{
		EmailAddress: "user@example.com",
		RequestedBy:  "admin@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 101 {
		t.Errorf("ID = %d, want 101", got.ID)
	}
	if got.EmailAddress != "user@example.com" {
		t.Errorf("EmailAddress = %q, want user@example.com", got.EmailAddress)
	}
	if got.Status != "Pending" {
		t.Errorf("Status = %q, want Pending", got.Status)
	}
	if got.RequestedAt != "2024-01-15T12:00:00Z" {
		t.Errorf("RequestedAt = %q, want 2024-01-15T12:00:00Z", got.RequestedAt)
	}
}

func TestRequestDataRemoval_Various(t *testing.T) {
	tests := []struct {
		name string
		req  *DataRemovalReq
	}{
		{
			name: "basic removal request",
			req:  &DataRemovalReq{EmailAddress: "alice@example.com", RequestedBy: "admin@example.com"},
		},
		{
			name: "removal with different emails",
			req:  &DataRemovalReq{EmailAddress: "bob@company.org", RequestedBy: "gdpr@company.org"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expectedEmail := tc.req.EmailAddress
			expectedBy := tc.req.RequestedBy
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				if body["EmailAddress"] != expectedEmail {
					t.Errorf("expected EmailAddress=%q, got %v", expectedEmail, body["EmailAddress"])
				}
				if body["RequestedBy"] != expectedBy {
					t.Errorf("expected RequestedBy=%q, got %v", expectedBy, body["RequestedBy"])
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: jsonBody(t, DataRemovalResp{
						ID:           1,
						EmailAddress: expectedEmail,
						Status:       "Pending",
						RequestedAt:  "2024-01-15T00:00:00Z",
					}),
				}, nil
			})))
			got, err := api.RequestDataRemoval(tc.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.EmailAddress != expectedEmail {
				t.Errorf("EmailAddress = %q, want %q", got.EmailAddress, expectedEmail)
			}
		})
	}
}

func TestRequestDataRemoval_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.RequestDataRemoval(&DataRemovalReq{EmailAddress: "user@example.com"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- GetDataRemoval ---------------------------------------------------------

func TestGetDataRemoval_Success(t *testing.T) {
	want := DataRemovalResp{
		ID:           202,
		EmailAddress: "remove@example.com",
		Status:       "Completed",
		RequestedAt:  "2024-01-10T08:30:00Z",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "data-removals/202") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetDataRemoval(202)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 202 {
		t.Errorf("ID = %d, want 202", got.ID)
	}
	if got.EmailAddress != "remove@example.com" {
		t.Errorf("EmailAddress = %q, want remove@example.com", got.EmailAddress)
	}
	if got.Status != "Completed" {
		t.Errorf("Status = %q, want Completed", got.Status)
	}
}

func TestGetDataRemoval_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "removal not found"}),
		}, nil
	})))

	_, err := api.GetDataRemoval(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestGetDataRemoval_PathContainsID(t *testing.T) {
	tests := []struct {
		name      string
		removalID int64
		wantPath  string
	}{
		{name: "id 1", removalID: 1, wantPath: "data-removals/1"},
		{name: "id 500", removalID: 500, wantPath: "data-removals/500"},
		{name: "large id", removalID: 987654321, wantPath: "data-removals/987654321"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				if !strings.HasSuffix(req.URL.Path, tc.wantPath) {
					t.Errorf("expected path suffix %s, got %s", tc.wantPath, req.URL.Path)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: jsonBody(t, DataRemovalResp{
						ID:     tc.removalID,
						Status: "Pending",
					}),
				}, nil
			})))
			got, err := api.GetDataRemoval(tc.removalID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tc.removalID {
				t.Errorf("ID = %d, want %d", got.ID, tc.removalID)
			}
		})
	}
}

func TestGetDataRemoval_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "internal error"}),
		}, nil
	})))

	_, err := api.GetDataRemoval(1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
