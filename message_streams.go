package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type (
	// SubscriptionManagementConfiguration holds the unsubscribe handling config
	// for a message stream.
	SubscriptionManagementConfiguration struct {
		UnsubscribeHandlingType string `json:"UnsubscribeHandlingType"`
	}

	// MessageStreamResp represents a Postmark Message Stream as returned by the API.
	MessageStreamResp struct {
		ID                                   string                               `json:"ID"`
		ServerID                             int                                  `json:"ServerID"`
		Name                                 string                               `json:"Name"`
		Description                          string                               `json:"Description"`
		MessageStreamType                    string                               `json:"MessageStreamType"`
		CreatedAt                            time.Time                            `json:"CreatedAt"`
		ArchivedAt                           *time.Time                           `json:"ArchivedAt"`
		ExpungeAt                            *time.Time                           `json:"ExpungeAt"`
		SubscriptionManagementConfiguration  SubscriptionManagementConfiguration  `json:"SubscriptionManagementConfiguration"`
	}

	// ListMessageStreamsResp is the response envelope returned by the
	// list-message-streams endpoint.
	ListMessageStreamsResp struct {
		TotalCount     int                 `json:"TotalCount"`
		MessageStreams []MessageStreamResp `json:"MessageStreams"`
	}

	// CreateMessageStreamReq is the request body for creating a new message stream.
	CreateMessageStreamReq struct {
		ID                                  string                              `json:"ID"`
		Name                                string                              `json:"Name"`
		Description                         string                              `json:"Description,omitempty"`
		MessageStreamType                   string                              `json:"MessageStreamType"`
		SubscriptionManagementConfiguration SubscriptionManagementConfiguration `json:"SubscriptionManagementConfiguration,omitempty"`
	}

	// UpdateMessageStreamReq is the request body for updating an existing message stream.
	// Only the fields provided will be changed.
	UpdateMessageStreamReq struct {
		Name                                string                              `json:"Name,omitempty"`
		Description                         string                              `json:"Description,omitempty"`
		SubscriptionManagementConfiguration SubscriptionManagementConfiguration `json:"SubscriptionManagementConfiguration,omitempty"`
	}

	// MessageStreamArchiveResp is the response returned when a message stream
	// is archived.
	MessageStreamArchiveResp struct {
		ID         string     `json:"ID"`
		ServerID   int        `json:"ServerID"`
		ExpungeAt  *time.Time `json:"ExpungeAt"`
	}
)

// newServerRequest is like newRequest but uses the X-Postmark-Server-Token
// header instead of X-Postmark-Account-Token, as required by the Message
// Streams and Suppressions endpoints.
func (a *API) newServerRequest(method, path string, body interface{}) (*http.Request, error) {
	req, err := a.newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	// Replace the account-level token header with the server-level token header.
	req.Header.Del("X-Postmark-Account-Token")
	req.Header.Set("X-Postmark-Server-Token", a.token)
	return req, nil
}

// ListMessageStreams returns all message streams on the server. Optionally
// filter by streamType (e.g. "Transactional", "Broadcasts") and/or include
// archived streams.
func (a *API) ListMessageStreams(streamType string, includeArchived bool) (*ListMessageStreamsResp, error) {
	params := url.Values{}
	if streamType != "" {
		params.Set("MessageStreamType", streamType)
	}
	if includeArchived {
		params.Set("IncludeArchivedStreams", "true")
	}

	path := "message-streams"
	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}

	req, err := a.newServerRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data ListMessageStreamsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetMessageStream fetches the message stream identified by streamID.
func (a *API) GetMessageStream(streamID string) (*MessageStreamResp, error) {
	req, err := a.newServerRequest(http.MethodGet, fmt.Sprintf("message-streams/%s", streamID), nil)
	if err != nil {
		return nil, err
	}

	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data MessageStreamResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateMessageStream creates a new message stream with the settings in req.
// It returns the full MessageStreamResp on success.
func (a *API) CreateMessageStream(req *CreateMessageStreamReq) (*MessageStreamResp, error) {
	httpReq, err := a.newServerRequest(http.MethodPost, "message-streams", req)
	if err != nil {
		return nil, err
	}

	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data MessageStreamResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateMessageStream applies the changes in req to the message stream
// identified by streamID and returns the updated MessageStreamResp.
func (a *API) UpdateMessageStream(streamID string, req *UpdateMessageStreamReq) (*MessageStreamResp, error) {
	httpReq, err := a.newServerRequest(http.MethodPatch, fmt.Sprintf("message-streams/%s", streamID), req)
	if err != nil {
		return nil, err
	}

	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data MessageStreamResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ArchiveMessageStream archives the message stream identified by streamID.
// It returns a MessageStreamArchiveResp containing the expunge timestamp.
func (a *API) ArchiveMessageStream(streamID string) (*MessageStreamArchiveResp, error) {
	httpReq, err := a.newServerRequest(http.MethodPost, fmt.Sprintf("message-streams/%s/archive", streamID), nil)
	if err != nil {
		return nil, err
	}

	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data MessageStreamArchiveResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UnarchiveMessageStream unarchives the message stream identified by streamID
// and returns the restored MessageStreamResp.
func (a *API) UnarchiveMessageStream(streamID string) (*MessageStreamResp, error) {
	httpReq, err := a.newServerRequest(http.MethodPost, fmt.Sprintf("message-streams/%s/unarchive", streamID), nil)
	if err != nil {
		return nil, err
	}

	resp, e := a.Do(httpReq)
	if e != nil {
		return nil, e
	}

	var data MessageStreamResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
