package postmark

import (
	"net/http"
	"strings"
	"testing"
)

// statsEndpointTest is a table-driven helper for verifying stats endpoints.
type statsEndpointTest struct {
	name         string
	expectedPath string
	params       StatsParams
}

func TestGetOutboundStats_Success(t *testing.T) {
	want := OutboundStatsResp{
		Sent:    100,
		Bounced: 5,
		Opens:   42,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundStats(StatsParams{})
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

func TestGetOutboundStats_WithParams(t *testing.T) {
	tests := []statsEndpointTest{
		{
			name:         "with tag",
			expectedPath: "stats/outbound",
			params:       StatsParams{Tag: "newsletter"},
		},
		{
			name:         "with date range",
			expectedPath: "stats/outbound",
			params:       StatsParams{FromDate: "2024-01-01", ToDate: "2024-01-31"},
		},
		{
			name:         "with message stream",
			expectedPath: "stats/outbound",
			params:       StatsParams{MessageStream: "outbound"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				if !strings.HasSuffix(req.URL.Path, tc.expectedPath) {
					t.Errorf("unexpected path: %s, want suffix %s", req.URL.Path, tc.expectedPath)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, OutboundStatsResp{}),
				}, nil
			})))
			_, err := api.GetOutboundStats(tc.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetOutboundStats_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.GetOutboundStats(StatsParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetOutboundSendCounts_Success(t *testing.T) {
	want := SendCountsResp{
		Sent: 50,
		Days: []DayStat{
			{Date: "2024-01-01", Sent: 25},
			{Date: "2024-01-02", Sent: 25},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/sends") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundSendCounts(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Sent != 50 {
		t.Errorf("Sent = %d, want 50", got.Sent)
	}
	if len(got.Days) != 2 {
		t.Errorf("len(Days) = %d, want 2", len(got.Days))
	}
}

func TestGetOutboundBounceCounts_Success(t *testing.T) {
	want := BounceCountsResp{
		HardBounce: 3,
		SoftBounce: 7,
		Days: []BounceDay{
			{Date: "2024-01-01", HardBounce: 2, SoftBounce: 4},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/bounces") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundBounceCounts(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.HardBounce != 3 {
		t.Errorf("HardBounce = %d, want 3", got.HardBounce)
	}
	if got.SoftBounce != 7 {
		t.Errorf("SoftBounce = %d, want 7", got.SoftBounce)
	}
}

func TestGetOutboundSpamCounts_Success(t *testing.T) {
	want := SpamCountsResp{
		SpamComplaint: 2,
		Days:          []SpamDay{{Date: "2024-01-01", SpamComplaint: 2}},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/spam") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundSpamCounts(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SpamComplaint != 2 {
		t.Errorf("SpamComplaint = %d, want 2", got.SpamComplaint)
	}
}

func TestGetOutboundTrackedEmailCounts_Success(t *testing.T) {
	want := TrackedEmailCountsResp{
		Tracked: 80,
		Days:    []TrackedDay{{Date: "2024-01-01", Tracked: 80}},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/tracked") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundTrackedEmailCounts(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Tracked != 80 {
		t.Errorf("Tracked = %d, want 80", got.Tracked)
	}
}

func TestGetOutboundOpenCounts_Success(t *testing.T) {
	want := OpenCountsResp{
		Opens:       60,
		UniqueOpens: 45,
		Days:        []OpenDay{{Date: "2024-01-01", Opens: 60, UniqueOpens: 45}},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/opens") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundOpenCounts(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Opens != 60 {
		t.Errorf("Opens = %d, want 60", got.Opens)
	}
	if got.UniqueOpens != 45 {
		t.Errorf("UniqueOpens = %d, want 45", got.UniqueOpens)
	}
}

func TestGetOutboundOpenPlatforms_Success(t *testing.T) {
	want := OpenPlatformsResp{
		Desktop: 30,
		Mobile:  20,
		WebMail: 10,
		Unknown: 5,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/opens/platforms") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundOpenPlatforms(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Desktop != 30 {
		t.Errorf("Desktop = %d, want 30", got.Desktop)
	}
	if got.Mobile != 20 {
		t.Errorf("Mobile = %d, want 20", got.Mobile)
	}
}

func TestGetOutboundOpenEmailClients_Success(t *testing.T) {
	want := OpenEmailClientsResp{
		EmailClients: []EmailClientUsage{
			{Name: "Apple Mail", PercentageOfOpens: 40.5},
			{Name: "Gmail", PercentageOfOpens: 35.0},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/opens/emailclients") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundOpenEmailClients(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.EmailClients) != 2 {
		t.Errorf("len(EmailClients) = %d, want 2", len(got.EmailClients))
	}
}

func TestGetOutboundClickCounts_Success(t *testing.T) {
	want := ClickCountsResp{
		Clicks:       25,
		UniqueClicks: 18,
		Days:         []ClickDay{{Date: "2024-01-01", Clicks: 25, UniqueClicks: 18}},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/clicks") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundClickCounts(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Clicks != 25 {
		t.Errorf("Clicks = %d, want 25", got.Clicks)
	}
	if got.UniqueClicks != 18 {
		t.Errorf("UniqueClicks = %d, want 18", got.UniqueClicks)
	}
}

func TestGetOutboundClickBrowserFamilies_Success(t *testing.T) {
	want := ClickBrowserFamiliesResp{
		BrowserFamilies: []BrowserFamilyUsage{
			{Name: "Chrome", PercentageOfClicks: 55.0},
			{Name: "Firefox", PercentageOfClicks: 25.0},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/clicks/browserfamilies") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundClickBrowserFamilies(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.BrowserFamilies) != 2 {
		t.Errorf("len(BrowserFamilies) = %d, want 2", len(got.BrowserFamilies))
	}
}

func TestGetOutboundClickPlatforms_Success(t *testing.T) {
	want := ClickPlatformsResp{
		Platforms: []ClickPlatformUsage{
			{Name: "Desktop", PercentageOfClicks: 70.0},
			{Name: "Mobile", PercentageOfClicks: 30.0},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/clicks/platforms") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundClickPlatforms(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Platforms) != 2 {
		t.Errorf("len(Platforms) = %d, want 2", len(got.Platforms))
	}
}

func TestGetOutboundClickLocations_Success(t *testing.T) {
	want := ClickLocationsResp{
		Locations: []ClickLocationUsage{
			{Name: "US", PercentageOfClicks: 60.0},
			{Name: "UK", PercentageOfClicks: 15.0},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "stats/outbound/clicks/location") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetOutboundClickLocations(StatsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Locations) != 2 {
		t.Errorf("len(Locations) = %d, want 2", len(got.Locations))
	}
}

func TestStatsParams_QueryEncoding(t *testing.T) {
	tests := []struct {
		name   string
		params StatsParams
		check  func(t *testing.T, query string)
	}{
		{
			name:   "tag parameter",
			params: StatsParams{Tag: "welcome"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "tag=welcome") {
					t.Errorf("expected tag param, query=%s", query)
				}
			},
		},
		{
			name:   "fromdate parameter",
			params: StatsParams{FromDate: "2024-01-01"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "fromdate=2024-01-01") {
					t.Errorf("expected fromdate param, query=%s", query)
				}
			},
		},
		{
			name:   "todate parameter",
			params: StatsParams{ToDate: "2024-01-31"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "todate=2024-01-31") {
					t.Errorf("expected todate param, query=%s", query)
				}
			},
		},
		{
			name:   "messagestream parameter",
			params: StatsParams{MessageStream: "broadcast"},
			check: func(t *testing.T, query string) {
				if !strings.Contains(query, "messagestream=broadcast") {
					t.Errorf("expected messagestream param, query=%s", query)
				}
			},
		},
		{
			name:   "empty params produce no query string",
			params: StatsParams{},
			check: func(t *testing.T, query string) {
				if query != "" {
					t.Errorf("expected empty query string, got %s", query)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				tc.check(t, req.URL.RawQuery)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, OutboundStatsResp{}),
				}, nil
			})))
			_, err := api.GetOutboundStats(tc.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
