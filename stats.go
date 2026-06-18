package postmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type (
	// StatsParams encapsulates the optional query parameters accepted by Stats
	// endpoints: tag, fromDate, toDate, and messageStreamID.
	StatsParams struct {
		Tag             string
		FromDate        string // YYYY-MM-DD
		ToDate          string // YYYY-MM-DD
		MessageStreamID string
	}

	// OutboundOverviewResp is the response from GET /stats/outbound.
	OutboundOverviewResp struct {
		Sent             int     `json:"Sent"`
		Bounced          int     `json:"Bounced"`
		SMTPAPIErrors    int     `json:"SMTPApiErrors"`
		BounceRate       float64 `json:"BounceRate"`
		SpamComplaints   int     `json:"SpamComplaints"`
		SpamComplaintsRate float64 `json:"SpamComplaintsRate"`
		Opens            int     `json:"Opens"`
		UniqueOpens      int     `json:"UniqueOpens"`
		Tracked          int     `json:"Tracked"`
		WithLinkTracking int     `json:"WithLinkTracking"`
		WithOpenTracking int     `json:"WithOpenTracking"`
		TotalTrackedLinksSent int `json:"TotalTrackedLinksSent"`
		UniqueLinksClicked    int `json:"UniqueLinksClicked"`
		TotalClicks           int `json:"TotalClicks"`
		WithClientRecording   int `json:"WithClientRecording"`
		WithPlatformRecording int `json:"WithPlatformRecording"`
	}

	// SentCountsResp is the response from GET /stats/outbound/sends.
	SentCountsResp struct {
		Days  []DayCount `json:"Days"`
		Sent  int        `json:"Sent"`
	}

	// BounceCountsResp is the response from GET /stats/outbound/bounces.
	BounceCountsResp struct {
		Days         []DayCount `json:"Days"`
		Bounced      int        `json:"Bounced"`
		SMTPAPIErrors int       `json:"SMTPApiErrors"`
		BounceRate   float64    `json:"BounceRate"`
		Transient    int        `json:"Transient"`
		HardBounce   int        `json:"HardBounce"`
		SoftBounce   int        `json:"SoftBounce"`
		ManuallyDeactivated int `json:"ManuallyDeactivated"`
	}

	// SpamComplaintsResp is the response from GET /stats/outbound/spam.
	SpamComplaintsResp struct {
		Days           []DayCount `json:"Days"`
		SpamComplaints int        `json:"SpamComplaints"`
		SpamComplaintsRate float64 `json:"SpamComplaintsRate"`
	}

	// ClickCountsResp is the response from GET /stats/outbound/clicks.
	ClickCountsResp struct {
		Days                  []DayCount `json:"Days"`
		TotalTrackedLinksSent int        `json:"TotalTrackedLinksSent"`
		UniqueLinksClicked    int        `json:"UniqueLinksClicked"`
		TotalClicks           int        `json:"TotalClicks"`
		WithLinkTracking      int        `json:"WithLinkTracking"`
	}

	// OpenCountsResp is the response from GET /stats/outbound/opens.
	OpenCountsResp struct {
		Days             []DayCount `json:"Days"`
		Opens            int        `json:"Opens"`
		UniqueOpens      int        `json:"UniqueOpens"`
		Tracked          int        `json:"Tracked"`
		WithOpenTracking int        `json:"WithOpenTracking"`
		WithClientRecording   int  `json:"WithClientRecording"`
		WithPlatformRecording int  `json:"WithPlatformRecording"`
	}

	// DayCount is a generic date+count entry used in stat time-series responses.
	DayCount struct {
		Date  string `json:"Date"`
		Sent  int    `json:"Sent,omitempty"`
		Count int    `json:"Count,omitempty"`
	}
)

// newServerRequest builds an *http.Request using the server API token
// (X-Postmark-Server-Token) instead of the account token. It is used for
// server-scoped endpoints such as the Stats API.
func (a *API) newServerRequest(method, path string, body interface{}) (*http.Request, error) {
	var reqBody io.Reader = http.NoBody
	hasBody := body != nil
	if hasBody {
		reqPayload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(reqPayload)
	}

	req, err := http.NewRequest(
		method,
		fmt.Sprintf("%s/%s", a.baseHost, path),
		reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}

	serverToken := a.serverToken
	if serverToken == "" {
		serverToken = os.Getenv("POSTMARK_SERVER_TOKEN")
	}
	req.Header.Set("X-Postmark-Server-Token", serverToken)

	return req, nil
}

// statsQuery builds the url-encoded query string from a *StatsParams.
// It returns an empty string when params is nil.
func statsQuery(params *StatsParams) string {
	if params == nil {
		return ""
	}
	v := url.Values{}
	if params.Tag != "" {
		v.Set("tag", params.Tag)
	}
	if params.FromDate != "" {
		v.Set("fromdate", params.FromDate)
	}
	if params.ToDate != "" {
		v.Set("todate", params.ToDate)
	}
	if params.MessageStreamID != "" {
		v.Set("messagestream", params.MessageStreamID)
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

// GetOutboundOverview returns an overview of outbound email statistics.
func (a *API) GetOutboundOverview(params *StatsParams) (*OutboundOverviewResp, error) {
	req, err := a.newServerRequest(http.MethodGet, "stats/outbound"+statsQuery(params), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data OutboundOverviewResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetSentCounts returns sent email counts over time.
func (a *API) GetSentCounts(params *StatsParams) (*SentCountsResp, error) {
	req, err := a.newServerRequest(http.MethodGet, "stats/outbound/sends"+statsQuery(params), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data SentCountsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetBounceCounts returns bounce counts over time.
func (a *API) GetBounceCounts(params *StatsParams) (*BounceCountsResp, error) {
	req, err := a.newServerRequest(http.MethodGet, "stats/outbound/bounces"+statsQuery(params), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data BounceCountsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetSpamComplaints returns spam complaint counts over time.
func (a *API) GetSpamComplaints(params *StatsParams) (*SpamComplaintsResp, error) {
	req, err := a.newServerRequest(http.MethodGet, "stats/outbound/spam"+statsQuery(params), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data SpamComplaintsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetClickCounts returns link click counts over time.
func (a *API) GetClickCounts(params *StatsParams) (*ClickCountsResp, error) {
	req, err := a.newServerRequest(http.MethodGet, "stats/outbound/clicks"+statsQuery(params), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ClickCountsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetEmailOpenCounts returns email open counts over time.
func (a *API) GetEmailOpenCounts(params *StatsParams) (*OpenCountsResp, error) {
	req, err := a.newServerRequest(http.MethodGet, "stats/outbound/opens"+statsQuery(params), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data OpenCountsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
