package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type (
	CreateServerReq struct {
		Name             string `json:"Name"`
		Color            string `json:"Color"`
		SmtpApiActivated bool   `json:"SmtpApiActivated"`
	}

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

	ListServerResp struct {
		TotalCount int          `json:"TotalCount"`
		Servers    []ServerResp `json:"Servers"`
	}

	DeleteResp struct {
		ErrorCode int    `json:"ErrorCode"`
		Message   string `json:"Message"`
	}
)

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

func (a *API) ListServers(count, offset string) (*ListServerResp, error) {
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("servers?count=%s&offset=%s", count, offset), nil)
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
