package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type (
	// BounceResp represents a single bounce record returned by the Postmark API.
	// Fields mirror the Postmark Bounce API response schema documented at
	// https://postmarkapp.com/developer/api/bounce-api.
	BounceResp struct {
		ID            int64     `json:"ID"`
		Type          string    `json:"Type"`
		TypeCode      int       `json:"TypeCode"`
		Name          string    `json:"Name"`
		Tag           string    `json:"Tag"`
		MessageID     string    `json:"MessageID"`
		ServerID      int64     `json:"ServerID"`
		MessageStream string    `json:"MessageStream"`
		Description   string    `json:"Description"`
		Details       string    `json:"Details"`
		Email         string    `json:"Email"`
		From          string    `json:"From"`
		BouncedAt     time.Time `json:"BouncedAt"`
		DumpAvailable bool      `json:"DumpAvailable"`
		Inactive      bool      `json:"Inactive"`
		CanActivate   bool      `json:"CanActivate"`
		Subject       string    `json:"Subject"`
	}

	// BounceCountByType holds the count of bounces for a specific bounce type.
	BounceCountByType struct {
		Name  string `json:"Name"`
		Count int    `json:"Count"`
		// Type is the machine-readable bounce type key (e.g. "HardBounce"),
		// as distinct from Name which is the human-readable label.
		Type string `json:"Type"`
	}

	// DeliveryStatsResp is the response from GET /deliverystats.
	DeliveryStatsResp struct {
		InactiveMails int                 `json:"InactiveMails"`
		Bounces       []BounceCountByType `json:"Bounces"`
	}

	// GetBouncesParams holds the optional query parameters for GET /bounces.
	//
	// All fields are optional. A nil pointer field (Offset, Inactive) means
	// "omit this parameter from the query string". Use the helper [IntPtr] and
	// [BoolPtr] functions, or take the address of a local variable, to set them.
	//
	// Count: a value of 0 (the zero value) omits the parameter; any value > 0
	// is sent as the page size.
	//
	// Offset: nil means "omit", non-nil means "send this value". To request
	// the first page explicitly, pass a pointer to 0. This is consistent with
	// the Inactive field and avoids the ambiguity of an integer sentinel value.
	GetBouncesParams struct {
		// Count is the number of bounces to return per page.
		// A value of 0 (the zero value) omits the parameter from the query
		// string; any value > 0 is sent as-is.
		Count int
		// Offset is the zero-based starting index for pagination.
		// A nil pointer omits the parameter (Postmark defaults to 0).
		// A non-nil pointer sends the pointed-to value, including 0.
		Offset          *int
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
// It replaces the X-Postmark-Account-Token set by newRequest with the
// correct server-level token header.
//
// Token usage: the API struct holds a single token field (a.token) populated
// by [APITokenOpt]. For bounce/server-level endpoints this value must be a
// Postmark *server* API token (found in the Postmark UI under
// Server → API Credentials). Using an account-level token here will result
// in 401 Unauthorized responses from the Postmark API.
func (a *API) newServerRequest(method, path string, body interface{}) (*http.Request, error) {
	req, err := a.newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	// newRequest sets X-Postmark-Account-Token; remove it and set the
	// server-level header instead so account credentials are not leaked.
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
//
// Pass nil for params to omit all query parameters and use the Postmark API
// defaults.
//
// Count: a value of 0 omits the parameter; any value > 0 is sent as the page
// size.
//
// Offset: a nil pointer omits the parameter (Postmark defaults to 0). A
// non-nil pointer sends the pointed-to value, including 0 to explicitly
// request the first page.
func (a *API) GetBounces(params *GetBouncesParams) (*GetBouncesResp, error) {
	q := url.Values{}
	if params != nil {
		if params.Count > 0 {
			q.Set("count", strconv.Itoa(params.Count))
		}
		if params.Offset != nil {
			q.Set("offset", strconv.Itoa(*params.Offset))
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

	req, err := a.newServerRequest(http.MethodGet, "bounces", nil)
	if err != nil {
		return nil, err
	}
	// Set the query string on the already-parsed URL so that the path
	// segment ("bounces") and the query string remain cleanly separated,
	// rather than embedding a literal '?' in the path string passed to
	// newRequest. Setting an empty string is equivalent to no query string.
	req.URL.RawQuery = q.Encode()

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
