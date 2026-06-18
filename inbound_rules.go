package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type (
	// InboundRuleResp represents a single inbound rule trigger.
	InboundRuleResp struct {
		ID   int64  `json:"ID"`
		Rule string `json:"Rule"`
	}

	// ListInboundRulesResp is the response for listing inbound rules.
	ListInboundRulesResp struct {
		TotalCount    int               `json:"TotalCount"`
		InboundRules  []InboundRuleResp `json:"InboundRules"`
	}

	// createInboundRuleReq is the request body for creating an inbound rule.
	createInboundRuleReq struct {
		Rule string `json:"Rule"`
	}
)

// ListInboundRules returns a paginated list of inbound rules.
// GET /triggers/inboundrules
func (a *API) ListInboundRules(count, offset int) (*ListInboundRulesResp, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(count))
	q.Set("offset", strconv.Itoa(offset))
	req, err := a.newRequest(http.MethodGet, "triggers/inboundrules?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data ListInboundRulesResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateInboundRule creates a new inbound rule trigger.
// POST /triggers/inboundrules
func (a *API) CreateInboundRule(rule string) (*InboundRuleResp, error) {
	body := &createInboundRuleReq{Rule: rule}
	req, err := a.newRequest(http.MethodPost, "triggers/inboundrules", body)
	if err != nil {
		return nil, err
	}
	resp, e := a.Do(req)
	if e != nil {
		return nil, e
	}
	var data InboundRuleResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteInboundRule deletes an inbound rule trigger by its ID.
// DELETE /triggers/inboundrules/{triggerid}
func (a *API) DeleteInboundRule(triggerID int64) (*DeleteResp, error) {
	req, err := a.newRequest(http.MethodDelete, fmt.Sprintf("triggers/inboundrules/%d", triggerID), nil)
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
