package postmark

import (
	"net/http"
	"strings"
	"testing"
)

// ---- ListBounces ---------------------------------------------------------------

func TestListBounces_Success(t *testing.T) {
	want := ListBouncesResp{
		TotalCount: 2,
		Bounces: []BounceResp{
			{ID: 1, Type: "HardBounce", Email: "user1@example.com"},
			{ID: 2, Type: "SoftBounce", Email: "user2@example.com"},
		},
	}

	api := New(APITokenOpt("test-server-token"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/bounces") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if req.Header.Get("X-Postmark-Server-Token") == "" {
			t.Error("expected X-Postmark-Server-Token header to be set")
		}
		if req.Header.Get("X-Postmark-Account-Token") != "" {
			t.Error("X-Postmark-Account-Token header must not be set on bounce requests")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListBounces(ListBouncesParams{Count: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
	if len(got.Bounces) != 2 {
		t.Errorf("len(Bounces) = %d, want 2", len(got.Bounces))
	}
}

func TestListBounces_WithParams(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.RawQuery
		if !strings.Contains(q, "type=HardBounce") {
			t.Errorf("expected type param in query: %s", q)
		}
		if !strings.Contains(q, "emailFilter=test%40example.com") {
			t.Errorf("expected emailFilter param in query: %s", q)
		}
		if !strings.Contains(q, "inactive=true") {
			t.Errorf("expected inactive param in query: %s", q)
		}
		if !strings.Contains(q, "count=25") {
			t.Errorf("expected count param in query: %s", q)
		}
		if !strings.Contains(q, "offset=5") {
			t.Errorf("expected offset param in query: %s", q)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, ListBouncesResp{}),
		}, nil
	})))

	_, err := api.ListBounces(ListBouncesParams{
		Type:        "HardBounce",
		Inactive:    true,
		EmailFilter: "test@example.com",
		Count:       25,
		Offset:      5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListBounces_AllParams(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.RawQuery
		if !strings.Contains(q, "tag=welcome") {
			t.Errorf("expected tag param in query: %s", q)
		}
		if !strings.Contains(q, "messageID=abc123") {
			t.Errorf("expected messageID param in query: %s", q)
		}
		if !strings.Contains(q, "fromdate=2024-01-01") {
			t.Errorf("expected fromdate param in query: %s", q)
		}
		if !strings.Contains(q, "todate=2024-12-31") {
			t.Errorf("expected todate param in query: %s", q)
		}
		if !strings.Contains(q, "messagestream=outbound") {
			t.Errorf("expected messagestream param in query: %s", q)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, ListBouncesResp{}),
		}, nil
	})))

	_, err := api.ListBounces(ListBouncesParams{
		Tag:           "welcome",
		MessageID:     "abc123",
		FromDate:      "2024-01-01",
		ToDate:        "2024-12-31",
		MessageStream: "outbound",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListBounces_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.ListBounces(ListBouncesParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- GetBounce -----------------------------------------------------------------

func TestGetBounce_Success(t *testing.T) {
	want := BounceResp{
		ID:            42,
		Type:          "HardBounce",
		TypeCode:      1,
		Name:          "Hard bounce",
		Tag:           "welcome",
		MessageID:     "msg-123",
		ServerID:      7,
		Description:   "The server was unable to deliver your message.",
		Details:       "smtp; 550 5.1.1 user unknown",
		Email:         "bounce@example.com",
		From:          "sender@example.com",
		BouncedAt:     "2024-01-15T10:00:00Z",
		DumpAvailable: true,
		Inactive:      true,
		CanActivate:   true,
		Subject:       "Welcome!",
	}

	api := New(APITokenOpt("test-server-token"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/bounces/42") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if req.Header.Get("X-Postmark-Server-Token") == "" {
			t.Error("expected X-Postmark-Server-Token header to be set")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetBounce(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("ID = %d, want 42", got.ID)
	}
	if got.Type != "HardBounce" {
		t.Errorf("Type = %q, want HardBounce", got.Type)
	}
	if got.Email != "bounce@example.com" {
		t.Errorf("Email = %q, want bounce@example.com", got.Email)
	}
	if !got.DumpAvailable {
		t.Error("DumpAvailable should be true")
	}
	if !got.Inactive {
		t.Error("Inactive should be true")
	}
	if !got.CanActivate {
		t.Error("CanActivate should be true")
	}
}

func TestGetBounce_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "bounce not found"}),
		}, nil
	})))

	_, err := api.GetBounce(9999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetBounce_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.GetBounce(1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- GetBounceDump -------------------------------------------------------------

func TestGetBounceDump_Success(t *testing.T) {
	want := BounceDumpResp{Body: "Received: from mail.example.com\r\nSubject: Test\r\n\r\nBody text"}

	api := New(APITokenOpt("test-server-token"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/bounces/42/dump") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if req.Header.Get("X-Postmark-Server-Token") == "" {
			t.Error("expected X-Postmark-Server-Token header to be set")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetBounceDump(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Body != want.Body {
		t.Errorf("Body = %q, want %q", got.Body, want.Body)
	}
}

func TestGetBounceDump_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "not found"}),
		}, nil
	})))

	_, err := api.GetBounceDump(9999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetBounceDump_EmptyBody(t *testing.T) {
	want := BounceDumpResp{Body: ""}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetBounceDump(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Body != "" {
		t.Errorf("expected empty Body, got %q", got.Body)
	}
}

// ---- ActivateBounce ------------------------------------------------------------

func TestActivateBounce_Success(t *testing.T) {
	wantBounce := BounceResp{ID: 42, Inactive: false, CanActivate: false}
	want := ActivateBounceResp{
		Message: "OK",
		Bounce:  wantBounce,
	}

	api := New(APITokenOpt("test-server-token"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/bounces/42/activate") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if req.Header.Get("X-Postmark-Server-Token") == "" {
			t.Error("expected X-Postmark-Server-Token header to be set")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ActivateBounce(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "OK" {
		t.Errorf("Message = %q, want OK", got.Message)
	}
	if got.Bounce.ID != 42 {
		t.Errorf("Bounce.ID = %d, want 42", got.Bounce.ID)
	}
	if got.Bounce.Inactive {
		t.Error("Bounce.Inactive should be false after activation")
	}
}

func TestActivateBounce_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "bounce not found"}),
		}, nil
	})))

	_, err := api.ActivateBounce(9999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestActivateBounce_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.ActivateBounce(1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- GetDeliveryStats ----------------------------------------------------------

func TestGetDeliveryStats_Success(t *testing.T) {
	want := DeliveryStatsResp{
		InactiveMails: 5,
		Bounces: []BounceCount{
			{Name: "All", Count: 10, Type: ""},
			{Name: "Hard bounce", Count: 3, Type: "HardBounce"},
			{Name: "Soft bounce", Count: 7, Type: "SoftBounce"},
		},
	}

	api := New(APITokenOpt("test-server-token"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/deliverystats") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if req.Header.Get("X-Postmark-Server-Token") == "" {
			t.Error("expected X-Postmark-Server-Token header to be set")
		}
		if req.Header.Get("X-Postmark-Account-Token") != "" {
			t.Error("X-Postmark-Account-Token header must not be set on delivery stats requests")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetDeliveryStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.InactiveMails != 5 {
		t.Errorf("InactiveMails = %d, want 5", got.InactiveMails)
	}
	if len(got.Bounces) != 3 {
		t.Errorf("len(Bounces) = %d, want 3", len(got.Bounces))
	}
	if got.Bounces[1].Type != "HardBounce" {
		t.Errorf("Bounces[1].Type = %q, want HardBounce", got.Bounces[1].Type)
	}
	if got.Bounces[2].Count != 7 {
		t.Errorf("Bounces[2].Count = %d, want 7", got.Bounces[2].Count)
	}
}

func TestGetDeliveryStats_Empty(t *testing.T) {
	want := DeliveryStatsResp{
		InactiveMails: 0,
		Bounces:       []BounceCount{},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetDeliveryStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.InactiveMails != 0 {
		t.Errorf("InactiveMails = %d, want 0", got.InactiveMails)
	}
}

func TestGetDeliveryStats_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.GetDeliveryStats()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- ServerToken header sanity -------------------------------------------------

// TestServerTokenHeader verifies that all bounce/delivery-stats endpoints
// use X-Postmark-Server-Token and do not leak X-Postmark-Account-Token.
func TestServerTokenHeader_AllEndpoints(t *testing.T) {
	const token = "server-tok-xyz"

	checkHeaders := func(t *testing.T, req *http.Request) {
		t.Helper()
		if got := req.Header.Get("X-Postmark-Server-Token"); got != token {
			t.Errorf("X-Postmark-Server-Token = %q, want %q", got, token)
		}
		if got := req.Header.Get("X-Postmark-Account-Token"); got != "" {
			t.Errorf("X-Postmark-Account-Token must be absent, got %q", got)
		}
	}

	t.Run("ListBounces", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, ListBouncesResp{})}, nil
		})))
		api.ListBounces(ListBouncesParams{}) //nolint:errcheck
	})

	t.Run("GetBounce", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, BounceResp{})}, nil
		})))
		api.GetBounce(1) //nolint:errcheck
	})

	t.Run("GetBounceDump", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, BounceDumpResp{})}, nil
		})))
		api.GetBounceDump(1) //nolint:errcheck
	})

	t.Run("ActivateBounce", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, ActivateBounceResp{})}, nil
		})))
		api.ActivateBounce(1) //nolint:errcheck
	})

	t.Run("GetDeliveryStats", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, DeliveryStatsResp{})}, nil
		})))
		api.GetDeliveryStats() //nolint:errcheck
	})
}
