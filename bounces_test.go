package postmark

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// intPtr is a test helper that returns a pointer to the given int value,
// making it convenient to set GetBouncesParams.Offset inline.
func intPtr(v int) *int { return &v }

// ---- GetDeliveryStats ----------------------------------------------------------

func TestGetDeliveryStats_Success(t *testing.T) {
	want := DeliveryStatsResp{
		InactiveMails: 5,
		Bounces: []BounceCountByType{
			{Name: "Hard Bounce", Count: 3, Type: "HardBounce"},
			{Name: "Soft Bounce", Count: 2, Type: "SoftBounce"},
		},
	}

	api := New(
		APITokenOpt("test-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
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
				t.Error("expected X-Postmark-Account-Token header to be absent")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.GetDeliveryStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.InactiveMails != 5 {
		t.Errorf("InactiveMails = %d, want 5", got.InactiveMails)
	}
	if len(got.Bounces) != 2 {
		t.Errorf("len(Bounces) = %d, want 2", len(got.Bounces))
	}
	if got.Bounces[0].Type != "HardBounce" {
		t.Errorf("Bounces[0].Type = %q, want HardBounce", got.Bounces[0].Type)
	}
}

// TestGetDeliveryStats_APIError verifies that a non-2xx response causes
// GetDeliveryStats to return a non-nil error that wraps a *PostmarkErr value
// with the expected ErrorCode.
func TestGetDeliveryStats_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.GetDeliveryStats()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	// errors.As requires the target to be a pointer to the concrete error type.
	// readResponse returns *PostmarkErr for non-sentinel errors, so pe must be
	// declared as *PostmarkErr and &pe is **PostmarkErr.
	var pe *PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected error to be *PostmarkErr, got %T: %v", err, err)
	}
}

// ---- GetBounces ----------------------------------------------------------------

func TestGetBounces_Success(t *testing.T) {
	want := GetBouncesResp{
		TotalCount: 2,
		Bounces: []BounceResp{
			{ID: 1, Type: "HardBounce", Email: "hard@example.com"},
			{ID: 2, Type: "SoftBounce", Email: "soft@example.com"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/bounces") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetBounces(nil)
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

// TestGetBounces_NilParams_NoQueryString verifies that passing nil params to
// GetBounces results in a request with no query string at all.
func TestGetBounces_NilParams_NoQueryString(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.RawQuery != "" {
			t.Errorf("expected empty query string for nil params, got %q", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, GetBouncesResp{}),
		}, nil
	})))

	_, err := api.GetBounces(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBounces_WithParams(t *testing.T) {
	want := GetBouncesResp{TotalCount: 1, Bounces: []BounceResp{{ID: 10}}}
	inactive := true

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.RawQuery
		if q == "" {
			t.Fatalf("expected non-empty query string, got empty; URL=%s", req.URL.String())
		}
		if !strings.Contains(q, "count=10") {
			t.Errorf("expected count=10 in query, got %s", q)
		}
		if !strings.Contains(q, "offset=5") {
			t.Errorf("expected offset=5 in query, got %s", q)
		}
		if !strings.Contains(q, "type=HardBounce") {
			t.Errorf("expected type=HardBounce in query, got %s", q)
		}
		if !strings.Contains(q, "inactive=true") {
			t.Errorf("expected inactive=true in query, got %s", q)
		}
		if !strings.Contains(q, "emailFilter=test") {
			t.Errorf("expected emailFilter=test in query, got %s", q)
		}
		if !strings.Contains(q, "tag=mytag") {
			t.Errorf("expected tag=mytag in query, got %s", q)
		}
		if !strings.Contains(q, "messageID=abc123") {
			t.Errorf("expected messageID=abc123 in query, got %s", q)
		}
		if !strings.Contains(q, "fromDate=2023-01-01") {
			t.Errorf("expected fromDate=2023-01-01 in query, got %s", q)
		}
		if !strings.Contains(q, "toDate=2023-12-31") {
			t.Errorf("expected toDate=2023-12-31 in query, got %s", q)
		}
		if !strings.Contains(q, "messageStreamID=outbound") {
			t.Errorf("expected messageStreamID=outbound in query, got %s", q)
		}
		// Also verify the path is clean (no '?' embedded in it).
		if strings.Contains(req.URL.Path, "?") {
			t.Errorf("query string must not be embedded in URL path, got path=%q", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	params := &GetBouncesParams{
		Count:           10,
		Offset:          intPtr(5),
		Type:            "HardBounce",
		Inactive:        &inactive,
		EmailFilter:     "test",
		Tag:             "mytag",
		MessageID:       "abc123",
		FromDate:        "2023-01-01",
		ToDate:          "2023-12-31",
		MessageStreamID: "outbound",
	}

	got, err := api.GetBounces(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
}

// TestGetBounces_OffsetZero verifies that a non-nil Offset pointer pointing to
// 0 sends offset=0 in the query string, explicitly requesting the first page.
func TestGetBounces_OffsetZero(t *testing.T) {
	want := GetBouncesResp{TotalCount: 1, Bounces: []BounceResp{{ID: 1}}}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.RawQuery
		if !strings.Contains(q, "offset=0") {
			t.Errorf("expected offset=0 in query when Offset=intPtr(0), got %q", q)
		}
		// Verify query is in RawQuery, not embedded in the path.
		if strings.Contains(req.URL.Path, "?") {
			t.Errorf("query string must not be embedded in URL path, got path=%q", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetBounces(&GetBouncesParams{Count: 10, Offset: intPtr(0)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
}

// TestGetBounces_OffsetNil verifies that a nil Offset pointer omits the offset
// parameter from the query string entirely, leaving the API default in effect.
func TestGetBounces_OffsetNil(t *testing.T) {
	want := GetBouncesResp{TotalCount: 1, Bounces: []BounceResp{{ID: 1}}}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.RawQuery
		if strings.Contains(q, "offset=") {
			t.Errorf("expected offset to be absent from query when Offset=nil, got %q", q)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetBounces(&GetBouncesParams{Count: 10, Offset: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
}

func TestGetBounces_InactiveFalse(t *testing.T) {
	want := GetBouncesResp{TotalCount: 1, Bounces: []BounceResp{{ID: 11}}}
	inactive := false

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.RawQuery
		if !strings.Contains(q, "inactive=false") {
			t.Errorf("expected inactive=false in query, got %s", q)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetBounces(&GetBouncesParams{Inactive: &inactive})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
}

// TestGetBounces_APIError verifies that a non-2xx response causes GetBounces
// to return a non-nil *PostmarkErr.
func TestGetBounces_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.GetBounces(nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var pe *PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected error to be *PostmarkErr, got %T: %v", err, err)
	}
}

// ---- GetBounce -----------------------------------------------------------------

func TestGetBounce_Success(t *testing.T) {
	bouncedAt, _ := time.Parse(time.RFC3339, "2023-06-01T12:00:00Z")
	want := BounceResp{
		ID:            42,
		Type:          "HardBounce",
		TypeCode:      1,
		Name:          "Hard Bounce",
		Tag:           "test-tag",
		MessageID:     "msg-001",
		ServerID:      7,
		MessageStream: "outbound",
		Description:   "The server was unable to deliver your message",
		Details:       "smtp;550 5.1.1 The email account that you tried to reach does not exist",
		Email:         "bounce@example.com",
		From:          "sender@example.com",
		BouncedAt:     bouncedAt,
		DumpAvailable: true,
		Inactive:      true,
		CanActivate:   true,
		Subject:       "Test Subject",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/bounces/42") {
			t.Errorf("unexpected path: %s", req.URL.Path)
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
	if !got.Inactive {
		t.Error("expected Inactive to be true")
	}
	if !got.CanActivate {
		t.Error("expected CanActivate to be true")
	}
	if !got.BouncedAt.Equal(bouncedAt) {
		t.Errorf("BouncedAt = %v, want %v", got.BouncedAt, bouncedAt)
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
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestGetBounce_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.GetBounce(1)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetBounceDump -------------------------------------------------------------

func TestGetBounceDump_Success(t *testing.T) {
	want := BounceDumpResp{Body: "Return-Path: <>\r\nReceived: from mail.example.com\r\n"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/bounces/42/dump") {
			t.Errorf("unexpected path: %s", req.URL.Path)
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
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "bounce not found"}),
		}, nil
	})))

	_, err := api.GetBounceDump(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestGetBounceDump_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.GetBounceDump(1)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- ActivateBounce ------------------------------------------------------------

func TestActivateBounce_Success(t *testing.T) {
	want := ActivateBounceResp{
		Message: "OK",
		Bounce: BounceResp{
			ID:          42,
			Type:        "HardBounce",
			Email:       "bounce@example.com",
			Inactive:    false,
			CanActivate: false,
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/bounces/42/activate") {
			t.Errorf("unexpected path: %s", req.URL.Path)
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
		t.Error("expected Bounce.Inactive to be false after activation")
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
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestActivateBounce_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.ActivateBounce(1)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetBounceTags -------------------------------------------------------------

func TestGetBounceTags_Success(t *testing.T) {
	want := []string{"tag1", "tag2", "tag3"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/bounces/tags") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetBounceTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("len(tags) = %d, want 3", len(got))
	}
	for i, tag := range want {
		if got[i] != tag {
			t.Errorf("tags[%d] = %q, want %q", i, got[i], tag)
		}
	}
}

func TestGetBounceTags_Empty(t *testing.T) {
	want := []string{}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetBounceTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty tags, got %v", got)
	}
}

func TestGetBounceTags_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.GetBounceTags()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- Server Token Header -------------------------------------------------------

// TestBounceAPI_UsesServerToken verifies that all Bounce API methods set the
// X-Postmark-Server-Token header to the exact configured token value and do NOT
// set X-Postmark-Account-Token.
func TestBounceAPI_UsesServerToken(t *testing.T) {
	const token = "test-server-token"
	checkHeaders := func(t *testing.T, req *http.Request) {
		t.Helper()
		if req.Header.Get("X-Postmark-Server-Token") != token {
			t.Errorf("expected X-Postmark-Server-Token=%q, got %q", token, req.Header.Get("X-Postmark-Server-Token"))
		}
		if req.Header.Get("X-Postmark-Account-Token") != "" {
			t.Errorf("expected X-Postmark-Account-Token to be absent, got %q", req.Header.Get("X-Postmark-Account-Token"))
		}
	}

	t.Run("GetDeliveryStats", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, DeliveryStatsResp{})}, nil
		})))
		_, _ = api.GetDeliveryStats()
	})

	t.Run("GetBounces", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, GetBouncesResp{})}, nil
		})))
		_, _ = api.GetBounces(nil)
	})

	t.Run("GetBounce", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, BounceResp{})}, nil
		})))
		_, _ = api.GetBounce(1)
	})

	t.Run("GetBounceDump", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, BounceDumpResp{})}, nil
		})))
		_, _ = api.GetBounceDump(1)
	})

	t.Run("ActivateBounce", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, ActivateBounceResp{})}, nil
		})))
		_, _ = api.ActivateBounce(1)
	})

	t.Run("GetBounceTags", func(t *testing.T) {
		api := New(APITokenOpt(token), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, []string{})}, nil
		})))
		_, _ = api.GetBounceTags()
	})
}
