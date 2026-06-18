package postmark

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type (
	// StatsParams holds common query parameters for stats endpoints.
	StatsParams struct {
		Tag           string
		FromDate      string
		ToDate        string
		MessageStream string
	}

	// OutboundStatsResp represents the response from GET /stats/outbound.
	OutboundStatsResp struct {
		Sent                 int     `json:"Sent"`
		Bounced              int     `json:"Bounced"`
		SMTPApiErrors        int     `json:"SMTPApiErrors"`
		BounceRate           float64 `json:"BounceRate"`
		SpamComplaints       int     `json:"SpamComplaints"`
		SpamComplaintsRate   float64 `json:"SpamComplaintsRate"`
		Opens                int     `json:"Opens"`
		UniqueOpens          int     `json:"UniqueOpens"`
		Tracked              int     `json:"Tracked"`
		WithClientRecorded   int     `json:"WithClientRecorded"`
		WithPlatformRecorded int     `json:"WithPlatformRecorded"`
		WithReadTimeRecorded int     `json:"WithReadTimeRecorded"`
	}

	// SendCountsResp represents the response from GET /stats/outbound/sends.
	SendCountsResp struct {
		Days []DayStat `json:"Days"`
		Sent int       `json:"Sent"`
	}

	// DayStat represents stats for a single day.
	DayStat struct {
		Date string `json:"Date"`
		Sent int    `json:"Sent"`
	}

	// BounceCountsResp represents the response from GET /stats/outbound/bounces.
	BounceCountsResp struct {
		Days         []BounceDay `json:"Days"`
		HardBounce   int         `json:"HardBounce"`
		SoftBounce   int         `json:"SoftBounce"`
		SMTPApiError int         `json:"SMTPApiError"`
		Transient    int         `json:"Transient"`
	}

	// BounceDay represents bounce stats for a single day.
	BounceDay struct {
		Date         string `json:"Date"`
		HardBounce   int    `json:"HardBounce"`
		SoftBounce   int    `json:"SoftBounce"`
		SMTPApiError int    `json:"SMTPApiError"`
		Transient    int    `json:"Transient"`
	}

	// SpamCountsResp represents the response from GET /stats/outbound/spam.
	SpamCountsResp struct {
		Days          []SpamDay `json:"Days"`
		SpamComplaint int       `json:"SpamComplaint"`
	}

	// SpamDay represents spam complaint stats for a single day.
	SpamDay struct {
		Date          string `json:"Date"`
		SpamComplaint int    `json:"SpamComplaint"`
	}

	// TrackedEmailCountsResp represents the response from GET /stats/outbound/tracked.
	TrackedEmailCountsResp struct {
		Days    []TrackedDay `json:"Days"`
		Tracked int          `json:"Tracked"`
	}

	// TrackedDay represents tracked email stats for a single day.
	TrackedDay struct {
		Date    string `json:"Date"`
		Tracked int    `json:"Tracked"`
	}

	// OpenCountsResp represents the response from GET /stats/outbound/opens.
	OpenCountsResp struct {
		Days        []OpenDay `json:"Days"`
		Opens       int       `json:"Opens"`
		UniqueOpens int       `json:"UniqueOpens"`
	}

	// OpenDay represents open stats for a single day.
	OpenDay struct {
		Date        string `json:"Date"`
		Opens       int    `json:"Opens"`
		UniqueOpens int    `json:"UniqueOpens"`
	}

	// OpenPlatformsResp represents the response from GET /stats/outbound/opens/platforms.
	OpenPlatformsResp struct {
		Days    []PlatformDay `json:"Days"`
		Desktop int           `json:"Desktop"`
		Mobile  int           `json:"Mobile"`
		Unknown int           `json:"Unknown"`
		WebMail int           `json:"WebMail"`
	}

	// PlatformDay represents platform open stats for a single day.
	PlatformDay struct {
		Date    string `json:"Date"`
		Desktop int    `json:"Desktop"`
		Mobile  int    `json:"Mobile"`
		Unknown int    `json:"Unknown"`
		WebMail int    `json:"WebMail"`
	}

	// EmailClientUsage represents usage stats for a single email client.
	EmailClientUsage struct {
		Name              string  `json:"Name"`
		CompanyName       string  `json:"CompanyName"`
		Family            string  `json:"Family"`
		Version           string  `json:"Version"`
		PercentageOfOpens float64 `json:"PercentageOfOpens"`
	}

	// OpenEmailClientsResp represents the response from GET /stats/outbound/opens/emailclients.
	OpenEmailClientsResp struct {
		Days         []EmailClientDay   `json:"Days"`
		EmailClients []EmailClientUsage `json:"EmailClients"`
	}

	// EmailClientDay represents email client open stats for a single day.
	EmailClientDay struct {
		Date         string             `json:"Date"`
		EmailClients []EmailClientUsage `json:"EmailClients"`
	}

	// ClickCountsResp represents the response from GET /stats/outbound/clicks.
	ClickCountsResp struct {
		Days         []ClickDay `json:"Days"`
		Clicks       int        `json:"Clicks"`
		UniqueClicks int        `json:"UniqueClicks"`
	}

	// ClickDay represents click stats for a single day.
	ClickDay struct {
		Date         string `json:"Date"`
		Clicks       int    `json:"Clicks"`
		UniqueClicks int    `json:"UniqueClicks"`
	}

	// BrowserFamilyUsage represents usage stats for a browser family.
	BrowserFamilyUsage struct {
		Name               string  `json:"Name"`
		PercentageOfClicks float64 `json:"PercentageOfClicks"`
	}

	// ClickBrowserFamiliesResp represents the response from GET /stats/outbound/clicks/browserfamilies.
	ClickBrowserFamiliesResp struct {
		Days            []BrowserFamilyDay   `json:"Days"`
		BrowserFamilies []BrowserFamilyUsage `json:"BrowserFamilies"`
	}

	// BrowserFamilyDay represents browser family click stats for a single day.
	BrowserFamilyDay struct {
		Date            string               `json:"Date"`
		BrowserFamilies []BrowserFamilyUsage `json:"BrowserFamilies"`
	}

	// ClickPlatformUsage represents usage stats for a click platform.
	ClickPlatformUsage struct {
		Name               string  `json:"Name"`
		PercentageOfClicks float64 `json:"PercentageOfClicks"`
	}

	// ClickPlatformsResp represents the response from GET /stats/outbound/clicks/platforms.
	ClickPlatformsResp struct {
		Days      []ClickPlatformDay   `json:"Days"`
		Platforms []ClickPlatformUsage `json:"Platforms"`
	}

	// ClickPlatformDay represents platform click stats for a single day.
	ClickPlatformDay struct {
		Date      string               `json:"Date"`
		Platforms []ClickPlatformUsage `json:"Platforms"`
	}

	// ClickLocationUsage represents usage stats for a click location.
	ClickLocationUsage struct {
		Name               string  `json:"Name"`
		PercentageOfClicks float64 `json:"PercentageOfClicks"`
	}

	// ClickLocationsResp represents the response from GET /stats/outbound/clicks/location.
	ClickLocationsResp struct {
		Days      []ClickLocationDay   `json:"Days"`
		Locations []ClickLocationUsage `json:"Location"`
	}

	// ClickLocationDay represents location click stats for a single day.
	ClickLocationDay struct {
		Date      string               `json:"Date"`
		Locations []ClickLocationUsage `json:"Location"`
	}
)

// statsQuery builds a url.Values from StatsParams.
func statsQuery(params StatsParams) url.Values {
	q := url.Values{}
	if params.Tag != "" {
		q.Set("tag", params.Tag)
	}
	if params.FromDate != "" {
		q.Set("fromdate", params.FromDate)
	}
	if params.ToDate != "" {
		q.Set("todate", params.ToDate)
	}
	if params.MessageStream != "" {
		q.Set("messagestream", params.MessageStream)
	}
	return q
}

// GetOutboundStats returns a summary of outbound email statistics.
// GET /stats/outbound
func (a *API) GetOutboundStats(params StatsParams) (*OutboundStatsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data OutboundStatsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundSendCounts returns send counts over time.
// GET /stats/outbound/sends
func (a *API) GetOutboundSendCounts(params StatsParams) (*SendCountsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/sends"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data SendCountsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundBounceCounts returns bounce counts over time.
// GET /stats/outbound/bounces
func (a *API) GetOutboundBounceCounts(params StatsParams) (*BounceCountsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/bounces"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
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

// GetOutboundSpamCounts returns spam complaint counts over time.
// GET /stats/outbound/spam
func (a *API) GetOutboundSpamCounts(params StatsParams) (*SpamCountsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/spam"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data SpamCountsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundTrackedEmailCounts returns tracked email counts over time.
// GET /stats/outbound/tracked
func (a *API) GetOutboundTrackedEmailCounts(params StatsParams) (*TrackedEmailCountsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/tracked"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data TrackedEmailCountsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundOpenCounts returns open counts over time.
// GET /stats/outbound/opens
func (a *API) GetOutboundOpenCounts(params StatsParams) (*OpenCountsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/opens"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
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

// GetOutboundOpenPlatforms returns open counts broken down by platform.
// GET /stats/outbound/opens/platforms
func (a *API) GetOutboundOpenPlatforms(params StatsParams) (*OpenPlatformsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/opens/platforms"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data OpenPlatformsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundOpenEmailClients returns open counts broken down by email client.
// GET /stats/outbound/opens/emailclients
func (a *API) GetOutboundOpenEmailClients(params StatsParams) (*OpenEmailClientsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/opens/emailclients"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data OpenEmailClientsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundClickCounts returns click counts over time.
// GET /stats/outbound/clicks
func (a *API) GetOutboundClickCounts(params StatsParams) (*ClickCountsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/clicks"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
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

// GetOutboundClickBrowserFamilies returns click counts broken down by browser family.
// GET /stats/outbound/clicks/browserfamilies
func (a *API) GetOutboundClickBrowserFamilies(params StatsParams) (*ClickBrowserFamiliesResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/clicks/browserfamilies"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ClickBrowserFamiliesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundClickPlatforms returns click counts broken down by platform.
// GET /stats/outbound/clicks/platforms
func (a *API) GetOutboundClickPlatforms(params StatsParams) (*ClickPlatformsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/clicks/platforms"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ClickPlatformsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundClickLocations returns click counts broken down by location.
// GET /stats/outbound/clicks/location
func (a *API) GetOutboundClickLocations(params StatsParams) (*ClickLocationsResp, error) {
	q := statsQuery(params)
	path := "stats/outbound/clicks/location"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ClickLocationsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
