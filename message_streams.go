package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type (
	// SubscriptionManagementConfiguration holds the unsubscribe handling settings
	// for a message stream.
	SubscriptionManagementConfiguration struct {
		UnsubscribeHandlingType string `json:"UnsubscribeHandlingType"`
	}

	// MessageStream represents a Postmark Message Stream as returned by the API.
	MessageStream struct {
		ID                                  string                              `json:"ID"`
		Name                                string                              `json:"Name"`
		Description                         string                              `json:"Description"`
		MessageStreamType                   string                              `json:"MessageStreamType"`
		ServerID                            int                                 `json:"ServerID"`
		CreatedAt                           time.Time                           `json:"CreatedAt"`
		UpdatedAt                           time.Time                           `json:"UpdatedAt"`
		ArchivedAt                          *time.Time                          `json:"ArchivedAt"`
		ExpectedPurgeDate                   *time.Time                          `json:"ExpectedPurgeDate"`
		SubscriptionManagementConfiguration SubscriptionManagementConfiguration `json:"SubscriptionManagementConfiguration"`
	}

	// CreateMessageStreamReq is the request body for creating a new Message Stream.
	CreateMessageStreamReq struct {
		ID                string `json:"ID"`
		Name              string `json:"Name"`
		Description       string `json:"Description"`
		MessageStreamType string `json:"MessageStreamType"`
	}

	// UpdateMessageStreamReq is the request body for updating an existing Message Stream.
	// Only the fields provided will be changed. Fields with omitempty are omitted when
	// empty, preventing unintentional overwrites of existing values.
	UpdateMessageStreamReq struct {
		Name        string `json:"Name,omitempty"`
		Description string `json:"Description,omitempty"`
	}

	// ListMessageStreamsResp is the response envelope returned by the list message streams endpoint.
	ListMessageStreamsResp struct {
		MessageStreams []MessageStream `json:"MessageStreams"`
		TotalCount     int             `json:"TotalCount"`
	}

	// ArchiveMessageStreamResp is the response returned when a message stream is archived.
	ArchiveMessageStreamResp struct {
		ID                string     `json:"ID"`
		ServerID          int        `json:"ServerID"`
		ExpectedPurgeDate *time.Time `json:"ExpectedPurgeDate"`
		ArchivedAt        *time.Time `json:"ArchivedAt"`
	}
)

// ListMessageStreams returns a list of message streams, optionally filtered by
// streamType ("Transactional", "Inbound", "Broadcasts", or "" for all) and
// whether to include archived streams.
func (a *API) ListMessageStreams(streamType string, includeArchived bool) (*ListMessageStreamsResp, error) {
	params := url.Values{}
	if streamType != "" {
		params.Set("MessageStreamType", streamType)
	}
	params.Set("IncludeArchivedStreams", fmt.Sprintf("%t", includeArchived))

	req, err := a.newServerRequest(http.MethodGet, "message-streams?"+params.Encode(), nil)
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

// GetMessageStream fetches the Message Stream identified by id.
func (a *API) GetMessageStream(id string) (*MessageStream, error) {
	req, err := a.newServerRequest(http.MethodGet, fmt.Sprintf("message-streams/%s", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}

	var data MessageStream
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateMessageStream creates a new Message Stream with the settings in req.
// It returns the full MessageStream on success.
func (a *API) CreateMessageStream(req *CreateMessageStreamReq) (*MessageStream, error) {
	httpReq, err := a.newServerRequest(http.MethodPost, "message-streams", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data MessageStream
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateMessageStream applies the changes in req to the Message Stream identified
// by id and returns the updated MessageStream.
func (a *API) UpdateMessageStream(id string, req *UpdateMessageStreamReq) (*MessageStream, error) {
	httpReq, err := a.newServerRequest(http.MethodPut, fmt.Sprintf("message-streams/%s", url.PathEscape(id)), req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data MessageStream
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ArchiveMessageStream archives the Message Stream identified by id.
// It returns an ArchiveMessageStreamResp containing the outcome from the API.
func (a *API) ArchiveMessageStream(id string) (*ArchiveMessageStreamResp, error) {
	httpReq, err := a.newServerRequest(http.MethodPost, fmt.Sprintf("message-streams/%s/archive", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data ArchiveMessageStreamResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UnarchiveMessageStream unarchives the Message Stream identified by id.
// It returns the updated MessageStream on success.
func (a *API) UnarchiveMessageStream(id string) (*MessageStream, error) {
	httpReq, err := a.newServerRequest(http.MethodPost, fmt.Sprintf("message-streams/%s/unarchive", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data MessageStream
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
