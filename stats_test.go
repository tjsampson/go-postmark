package postmark

import (
	"net/http"
	"strings"
	"testing"
)

// ---- GetOutboundOverview ------------------------------------------------------

func TestGetOutboundOverview_Success(t *testing.T) {
	want := OutboundOverviewResp{
		Sent:    100,
		Bounced: 5,
		Opens:   80,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/stats/outbound") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if req.Header.Get("X-Postmark-Server-Token") != "srv-tok" {
			t.Errorf("expected X-Postmark-Server-Token header, got %q", req.Header.Get("X-Postmark-Server-Token"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})), ServerTokenOpt("srv-tok"))

	got, err := api.GetOutboundOverview(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Sent != 100 {
		t.Errorf("Sent = %d, want 100", got.Sent)
	}
	if got.Bounced != 5 {
		t.Errorf("Bounced = %d, want 5", got.Bounced)
	}
}

func TestGetOutboundOverview_WithParams(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.RawQuery
		if !strings.Contains(q, "tag=welcome") {
			t.Errorf("expected tag param, query=%s", q)
		}
		if !strings.Contains(q, "fromdate=2024-01-01") {
			t.Errorf("expected fromdate param, query=%s", q)
		}
		if !strings.Contains(q, "todate=2024-01-31") {
			t.Errorf("expected todate param, query=%s", q)
		}
		if !strings.Contains(q, "messagestream=outbound") {
			t.Errorf("expected messagestream param, query=%s", q)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, OutboundOverviewResp{}),
		}, nil
	})))

	_, err := api.GetOutboundOverview(&StatsParams{
		Tag:             "welcome",
		FromDate:        "2024-01-01",
		ToDate:          "2024-01-31",
		MessageStreamID: "outbound",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetOutboundOverview_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.GetOutboundOverview(nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetSentCounts ------------------------------------------------------------

func TestGetSentCounts_Success(t *testing.T) {
	want := SentCountsResp{
		Sent: 250,
		Days: []DayCount{
			{Date: "2024-01-01", Sent: 100},
			{Date: "2024-01-02", Sent: 150},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/stats/outbound/sends") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetSentCounts(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Sent != 250 {
		t.Errorf("Sent = %d, want 250", got.Sent)
	}
	if len(got.Days) != 2 {
		t.Errorf("len(Days) = %d, want 2", len(got.Days))
	}
}

func TestGetSentCounts_WithTag(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.RawQuery, "tag=newsletter") {
			t.Errorf("expected tag=newsletter, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, SentCountsResp{}),
		}, nil
	})))

	_, err := api.GetSentCounts(&StatsParams{Tag: "newsletter"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- GetBounceCounts ----------------------------------------------------------

func TestGetBounceCounts_Success(t *testing.T) {
	want := BounceCountsResp{
		Bounced:    10,
		HardBounce: 3,
		SoftBounce: 7,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/stats/outbound/bounces") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetBounceCounts(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Bounced != 10 {
		t.Errorf("Bounced = %d, want 10", got.Bounced)
	}
	if got.HardBounce != 3 {
		t.Errorf("HardBounce = %d, want 3", got.HardBounce)
	}
}

func TestGetBounceCounts_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.GetBounceCounts(nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetSpamComplaints --------------------------------------------------------

func TestGetSpamComplaints_Success(t *testing.T) {
	want := SpamComplaintsResp{
		SpamComplaints:     3,
		SpamComplaintsRate: 0.03,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/stats/outbound/spam") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetSpamComplaints(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SpamComplaints != 3 {
		t.Errorf("SpamComplaints = %d, want 3", got.SpamComplaints)
	}
}

// ---- GetClickCounts -----------------------------------------------------------

func TestGetClickCounts_Success(t *testing.T) {
	want := ClickCountsResp{
		TotalClicks:        150,
		UniqueLinksClicked: 80,
		WithLinkTracking:   200,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/stats/outbound/clicks") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetClickCounts(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalClicks != 150 {
		t.Errorf("TotalClicks = %d, want 150", got.TotalClicks)
	}
	if got.UniqueLinksClicked != 80 {
		t.Errorf("UniqueLinksClicked = %d, want 80", got.UniqueLinksClicked)
	}
}

func TestGetClickCounts_WithDateRange(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		q := req.URL.RawQuery
		if !strings.Contains(q, "fromdate=2024-06-01") {
			t.Errorf("expected fromdate param, query=%s", q)
		}
		if !strings.Contains(q, "todate=2024-06-30") {
			t.Errorf("expected todate param, query=%s", q)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, ClickCountsResp{}),
		}, nil
	})))

	_, err := api.GetClickCounts(&StatsParams{
		FromDate: "2024-06-01",
		ToDate:   "2024-06-30",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- GetEmailOpenCounts -------------------------------------------------------

func TestGetEmailOpenCounts_Success(t *testing.T) {
	want := OpenCountsResp{
		Opens:       500,
		UniqueOpens: 350,
		Tracked:     600,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/stats/outbound/opens") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetEmailOpenCounts(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Opens != 500 {
		t.Errorf("Opens = %d, want 500", got.Opens)
	}
	if got.UniqueOpens != 350 {
		t.Errorf("UniqueOpens = %d, want 350", got.UniqueOpens)
	}
}

func TestGetEmailOpenCounts_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.GetEmailOpenCounts(nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- StatsParams / statsQuery -------------------------------------------------

func TestStatsQuery_Nil(t *testing.T) {
	got := statsQuery(nil)
	if got != "" {
		t.Errorf("statsQuery(nil) = %q, want empty string", got)
	}
}

func TestStatsQuery_Empty(t *testing.T) {
	got := statsQuery(&StatsParams{})
	if got != "" {
		t.Errorf("statsQuery(&StatsParams{}) = %q, want empty string", got)
	}
}

func TestStatsQuery_AllParams(t *testing.T) {
	got := statsQuery(&StatsParams{
		Tag:             "test",
		FromDate:        "2024-01-01",
		ToDate:          "2024-12-31",
		MessageStreamID: "stream1",
	})
	if !strings.HasPrefix(got, "?") {
		t.Errorf("expected query to start with '?', got %q", got)
	}
	for _, param := range []string{"tag=test", "fromdate=2024-01-01", "todate=2024-12-31", "messagestream=stream1"} {
		if !strings.Contains(got, param) {
			t.Errorf("expected %q in query %q", param, got)
		}
	}
}

// ---- ServerTokenOpt -----------------------------------------------------------

func TestServerTokenOpt(t *testing.T) {
	api := New(ServerTokenOpt("my-server-token"))
	if api.serverToken != "my-server-token" {
		t.Errorf("serverToken = %q, want my-server-token", api.serverToken)
	}
}

func TestNewServerRequest_UsesServerToken(t *testing.T) {
	api := New(ServerTokenOpt("srv-123"))
	req, err := api.newServerRequest(http.MethodGet, "stats/outbound", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("X-Postmark-Server-Token"); got != "srv-123" {
		t.Errorf("X-Postmark-Server-Token = %q, want srv-123", got)
	}
	// Account token header should not be set on server requests.
	if got := req.Header.Get("X-Postmark-Account-Token"); got != "" {
		t.Errorf("unexpected X-Postmark-Account-Token header = %q", got)
	}
}
