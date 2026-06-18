package postmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// MessageStreamType is the set of allowed values for the MessageStreamType field.
type MessageStreamType string

const (
	// MessageStreamTypeTransactional is a transactional message stream.
	MessageStreamTypeTransactional MessageStreamType = "Transactional"
	// MessageStreamTypeBroadcasts is a broadcast message stream.
	MessageStreamTypeBroadcasts MessageStreamType = "Broadcasts"
	// MessageStreamTypeInbound is an inbound message stream.
	MessageStreamTypeInbound MessageStreamType = "Inbound"
)

// UnsubscribeHandlingType is the set of allowed values for
// SubscriptionManagementConfiguration.UnsubscribeHandlingType.
type UnsubscribeHandlingType string

const (
	// UnsubscribeHandlingNone means no unsubscribe handling is configured.
	UnsubscribeHandlingNone UnsubscribeHandlingType = "None"
	// UnsubscribeHandlingPostmarkManaged means Postmark handles unsubscribes.
	UnsubscribeHandlingPostmarkManaged UnsubscribeHandlingType = "PostmarkManaged"
	// UnsubscribeHandlingCustom means the caller handles unsubscribes.
	UnsubscribeHandlingCustom UnsubscribeHandlingType = "Custom"
)

type (
	// SubscriptionManagementConfiguration holds configuration for subscription
	// management on a message stream.
	SubscriptionManagementConfiguration struct {
		UnsubscribeHandlingType UnsubscribeHandlingType `json:"UnsubscribeHandlingType"`
	}

	// MessageStreamResp represents a Postmark Message Stream as returned by the API.
	MessageStreamResp struct {
		ID                                  string                               `json:"ID"`
		ServerID                            int                                  `json:"ServerID"`
		Name                                string                               `json:"Name"`
		Description                         string                               `json:"Description"`
		MessageStreamType                   MessageStreamType                    `json:"MessageStreamType"`
		CreatedAt                           time.Time                            `json:"CreatedAt"`
		ArchivedAt                          *time.Time                           `json:"ArchivedAt"`
		ExpungeAt                           *time.Time                           `json:"ExpungeAt"`
		SubscriptionManagementConfiguration *SubscriptionManagementConfiguration `json:"SubscriptionManagementConfiguration"`
	}

	// CreateMessageStreamReq is the request body for creating a new Message Stream.
	CreateMessageStreamReq struct {
		ID                string            `json:"ID"`
		Name              string            `json:"Name"`
		MessageStreamType MessageStreamType `json:"MessageStreamType"`
		Description       string            `json:"Description,omitempty"`
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
	// It embeds MessageStreamResp and adds the ErrorCode and Message fields that
	// Postmark may include in error scenarios on this endpoint.
	ArchiveMessageStreamResp struct {
		MessageStreamResp
		ErrorCode *int    `json:"ErrorCode"`
		Message   *string `json:"Message"`
	}
)

// ListMessageStreams returns a list of all Message Streams for the server.
// Pass includeArchived as true to include archived streams in the results;
// false uses the API default (archived streams excluded).
func (a *API) ListMessageStreams(includeArchived bool) (*ListMessageStreamsResp, error) {
	path := "message-streams"
	if includeArchived {
		path += "?includeArchived=true"
	}

	req, err := a.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data ListMessageStreamsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetMessageStream fetches the Message Stream identified by streamID.
func (a *API) GetMessageStream(streamID string) (*MessageStreamResp, error) {
	if streamID == "" {
		return nil, errors.New("streamID must not be empty")
	}

	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("message-streams/%s", url.PathEscape(streamID)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data MessageStreamResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateMessageStream creates a new Message Stream with the settings in req.
// It returns the full MessageStreamResp on success.
// ID, Name, and MessageStreamType are required fields and must not be empty.
func (a *API) CreateMessageStream(req *CreateMessageStreamReq) (*MessageStreamResp, error) {
	if req == nil {
		return nil, errors.New("req must not be nil")
	}
	if req.ID == "" {
		return nil, errors.New("req.ID must not be empty")
	}
	if req.Name == "" {
		return nil, errors.New("req.Name must not be empty")
	}
	if req.MessageStreamType == "" {
		return nil, errors.New("req.MessageStreamType must not be empty")
	}

	httpReq, err := a.newRequest(http.MethodPost, "message-streams", req)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
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
	if streamID == "" {
		return nil, errors.New("streamID must not be empty")
	}
	if req == nil {
		return nil, errors.New("req must not be nil")
	}

	httpReq, err := a.newRequest(http.MethodPatch, fmt.Sprintf("message-streams/%s", url.PathEscape(streamID)), req)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
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
	if streamID == "" {
		return nil, errors.New("streamID must not be empty")
	}

	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("message-streams/%s/archive", url.PathEscape(streamID)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(req)
	if err != nil {
		return nil, err
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
	if streamID == "" {
		return nil, errors.New("streamID must not be empty")
	}

	req, err := a.newRequest(http.MethodPost, fmt.Sprintf("message-streams/%s/unarchive", url.PathEscape(streamID)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data MessageStreamResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
