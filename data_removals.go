package postmark

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type (
	// DataRemovalReq is the request body for creating a data removal request.
	DataRemovalReq struct {
		EmailAddress string `json:"EmailAddress"`
		RequestedBy  string `json:"RequestedBy"`
	}

	// DataRemovalResp represents a data removal record returned by the API.
	DataRemovalResp struct {
		ID           int64  `json:"ID"`
		EmailAddress string `json:"EmailAddress"`
		Status       string `json:"Status"`
		RequestedAt  string `json:"RequestedAt"`
		RequestedBy  string `json:"RequestedBy"`
	}
)

// RequestDataRemoval submits a new data removal request.
// POST /data-removals
func (a *API) RequestDataRemoval(req *DataRemovalReq) (*DataRemovalResp, error) {
	httpReq, err := a.newRequest(http.MethodPost, "data-removals", req)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(httpReq)
	if err != nil {
		return nil, err
	}
	var data DataRemovalResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetDataRemoval retrieves an existing data removal request by its ID.
// GET /data-removals/{id}
func (a *API) GetDataRemoval(removalID int64) (*DataRemovalResp, error) {
	req, err := a.newRequest(http.MethodGet, fmt.Sprintf("data-removals/%d", removalID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Do(req)
	if err != nil {
		return nil, err
	}
	var data DataRemovalResp
	if err = json.Unmarshal(resp.rawBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
