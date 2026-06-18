package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type (
	// SubscriptionManagementConfiguration holds configuration for subscription
	// management on a message stream.
	SubscriptionManagementConfiguration struct {
		UnsubscribeHandlingType string `json:"UnsubscribeHandlingType"`
	}

	// MessageStreamResp represents a Postmark Message Stream as returned by the API.
	MessageStreamResp struct {
		ID                                  string                               `json:"ID"`
		ServerID                            int                                  `json:"ServerID"`
		Name                                string                               `json:"Name"`
		Description                         string                               `json:"Description"`
		MessageStreamType                   string                               `json:"MessageStreamType"`
		CreatedAt                           string                               `json:"CreatedAt"`
		ArchivedAt                          *string                              `json:"ArchivedAt"`
		ExpungeAt                           *string                              `json:"ExpungeAt"`
		SubscriptionManagementConfiguration *SubscriptionManagementConfiguration `json:"SubscriptionManagementConfiguration"`
	}

	// CreateMessageStreamReq is the request body for creating a new Message Stream.
	CreateMessageStreamReq struct {
		ID                string `json:"ID"`
		Name              string `json:"Name"`
		MessageStreamType string `json:"MessageStreamType"`
		Description       string `json:"Description,omitempty"`
	}

	// EditMessageStreamReq is the request body for editing an existing Message Stream.
	EditMessageStreamReq struct {
		Name        string `json:"Name,omitempty"`
		Description string `json:"Description,omitempty"`
	}

	// ListMessageStreamsResp is the response envelope returned by the list message streams endpoint.
	ListMessageStreamsResp struct {
		MessageStreams []MessageStreamResp `json:"MessageStreams"`
		TotalCount     int                 `json:"TotalCount"`
	}

	// ArchiveMessageStreamResp is the response returned when a message stream is archived.
	ArchiveMessageStreamResp struct {
		ID          string  `json:"ID"`
		ServerID    int     `json:"ServerID"`
		Name        string  `json:"Name"`
		Description string  `json:"Description"`
		ArchivedAt  string  `json:"ArchivedAt"`
		ExpungeAt   string  `json:"ExpungeAt"`
		ErrorCode   *int    `json:"ErrorCode"`
		Message     *string `json:"Message"`
	}
)

// ListMessageStreams returns a list of all Message Streams for the server.
// Pass includeArchivedStr as "true" to include archived streams, or "" to omit the param.
func (a *API) ListMessageStreams(includeArchivedStr string) (*ListMessageStreamsResp, error) {
	path := "message-streams"
	if includeArchivedStr != "" {
		params := url.Values{}
		params.Set("includeArchived", includeArchivedStr)
		path = path + "?" + params.Encode()
	}

	req, err := a.newRequest(http.MethodGet, path, nil)
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

// GetMessageStream fetches the Message Stream identified by streamID.
func (a *API) GetMessageStream(streamID string) (*MessageStreamResp, error) {
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("message-streams/%s", streamID), nil)
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

// CreateMessageStream creates a new Message Stream with the settings in req.
// It returns the full MessageStreamResp on success.
func (a *API) CreateMessageStream(req *CreateMessageStreamReq) (*MessageStreamResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "message-streams", req)
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

// EditMessageStream applies the changes in req to the Message Stream identified
// by streamID and returns the updated MessageStreamResp.
func (a *API) EditMessageStream(streamID string, req *EditMessageStreamReq) (*MessageStreamResp, error) {
	httpReq, err := a.newRequest(http.MethodPatch, fmt.Sprintf("message-streams/%s", streamID), req)
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

// ArchiveMessageStream archives the Message Stream identified by streamID.
// Archived streams are scheduled for permanent deletion after a grace period.
func (a *API) ArchiveMessageStream(streamID string) (*ArchiveMessageStreamResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("message-streams/%s/archive", streamID), nil)
	if err != nil {
		return nil, err
	}

	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}

	var data ArchiveMessageStreamResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UnarchiveMessageStream restores a previously archived Message Stream identified
// by streamID and returns the updated MessageStreamResp.
func (a *API) UnarchiveMessageStream(streamID string) (*MessageStreamResp, error) {
	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("message-streams/%s/unarchive", streamID), nil)
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
