package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// BounceResp represents a single bounce record returned by the Postmark API.
	BounceResp struct {
		ID              int64   `json:"ID"`
		Type            string  `json:"Type"`
		TypeCode        int     `json:"TypeCode"`
		Name            string  `json:"Name"`
		Tag             string  `json:"Tag"`
		MessageID       string  `json:"MessageID"`
		ServerID        int64   `json:"ServerID"`
		MessageStream   string  `json:"MessageStream"`
		Description     string  `json:"Description"`
		Details         string  `json:"Details"`
		Email           string  `json:"Email"`
		From            string  `json:"From"`
		BouncedAt       string  `json:"BouncedAt"`
		DumpAvailable   bool    `json:"DumpAvailable"`
		Inactive        bool    `json:"Inactive"`
		CanActivate     bool    `json:"CanActivate"`
		Subject         string  `json:"Subject"`
		Content         string  `json:"Content"`
	}

	// BounceCountByType holds the count of bounces for a specific bounce type.
	BounceCountByType struct {
		Name  string `json:"Name"`
		Count int    `json:"Count"`
		Type  string `json:"Type"`
	}

	// DeliveryStatsResp is the response from GET /deliverystats.
	DeliveryStatsResp struct {
		InactiveMails int                 `json:"InactiveMails"`
		Bounces       []BounceCountByType `json:"Bounces"`
	}

	// GetBouncesParams holds the optional query parameters for GET /bounces.
	GetBouncesParams struct {
		Count           int
		Offset          int
		Type            string
		Inactive        *bool
		EmailFilter     string
		Tag             string
		MessageID       string
		FromDate        string
		ToDate          string
		MessageStreamID string
	}

	// GetBouncesResp is the response from GET /bounces.
	GetBouncesResp struct {
		TotalCount int          `json:"TotalCount"`
		Bounces    []BounceResp `json:"Bounces"`
	}

	// BounceDumpResp is the response from GET /bounces/{bounceID}/dump.
	BounceDumpResp struct {
		Body string `json:"Body"`
	}

	// ActivateBounceResp is the response from PUT /bounces/{bounceID}/activate.
	ActivateBounceResp struct {
		Message string     `json:"Message"`
		Bounce  BounceResp `json:"Bounce"`
	}
)

// newServerRequest builds an *http.Request for the Bounce API using the
// X-Postmark-Server-Token header required by server-level endpoints.
func (a *API) newServerRequest(method, path string, body interface{}) (*http.Request, error) {
	req, err := a.newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	// The Bounce API requires a server token, not an account token.
	req.Header.Del("X-Postmark-Account-Token")
	req.Header.Set("X-Postmark-Server-Token", a.token)
	return req, nil
}

// GetDeliveryStats returns delivery statistics including counts of each bounce
// type. It calls GET /deliverystats.
func (a *API) GetDeliveryStats() (*DeliveryStatsResp, error) {
	req, err := a.newServerRequest(http.MethodGet, "deliverystats", nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data DeliveryStatsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetBounces returns a paginated list of bounces filtered by the supplied
// params. It calls GET /bounces with the query parameters derived from params.
func (a *API) GetBounces(params *GetBouncesParams) (*GetBouncesResp, error) {
	q := url.Values{}
	if params != nil {
		if params.Count > 0 {
			q.Set("count", strconv.Itoa(params.Count))
		}
		if params.Offset > 0 {
			q.Set("offset", strconv.Itoa(params.Offset))
		}
		if params.Type != "" {
			q.Set("type", params.Type)
		}
		if params.Inactive != nil {
			q.Set("inactive", strconv.FormatBool(*params.Inactive))
		}
		if params.EmailFilter != "" {
			q.Set("emailFilter", params.EmailFilter)
		}
		if params.Tag != "" {
			q.Set("tag", params.Tag)
		}
		if params.MessageID != "" {
			q.Set("messageID", params.MessageID)
		}
		if params.FromDate != "" {
			q.Set("fromDate", params.FromDate)
		}
		if params.ToDate != "" {
			q.Set("toDate", params.ToDate)
		}
		if params.MessageStreamID != "" {
			q.Set("messageStreamID", params.MessageStreamID)
		}
	}

	path := "bounces"
	if len(q) > 0 {
		path = "bounces?" + q.Encode()
	}

	req, err := a.newServerRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data GetBouncesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetBounce returns the bounce record identified by bounceID.
// It calls GET /bounces/{bounceID}.
func (a *API) GetBounce(bounceID int64) (*BounceResp, error) {
	req, err := a.newServerRequest(http.MethodGet, fmt.Sprintf("bounces/%d", bounceID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data BounceResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetBounceDump returns the raw SMTP data dump for the bounce identified by
// bounceID. It calls GET /bounces/{bounceID}/dump.
func (a *API) GetBounceDump(bounceID int64) (*BounceDumpResp, error) {
	req, err := a.newServerRequest(http.MethodGet, fmt.Sprintf("bounces/%d/dump", bounceID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data BounceDumpResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ActivateBounce reactivates the email address associated with the bounce
// identified by bounceID. It calls PUT /bounces/{bounceID}/activate.
func (a *API) ActivateBounce(bounceID int64) (*ActivateBounceResp, error) {
	req, err := a.newServerRequest(http.MethodPut, fmt.Sprintf("bounces/%d/activate", bounceID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data ActivateBounceResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetBounceTags returns the list of tags associated with bounces on the server.
// It calls GET /bounces/tags.
func (a *API) GetBounceTags() ([]string, error) {
	req, err := a.newServerRequest(http.MethodGet, "bounces/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data []string
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return data, nil
}
