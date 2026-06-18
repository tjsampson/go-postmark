package postmark

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	// This is a Postmark-defined stream type for processing inbound spam;
	// it is included here for completeness but is not typically user-created.
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
		ArchivedAt                          *time.Time                           `json:"ArchivedAt,omitempty"`
		ExpungeAt                           *time.Time                           `json:"ExpungeAt,omitempty"`
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
		// ID is the unique slug for the new stream. It must not be empty.
		ID                                  string                               `json:"ID,omitempty"`
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
		ExpungeAt *time.Time `json:"ExpungeAt,omitempty"`
	}
)

// errEmptyStreamID is returned by any method that requires a non-empty stream ID.
var errEmptyStreamID = errors.New("streamID must not be empty")

// errNilCreateReq is returned when a nil *CreateMessageStreamReq is passed.
var errNilCreateReq = errors.New("CreateMessageStreamReq must not be nil")

// errNilUpdateReq is returned when a nil *UpdateMessageStreamReq is passed.
var errNilUpdateReq = errors.New("UpdateMessageStreamReq must not be nil")

// newServerRequest builds an *http.Request that carries X-Postmark-Server-Token
// (not X-Postmark-Account-Token) as required by the Message Streams and
// Suppressions endpoints. It is a dedicated builder that never touches the
// account-level auth header, avoiding the fragility of deleting a header that
// may or may not have been set by a base helper.
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
		reqBody,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	// Message Streams endpoints authenticate with the server-level token.
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
		path += "?" + params.Encode()
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
// req must not be nil; passing nil will return errNilCreateReq immediately,
// before any network call. req.ID must not be empty. It returns the full
// MessageStreamResp on success.
func (a *API) CreateMessageStream(req *CreateMessageStreamReq) (*MessageStreamResp, error) {
	if req == nil {
		return nil, errNilCreateReq
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
// req must not be nil; passing nil will return errNilUpdateReq immediately,
// before any network call.
func (a *API) UpdateMessageStream(streamID string, req *UpdateMessageStreamReq) (*MessageStreamResp, error) {
	if streamID == "" {
		return nil, errEmptyStreamID
	}
	if req == nil {
		return nil, errNilUpdateReq
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

	// Pass an empty struct so newServerRequest sets Content-Type: application/json
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

	// Pass an empty struct so newServerRequest sets Content-Type: application/json
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
