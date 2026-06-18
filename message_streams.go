package postmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// MessageStreamType is a named string type for the Postmark message stream
// category. Using a named type instead of a plain string prevents silent typos
// at compile time when the caller constructs a CreateMessageStreamReq.
type MessageStreamType string

const (
	// MessageStreamTypeTransactional is used for transactional email streams.
	MessageStreamTypeTransactional MessageStreamType = "Transactional"
	// MessageStreamTypeBroadcasts is used for broadcast / bulk email streams.
	MessageStreamTypeBroadcasts MessageStreamType = "Broadcasts"
	// MessageStreamTypeInboundSpam is used for inbound spam streams.
	MessageStreamTypeInboundSpam MessageStreamType = "InboundSpam"
)

type (
	// SubscriptionManagementConfiguration holds the unsubscribe handling config
	// for a message stream.
	SubscriptionManagementConfiguration struct {
		UnsubscribeHandlingType string `json:"UnsubscribeHandlingType"`
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
		SubscriptionManagementConfiguration *SubscriptionManagementConfiguration `json:"SubscriptionManagementConfiguration,omitempty"`
	}

	// ListMessageStreamsResp is the response envelope returned by the
	// list-message-streams endpoint.
	ListMessageStreamsResp struct {
		TotalCount     int                 `json:"TotalCount"`
		MessageStreams []MessageStreamResp `json:"MessageStreams"`
	}

	// CreateMessageStreamReq is the request body for creating a new message stream.
	CreateMessageStreamReq struct {
		ID                                  string                               `json:"ID"`
		Name                                string                               `json:"Name"`
		Description                         string                               `json:"Description,omitempty"`
		MessageStreamType                   MessageStreamType                    `json:"MessageStreamType"`
		SubscriptionManagementConfiguration *SubscriptionManagementConfiguration `json:"SubscriptionManagementConfiguration,omitempty"`
	}

	// UpdateMessageStreamReq is the request body for updating an existing message stream.
	// Only the fields provided will be changed.
	UpdateMessageStreamReq struct {
		Name                                string                               `json:"Name,omitempty"`
		Description                         string                               `json:"Description,omitempty"`
		SubscriptionManagementConfiguration *SubscriptionManagementConfiguration `json:"SubscriptionManagementConfiguration,omitempty"`
	}

	// MessageStreamArchiveResp is the response returned when a message stream
	// is archived.
	MessageStreamArchiveResp struct {
		ID        string     `json:"ID"`
		ServerID  int        `json:"ServerID"`
		ExpungeAt *time.Time `json:"ExpungeAt"`
	}
)

// errEmptyStreamID is returned by any method that requires a non-empty stream ID.
var errEmptyStreamID = errors.New("streamID must not be empty")

// newServerRequest is like newRequest but replaces the X-Postmark-Account-Token
// header with X-Postmark-Server-Token, as required by the Message Streams and
// Suppressions endpoints. a.token is expected to hold the server-level API token
// (set via APITokenOpt or the POSTMARK_API_TOKEN environment variable).
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

// GetMessageStream fetches the message stream identified by streamID.
func (a *API) GetMessageStream(streamID string) (*MessageStreamResp, error) {
	if streamID == "" {
		return nil, errEmptyStreamID
	}

	req, err := a.newServerRequest(http.MethodGet, fmt.Sprintf("message-streams/%s", url.PathEscape(streamID)), nil)
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

// CreateMessageStream creates a new message stream with the settings in req.
// req must not be nil; passing nil will return an error before any network call.
// It returns the full MessageStreamResp on success.
func (a *API) CreateMessageStream(req *CreateMessageStreamReq) (*MessageStreamResp, error) {
	if req == nil {
		return nil, fmt.Errorf("CreateMessageStreamReq must not be nil")
	}

	httpReq, err := a.newServerRequest(http.MethodPost, "message-streams", req)
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

// UpdateMessageStream applies the changes in req to the message stream
// identified by streamID and returns the updated MessageStreamResp.
// req must not be nil; passing nil will return an error before any network call.
func (a *API) UpdateMessageStream(streamID string, req *UpdateMessageStreamReq) (*MessageStreamResp, error) {
	if streamID == "" {
		return nil, errEmptyStreamID
	}
	if req == nil {
		return nil, fmt.Errorf("UpdateMessageStreamReq must not be nil")
	}

	httpReq, err := a.newServerRequest(http.MethodPatch, fmt.Sprintf("message-streams/%s", url.PathEscape(streamID)), req)
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

// ArchiveMessageStream archives the message stream identified by streamID.
// It returns a MessageStreamArchiveResp containing the expunge timestamp.
// An empty JSON body ({}) is sent so that the server receives a valid
// Content-Type: application/json header, which some Postmark endpoints require.
func (a *API) ArchiveMessageStream(streamID string) (*MessageStreamArchiveResp, error) {
	if streamID == "" {
		return nil, errEmptyStreamID
	}

	// Pass an empty struct so newRequest sets Content-Type: application/json
	// and sends `{}` as the body, satisfying Postmark's POST requirements.
	httpReq, err := a.newServerRequest(
		http.MethodPost,
		fmt.Sprintf("message-streams/%s/archive", url.PathEscape(streamID)),
		struct{}{},
	)
	if err != nil {
		return nil, err
	}

	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var data MessageStreamArchiveResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// UnarchiveMessageStream unarchives the message stream identified by streamID
// and returns the restored MessageStreamResp.
// An empty JSON body ({}) is sent so that the server receives a valid
// Content-Type: application/json header, which some Postmark endpoints require.
func (a *API) UnarchiveMessageStream(streamID string) (*MessageStreamResp, error) {
	if streamID == "" {
		return nil, errEmptyStreamID
	}

	// Pass an empty struct so newRequest sets Content-Type: application/json
	// and sends `{}` as the body, satisfying Postmark's POST requirements.
	httpReq, err := a.newServerRequest(
		http.MethodPost,
		fmt.Sprintf("message-streams/%s/unarchive", url.PathEscape(streamID)),
		struct{}{},
	)
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
