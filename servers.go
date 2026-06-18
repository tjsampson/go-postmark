package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// CreateServerReq is the request body for creating a new Postmark Server.
	CreateServerReq struct {
		Name             string `json:"Name"`
		Color            string `json:"Color"`
		SmtpApiActivated bool   `json:"SmtpApiActivated"`
	}

	// UpdateServerReq is the request body for updating an existing Postmark Server.
	// Only the fields provided will be changed.
	UpdateServerReq struct {
		Name                       string `json:"Name"`
		Color                      string `json:"Color"`
		SmtpApiActivated           bool   `json:"SmtpApiActivated"`
		RawEmailEnabled            bool   `json:"RawEmailEnabled"`
		InboundHookUrl             string `json:"InboundHookUrl"`
		BounceHookUrl              string `json:"BounceHookUrl"`
		OpenHookUrl                string `json:"OpenHookUrl"`
		DeliveryHookUrl            string `json:"DeliveryHookUrl"`
		PostFirstOpenOnly          bool   `json:"PostFirstOpenOnly"`
		InboundDomain              string `json:"InboundDomain"`
		InboundSpamThreshold       int    `json:"InboundSpamThreshold"`
		TrackOpens                 bool   `json:"TrackOpens"`
		TrackLinks                 string `json:"TrackLinks"`
		IncludeBounceContentInHook bool   `json:"IncludeBounceContentInHook"`
		ClickHookUrl               string `json:"ClickHookUrl"`
		EnableSmtpApiErrorHooks    bool   `json:"EnableSmtpApiErrorHooks"`
	}

	// ServerResp represents a Postmark Server as returned by the API.
	ServerResp struct {
		ID                         int      `json:"ID"`
		Name                       string   `json:"Name"`
		ApiTokens                  []string `json:"ApiTokens"`
		Color                      string   `json:"Color"`
		SmtpApiActivated           bool     `json:"SmtpApiActivated"`
		RawEmailEnabled            bool     `json:"RawEmailEnabled"`
		DeliveryType               string   `json:"DeliveryType"`
		ServerLink                 string   `json:"ServerLink"`
		InboundAddress             string   `json:"InboundAddress"`
		InboundHookUrl             string   `json:"InboundHookUrl"`
		BounceHookUrl              string   `json:"BounceHookUrl"`
		OpenHookUrl                string   `json:"OpenHookUrl"`
		DeliveryHookUrl            string   `json:"DeliveryHookUrl"`
		PostFirstOpenOnly          bool     `json:"PostFirstOpenOnly"`
		InboundDomain              string   `json:"InboundDomain"`
		InboundHash                string   `json:"InboundHash"`
		InboundSpamThreshold       int      `json:"InboundSpamThreshold"`
		TrackOpens                 bool     `json:"TrackOpens"`
		TrackLinks                 string   `json:"TrackLinks"`
		IncludeBounceContentInHook bool     `json:"IncludeBounceContentInHook"`
		ClickHookUrl               string   `json:"ClickHookUrl"`
		EnableSmtpApiErrorHooks    bool     `json:"EnableSmtpApiErrorHooks"`
	}

	// ListServerResp is the response envelope returned by the list-servers endpoint.
	ListServerResp struct {
		TotalCount int          `json:"TotalCount"`
		Servers    []ServerResp `json:"Servers"`
	}

	// DeleteResp is the response returned when a server is deleted.
	DeleteResp struct {
		ErrorCode int    `json:"ErrorCode"`
		Message   string `json:"Message"`
	}

	// PostmarkErr represents an error response from the Postmark API,
	// containing a numeric ErrorCode and a human-readable Message.
	PostmarkErr struct {
		ErrorCode int    `json:"ErrorCode"`
		Message   string `json:"Message"`
	}
)

// ErrExists is returned when a create operation is rejected because a server
// with the same name already exists (HTTP 409 Conflict).
// Its ErrorCode is 0 because this sentinel is dispatched on the HTTP status
// code (resp.StatusCode == 409), not on a Postmark application error code;
// Postmark uses its own four-digit codes (e.g. 505 for duplicate server name)
// in the JSON body which are unrelated to the HTTP status.
var ErrExists = &PostmarkErr{ErrorCode: 0, Message: "server already exists"}

// ErrNotFound is returned when the requested server does not exist (HTTP 404 Not Found).
// Callers can detect this condition with errors.Is(err, postmark.ErrNotFound).
// Its ErrorCode is 0 for the same reason as ErrExists: dispatch is keyed on
// resp.StatusCode, not the Postmark application error code.
var ErrNotFound = &PostmarkErr{ErrorCode: 0, Message: "server not found"}

// Error implements the error interface for PostmarkErr.
func (pe PostmarkErr) Error() string {
	return fmt.Sprintf("%s Error Code=%v", pe.Message, pe.ErrorCode)
}

// Code returns the numeric Postmark error code from the API response.
func (pe PostmarkErr) Code() int {
	return pe.ErrorCode
}

// Is reports whether pe matches target, enabling errors.Is comparisons.
// Sentinel errors (ErrExists, ErrNotFound) are matched by pointer identity
// before this method is called, so this method handles structural equality
// for non-sentinel PostmarkErr values returned from API responses.
func (pe PostmarkErr) Is(target error) bool {
	var t *PostmarkErr
	switch v := target.(type) {
	case *PostmarkErr:
		t = v
	case PostmarkErr:
		t = &v
	default:
		return false
	}
	return pe.ErrorCode == t.ErrorCode
}

// NewError creates a new *PostmarkErr with the given code and formatted message.
func NewError(code int, format string, a ...interface{}) *PostmarkErr {
	return &PostmarkErr{
		ErrorCode: code,
		Message:   fmt.Sprintf(format, a...),
	}
}

// CreateServer creates a new Postmark Server with the settings in serverReq.
// It returns the full ServerResp on success.
func (a *API) CreateServer(serverReq *CreateServerReq) (*ServerResp, error) {
	req, err := a.newRequest(http.MethodPost, "servers", serverReq)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data ServerResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ReadServer fetches the Postmark Server identified by serverID.
func (a *API) ReadServer(serverID string) (*ServerResp, error) {
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("servers/%s", serverID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data ServerResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateServer applies the changes in body to the Postmark Server identified
// by serverID and returns the updated ServerResp.
func (a *API) UpdateServer(serverID string, body *UpdateServerReq) (*ServerResp, error) {
	req, err := a.newRequest(http.MethodPut, fmt.Sprintf("servers/%s", serverID), body)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data ServerResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ListServers returns a paginated list of all Postmark Servers on the account.
// count controls the page size and offset controls the starting position.
func (a *API) ListServers(count, offset int) (*ListServerResp, error) {
	params := url.Values{}
	params.Set("count", strconv.Itoa(count))
	params.Set("offset", strconv.Itoa(offset))
	req, err := a.newRequest(http.MethodGet, "servers?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data ListServerResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteServer deletes the Postmark Server identified by serverId.
// It returns a DeleteResp containing the outcome message from the API.
func (a *API) DeleteServer(serverId string) (*DeleteResp, error) {
	req, err := a.newRequest(http.MethodDelete, fmt.Sprintf("servers/%s", serverId), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data DeleteResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
