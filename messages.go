package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// OutboundMessageSearchParams holds query parameters for searching outbound messages.
	OutboundMessageSearchParams struct {
		Count         int
		Offset        int
		Recipient     string
		FromEmail     string
		Tag           string
		Status        string
		FromDate      string
		ToDate        string
		MessageStream string
		Subject       string
	}

	// OutboundOpensParams holds query parameters for searching outbound opens.
	OutboundOpensParams struct {
		Count         int
		Offset        int
		Recipient     string
		Tag           string
		ClientName    string
		ClientCompany string
		ClientFamily  string
		OSName        string
		OSFamily      string
		OSCompany     string
		Platform      string
		Country       string
		Region        string
		City          string
	}

	// OutboundClicksParams holds query parameters for searching outbound clicks.
	OutboundClicksParams struct {
		Count         int
		Offset        int
		Recipient     string
		Tag           string
		ClientName    string
		ClientCompany string
		ClientFamily  string
		OSName        string
		OSFamily      string
		OSCompany     string
		Platform      string
		Country       string
		Region        string
		City          string
	}

	// InboundMessageSearchParams holds query parameters for searching inbound messages.
	InboundMessageSearchParams struct {
		Count         int
		Offset        int
		Recipient     string
		FromEmail     string
		Tag           string
		Status        string
		FromDate      string
		ToDate        string
		InboxID       string
		Subject       string
		MailboxHash   string
		MessageStream string
	}

	// OutboundMessageSummary represents a summary of an outbound message.
	OutboundMessageSummary struct {
		TextBody      string            `json:"TextBody"`
		HTMLBody      string            `json:"HtmlBody"`
		Body          string            `json:"Body"`
		Tag           string            `json:"Tag"`
		MessageID     string            `json:"MessageID"`
		To            []To              `json:"To"`
		CC            []To              `json:"Cc"`
		BCC           []To              `json:"Bcc"`
		Recipients    []string          `json:"Recipients"`
		ReceivedAt    string            `json:"ReceivedAt"`
		From          string            `json:"From"`
		Subject       string            `json:"Subject"`
		Attachments   []string          `json:"Attachments"`
		Status        string            `json:"Status"`
		TrackOpens    bool              `json:"TrackOpens"`
		TrackLinks    string            `json:"TrackLinks"`
		MessageStream string            `json:"MessageStream"`
		Metadata      map[string]string `json:"Metadata"`
	}

	// To represents a recipient address.
	To struct {
		Email string `json:"Email"`
		Name  string `json:"Name"`
	}

	// ListOutboundMessagesResp is the response for listing outbound messages.
	ListOutboundMessagesResp struct {
		TotalCount int                      `json:"TotalCount"`
		Messages   []OutboundMessageSummary `json:"Messages"`
	}

	// DeliveryEvent represents a delivery event for a message.
	DeliveryEvent struct {
		Recipient  string `json:"Recipient"`
		Type       string `json:"Type"`
		ReceivedAt string `json:"ReceivedAt"`
		Details    string `json:"Details"`
	}

	// OutboundMessageDetailsResp represents the full details of an outbound message.
	OutboundMessageDetailsResp struct {
		TextBody      string            `json:"TextBody"`
		HTMLBody      string            `json:"HtmlBody"`
		Body          string            `json:"Body"`
		Tag           string            `json:"Tag"`
		MessageID     string            `json:"MessageID"`
		To            []To              `json:"To"`
		CC            []To              `json:"Cc"`
		BCC           []To              `json:"Bcc"`
		Recipients    []string          `json:"Recipients"`
		ReceivedAt    string            `json:"ReceivedAt"`
		From          string            `json:"From"`
		Subject       string            `json:"Subject"`
		Attachments   []string          `json:"Attachments"`
		Status        string            `json:"Status"`
		TrackOpens    bool              `json:"TrackOpens"`
		TrackLinks    string            `json:"TrackLinks"`
		MessageStream string            `json:"MessageStream"`
		Metadata      map[string]string `json:"Metadata"`
		MessageEvents []DeliveryEvent   `json:"MessageEvents"`
	}

	// MessageDumpResp represents the raw email dump for a message.
	MessageDumpResp struct {
		Body string `json:"Body"`
	}

	// OpenEvent represents a single open event.
	OpenEvent struct {
		FirstOpen     bool   `json:"FirstOpen"`
		Client        Client `json:"Client"`
		OS            OS     `json:"OS"`
		Platform      string `json:"Platform"`
		ReadSeconds   int    `json:"ReadSeconds"`
		Geo           Geo    `json:"Geo"`
		MessageID     string `json:"MessageID"`
		ReceivedAt    string `json:"ReceivedAt"`
		Tag           string `json:"Tag"`
		MessageStream string `json:"MessageStream"`
		Recipient     string `json:"Recipient"`
	}

	// Client represents client information from an open or click event.
	Client struct {
		Name    string `json:"Name"`
		Company string `json:"Company"`
		Family  string `json:"Family"`
	}

	// OS represents OS information from an open or click event.
	OS struct {
		Name    string `json:"Name"`
		Company string `json:"Company"`
		Family  string `json:"Family"`
	}

	// Geo represents geographic information from an open or click event.
	Geo struct {
		CountryISOCode string `json:"CountryISOCode"`
		Country        string `json:"Country"`
		RegionISOCode  string `json:"RegionISOCode"`
		Region         string `json:"Region"`
		City           string `json:"City"`
		Zip            string `json:"Zip"`
		Coords         string `json:"Coords"`
		IP             string `json:"IP"`
	}

	// ListOutboundOpensResp is the response for listing outbound opens.
	ListOutboundOpensResp struct {
		TotalCount int         `json:"TotalCount"`
		Opens      []OpenEvent `json:"Opens"`
	}

	// ClickEvent represents a single click event.
	ClickEvent struct {
		Client        Client `json:"Client"`
		OS            OS     `json:"OS"`
		Platform      string `json:"Platform"`
		Geo           Geo    `json:"Geo"`
		MessageID     string `json:"MessageID"`
		ReceivedAt    string `json:"ReceivedAt"`
		Tag           string `json:"Tag"`
		MessageStream string `json:"MessageStream"`
		Recipient     string `json:"Recipient"`
		OriginalLink  string `json:"OriginalLink"`
	}

	// ListOutboundClicksResp is the response for listing outbound clicks.
	ListOutboundClicksResp struct {
		TotalCount int          `json:"TotalCount"`
		Clicks     []ClickEvent `json:"Clicks"`
	}

	// InboundMessageSummary represents a summary of an inbound message.
	InboundMessageSummary struct {
		From          string              `json:"From"`
		FromName      string              `json:"FromName"`
		FromFull      To                  `json:"FromFull"`
		To            string              `json:"To"`
		ToFull        []To                `json:"ToFull"`
		CC            string              `json:"Cc"`
		CCFull        []To                `json:"CcFull"`
		BCC           string              `json:"Bcc"`
		BCCFull       []To                `json:"BccFull"`
		ReplyTo       string              `json:"ReplyTo"`
		Subject       string              `json:"Subject"`
		Date          string              `json:"Date"`
		MailboxHash   string              `json:"MailboxHash"`
		TextBody      string              `json:"TextBody"`
		HTMLBody      string              `json:"HtmlBody"`
		Tag           string              `json:"Tag"`
		Headers       []Header            `json:"Headers"`
		Attachments   []InboundAttachment `json:"Attachments"`
		MessageID     string              `json:"MessageID"`
		Status        string              `json:"Status"`
		MessageStream string              `json:"MessageStream"`
	}

	// Header represents an email header key-value pair.
	Header struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}

	// InboundAttachment represents an email attachment.
	InboundAttachment struct {
		Name          string `json:"Name"`
		Content       string `json:"Content"`
		ContentType   string `json:"ContentType"`
		ContentLength int    `json:"ContentLength"`
	}

	// ListInboundMessagesResp is the response for listing inbound messages.
	ListInboundMessagesResp struct {
		TotalCount      int                     `json:"TotalCount"`
		InboundMessages []InboundMessageSummary `json:"InboundMessages"`
	}

	// InboundMessageDetailsResp represents the full details of an inbound message.
	InboundMessageDetailsResp struct {
		From          string              `json:"From"`
		FromName      string              `json:"FromName"`
		FromFull      To                  `json:"FromFull"`
		To            string              `json:"To"`
		ToFull        []To                `json:"ToFull"`
		CC            string              `json:"Cc"`
		CCFull        []To                `json:"CcFull"`
		BCC           string              `json:"Bcc"`
		BCCFull       []To                `json:"BccFull"`
		ReplyTo       string              `json:"ReplyTo"`
		Subject       string              `json:"Subject"`
		Date          string              `json:"Date"`
		MailboxHash   string              `json:"MailboxHash"`
		TextBody      string              `json:"TextBody"`
		HTMLBody      string              `json:"HtmlBody"`
		Tag           string              `json:"Tag"`
		Headers       []Header            `json:"Headers"`
		Attachments   []InboundAttachment `json:"Attachments"`
		MessageID     string              `json:"MessageID"`
		Status        string              `json:"Status"`
		MessageStream string              `json:"MessageStream"`
	}

	// InboundBypassResp is the response from bypassing inbound message rules.
	InboundBypassResp struct {
		ErrorCode int    `json:"ErrorCode"`
		Message   string `json:"Message"`
	}

	// InboundRetryResp is the response from retrying an inbound message.
	InboundRetryResp struct {
		ErrorCode int    `json:"ErrorCode"`
		Message   string `json:"Message"`
	}
)

// SearchOutboundMessages searches outbound messages using the given parameters.
// GET /messages/outbound
func (a *API) SearchOutboundMessages(params OutboundMessageSearchParams) (*ListOutboundMessagesResp, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(params.Count))
	q.Set("offset", strconv.Itoa(params.Offset))
	if params.Recipient != "" {
		q.Set("recipient", params.Recipient)
	}
	if params.FromEmail != "" {
		q.Set("fromemail", params.FromEmail)
	}
	if params.Tag != "" {
		q.Set("tag", params.Tag)
	}
	if params.Status != "" {
		q.Set("status", params.Status)
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
	if params.Subject != "" {
		q.Set("subject", params.Subject)
	}

	req, err := a.newRequest(http.MethodGet, "messages/outbound?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ListOutboundMessagesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundMessageDetails retrieves the details for an outbound message.
// GET /messages/outbound/{messageid}/details
func (a *API) GetOutboundMessageDetails(messageID string) (*OutboundMessageDetailsResp, error) {
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("messages/outbound/%s/details", messageID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data OutboundMessageDetailsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundMessageDump retrieves the raw email dump for an outbound message.
// GET /messages/outbound/{messageid}/dump
func (a *API) GetOutboundMessageDump(messageID string) (*MessageDumpResp, error) {
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("messages/outbound/%s/dump", messageID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data MessageDumpResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundMessageOpens retrieves open events for outbound messages.
// GET /messages/outbound/opens
func (a *API) GetOutboundMessageOpens(params OutboundOpensParams) (*ListOutboundOpensResp, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(params.Count))
	q.Set("offset", strconv.Itoa(params.Offset))
	if params.Recipient != "" {
		q.Set("recipient", params.Recipient)
	}
	if params.Tag != "" {
		q.Set("tag", params.Tag)
	}
	if params.ClientName != "" {
		q.Set("client_name", params.ClientName)
	}
	if params.ClientCompany != "" {
		q.Set("client_company", params.ClientCompany)
	}
	if params.ClientFamily != "" {
		q.Set("client_family", params.ClientFamily)
	}
	if params.OSName != "" {
		q.Set("os_name", params.OSName)
	}
	if params.OSFamily != "" {
		q.Set("os_family", params.OSFamily)
	}
	if params.OSCompany != "" {
		q.Set("os_company", params.OSCompany)
	}
	if params.Platform != "" {
		q.Set("platform", params.Platform)
	}
	if params.Country != "" {
		q.Set("country", params.Country)
	}
	if params.Region != "" {
		q.Set("region", params.Region)
	}
	if params.City != "" {
		q.Set("city", params.City)
	}

	req, err := a.newRequest(http.MethodGet, "messages/outbound/opens?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ListOutboundOpensResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundMessageOpensByMessageID retrieves open events for a specific outbound message.
// GET /messages/outbound/opens/{messageid}
func (a *API) GetOutboundMessageOpensByMessageID(messageID string, count, offset int) (*ListOutboundOpensResp, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(count))
	q.Set("offset", strconv.Itoa(offset))
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("messages/outbound/opens/%s?%s", messageID, q.Encode()), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ListOutboundOpensResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundMessageClicks retrieves click events for outbound messages.
// GET /messages/outbound/clicks
func (a *API) GetOutboundMessageClicks(params OutboundClicksParams) (*ListOutboundClicksResp, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(params.Count))
	q.Set("offset", strconv.Itoa(params.Offset))
	if params.Recipient != "" {
		q.Set("recipient", params.Recipient)
	}
	if params.Tag != "" {
		q.Set("tag", params.Tag)
	}
	if params.ClientName != "" {
		q.Set("client_name", params.ClientName)
	}
	if params.ClientCompany != "" {
		q.Set("client_company", params.ClientCompany)
	}
	if params.ClientFamily != "" {
		q.Set("client_family", params.ClientFamily)
	}
	if params.OSName != "" {
		q.Set("os_name", params.OSName)
	}
	if params.OSFamily != "" {
		q.Set("os_family", params.OSFamily)
	}
	if params.OSCompany != "" {
		q.Set("os_company", params.OSCompany)
	}
	if params.Platform != "" {
		q.Set("platform", params.Platform)
	}
	if params.Country != "" {
		q.Set("country", params.Country)
	}
	if params.Region != "" {
		q.Set("region", params.Region)
	}
	if params.City != "" {
		q.Set("city", params.City)
	}

	req, err := a.newRequest(http.MethodGet, "messages/outbound/clicks?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ListOutboundClicksResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetOutboundMessageClicksByMessageID retrieves click events for a specific outbound message.
// GET /messages/outbound/clicks/{messageid}
func (a *API) GetOutboundMessageClicksByMessageID(messageID string, count, offset int) (*ListOutboundClicksResp, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(count))
	q.Set("offset", strconv.Itoa(offset))
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("messages/outbound/clicks/%s?%s", messageID, q.Encode()), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ListOutboundClicksResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// SearchInboundMessages searches inbound messages using the given parameters.
// GET /messages/inbound
func (a *API) SearchInboundMessages(params InboundMessageSearchParams) (*ListInboundMessagesResp, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(params.Count))
	q.Set("offset", strconv.Itoa(params.Offset))
	if params.Recipient != "" {
		q.Set("recipient", params.Recipient)
	}
	if params.FromEmail != "" {
		q.Set("fromemail", params.FromEmail)
	}
	if params.Tag != "" {
		q.Set("tag", params.Tag)
	}
	if params.Status != "" {
		q.Set("status", params.Status)
	}
	if params.FromDate != "" {
		q.Set("fromdate", params.FromDate)
	}
	if params.ToDate != "" {
		q.Set("todate", params.ToDate)
	}
	if params.Subject != "" {
		q.Set("subject", params.Subject)
	}
	if params.MailboxHash != "" {
		q.Set("mailboxhash", params.MailboxHash)
	}
	if params.MessageStream != "" {
		q.Set("messagestream", params.MessageStream)
	}

	req, err := a.newRequest(http.MethodGet, "messages/inbound?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ListInboundMessagesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetInboundMessageDetails retrieves the details for an inbound message.
// GET /messages/inbound/{messageid}/details
func (a *API) GetInboundMessageDetails(messageID string) (*InboundMessageDetailsResp, error) {
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("messages/inbound/%s/details", messageID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data InboundMessageDetailsResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// BypassInboundMessageRules bypasses the inbound rules for a specific inbound message.
// PUT /messages/inbound/{messageid}/bypass
func (a *API) BypassInboundMessageRules(messageID string) (*InboundBypassResp, error) {
	req, err := a.newRequest(http.MethodPut, fmt.Sprintf("messages/inbound/%s/bypass", messageID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data InboundBypassResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// RetryInboundMessage retries processing of a failed inbound message.
// PUT /messages/inbound/{messageid}/retry
func (a *API) RetryInboundMessage(messageID string) (*InboundRetryResp, error) {
	req, err := a.newRequest(http.MethodPut, fmt.Sprintf("messages/inbound/%s/retry", messageID), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data InboundRetryResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
