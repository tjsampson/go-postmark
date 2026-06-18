package postmark

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// ---- ListDomains ---------------------------------------------------------------

func TestListDomains_Success(t *testing.T) {
	want := ListDomainsResp{
		TotalCount: 2,
		Domains: []DomainListEntry{
			{ID: 1, Name: "example.com"},
			{ID: 2, Name: "example.net"},
		},
	}

	api := New(APITokenOpt("acct-tok"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if !strings.Contains(req.URL.RawQuery, "count=10") {
			t.Errorf("expected count param, query=%s", req.URL.RawQuery)
		}
		if !strings.Contains(req.URL.RawQuery, "offset=0") {
			t.Errorf("expected offset param, query=%s", req.URL.RawQuery)
		}
		if got := req.Header.Get("X-Postmark-Account-Token"); got != "acct-tok" {
			t.Errorf("X-Postmark-Account-Token = %q, want acct-tok", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListDomains(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
	if len(got.Domains) != 2 {
		t.Errorf("len(Domains) = %d, want 2", len(got.Domains))
	}
}

func TestListDomains_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.ListDomains(10, 0)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetDomain ----------------------------------------------------------------

func TestGetDomain_Success(t *testing.T) {
	want := DomainResp{ID: 42, Name: "example.com", DKIMVerified: true}

	api := New(APITokenOpt("acct-tok"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/domains/42") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if got := req.Header.Get("X-Postmark-Account-Token"); got != "acct-tok" {
			t.Errorf("X-Postmark-Account-Token = %q, want acct-tok", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetDomain(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("ID = %d, want 42", got.ID)
	}
	if got.Name != "example.com" {
		t.Errorf("Name = %q, want example.com", got.Name)
	}
}

func TestGetDomain_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.GetDomain(9999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- CreateDomain -------------------------------------------------------------

func TestCreateDomain_Success(t *testing.T) {
	want := DomainResp{ID: 10, Name: "newdomain.com"}

	api := New(APITokenOpt("acct-tok"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if got := req.Header.Get("X-Postmark-Account-Token"); got != "acct-tok" {
			t.Errorf("X-Postmark-Account-Token = %q, want acct-tok", got)
		}
		// Verify the request body contains the expected fields.
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		var sent CreateDomainReq
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if sent.Name != "newdomain.com" {
			t.Errorf("request body Name = %q, want newdomain.com", sent.Name)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.CreateDomain(&CreateDomainReq{Name: "newdomain.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 10 {
		t.Errorf("ID = %d, want 10", got.ID)
	}
	if got.Name != "newdomain.com" {
		t.Errorf("Name = %q, want newdomain.com", got.Name)
	}
}

func TestCreateDomain_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.CreateDomain(&CreateDomainReq{Name: "bad"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- EditDomain ---------------------------------------------------------------

func TestEditDomain_Success(t *testing.T) {
	want := DomainResp{ID: 7, Name: "example.com", ReturnPathDomain: "pm-bounces.example.com"}

	api := New(APITokenOpt("acct-tok"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/domains/7") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if got := req.Header.Get("X-Postmark-Account-Token"); got != "acct-tok" {
			t.Errorf("X-Postmark-Account-Token = %q, want acct-tok", got)
		}
		// Verify the request body contains the expected fields.
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		var sent EditDomainReq
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if sent.ReturnPathDomain != "pm-bounces.example.com" {
			t.Errorf("request body ReturnPathDomain = %q, want pm-bounces.example.com", sent.ReturnPathDomain)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.EditDomain(7, &EditDomainReq{ReturnPathDomain: "pm-bounces.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ReturnPathDomain != "pm-bounces.example.com" {
		t.Errorf("ReturnPathDomain = %q", got.ReturnPathDomain)
	}
}

func TestEditDomain_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.EditDomain(9999, &EditDomainReq{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- DeleteDomain -------------------------------------------------------------

func TestDeleteDomain_Success(t *testing.T) {
	want := DeleteResp{Message: "Domain deleted."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/domains/5") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteDomain(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Domain deleted." {
		t.Errorf("Message = %q", got.Message)
	}
}

func TestDeleteDomain_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.DeleteDomain(9999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- VerifyDomainDKIM ---------------------------------------------------------

func TestVerifyDomainDKIM_Success(t *testing.T) {
	want := DomainResp{ID: 3, Name: "example.com", DKIMVerified: true}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/domains/3/verifyDkim") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.VerifyDomainDKIM(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.DKIMVerified {
		t.Errorf("DKIMVerified = false, want true")
	}
}

func TestVerifyDomainDKIM_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.VerifyDomainDKIM(3)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- VerifyDomainReturnPath ---------------------------------------------------

func TestVerifyDomainReturnPath_Success(t *testing.T) {
	want := DomainResp{ID: 4, Name: "example.com", ReturnPathDomainVerified: true}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/domains/4/verifyReturnPath") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.VerifyDomainReturnPath(4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.ReturnPathDomainVerified {
		t.Errorf("ReturnPathDomainVerified = false, want true")
	}
}

func TestVerifyDomainReturnPath_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.VerifyDomainReturnPath(4)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- RotateDomainDKIM ---------------------------------------------------------

func TestRotateDomainDKIM_Success(t *testing.T) {
	want := DomainResp{ID: 6, Name: "example.com", DKIMPendingHost: "new-dkim._domainkey.example.com"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/domains/6/rotateDkim") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.RotateDomainDKIM(6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DKIMPendingHost != "new-dkim._domainkey.example.com" {
		t.Errorf("DKIMPendingHost = %q", got.DKIMPendingHost)
	}
}

func TestRotateDomainDKIM_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.RotateDomainDKIM(6)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
