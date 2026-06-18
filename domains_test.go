package postmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// ---- ListDomains ---------------------------------------------------------------

func TestListDomains_Success(t *testing.T) {
	want := ListDomainsResp{
		TotalCount: 2,
		Domains: []DomainResp{
			{ID: 1, Name: "example.com"},
			{ID: 2, Name: "another.com"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		// Exact path check — avoids false positives from paths like /senders/domains.
		if req.URL.Path != "/domains" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if !strings.Contains(req.URL.RawQuery, "count=10") {
			t.Errorf("expected count param, query=%s", req.URL.RawQuery)
		}
		if !strings.Contains(req.URL.RawQuery, "offset=0") {
			t.Errorf("expected offset param, query=%s", req.URL.RawQuery)
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

// TestListDomains_InvalidCount verifies that passing count < 1 returns an
// error immediately without making an HTTP request.
func TestListDomains_InvalidCount(t *testing.T) {
	for _, count := range []int{0, -1, -100} {
		count := count
		t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				t.Errorf("HTTP request should not be made for count=%d", count)
				return nil, nil
			})))

			_, err := api.ListDomains(count, 0)
			if err == nil {
				t.Fatalf("expected an error for count=%d, got nil", count)
			}
		})
	}
}

func TestListDomains_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.ListDomains(10, 0)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected errors.As(err, &PostmarkErr) to be true, got err=%v", err)
	}
}

func TestListDomains_TransportError(t *testing.T) {
	transportErr := fmt.Errorf("dial tcp: connection refused")

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, transportErr
	})))

	_, err := api.ListDomains(10, 0)
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
}

// ---- GetDomain -----------------------------------------------------------------

func TestGetDomain_Success(t *testing.T) {
	want := DomainResp{
		ID:               42,
		Name:             "example.com",
		SPFVerified:      true,
		DKIMVerified:     true,
		ReturnPathDomain: "pm-bounces.example.com",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains/42") {
			t.Errorf("unexpected path: %s", req.URL.Path)
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
	if !got.SPFVerified {
		t.Error("expected SPFVerified to be true")
	}
}

func TestGetDomain_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.GetDomain(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestGetDomain_TransportError(t *testing.T) {
	transportErr := fmt.Errorf("dial tcp: connection refused")

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, transportErr
	})))

	_, err := api.GetDomain(42)
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
}

// ---- CreateDomain --------------------------------------------------------------

func TestCreateDomain_Success(t *testing.T) {
	want := DomainResp{
		ID:   10,
		Name: "newdomain.com",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		// Verify the request body is correctly serialised.
		var body CreateDomainReq
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Name != "newdomain.com" {
			t.Errorf("request body Name = %q, want newdomain.com", body.Name)
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

func TestCreateDomain_WithReturnPath(t *testing.T) {
	want := DomainResp{
		ID:               11,
		Name:             "customdomain.com",
		ReturnPathDomain: "pm-bounces.customdomain.com",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		// Verify both fields are serialised in the request body.
		var body CreateDomainReq
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Name != "customdomain.com" {
			t.Errorf("request body Name = %q, want customdomain.com", body.Name)
		}
		if body.ReturnPathDomain != "pm-bounces.customdomain.com" {
			t.Errorf("request body ReturnPathDomain = %q, want pm-bounces.customdomain.com", body.ReturnPathDomain)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.CreateDomain(&CreateDomainReq{
		Name:             "customdomain.com",
		ReturnPathDomain: "pm-bounces.customdomain.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ReturnPathDomain != "pm-bounces.customdomain.com" {
		t.Errorf("ReturnPathDomain = %q, want pm-bounces.customdomain.com", got.ReturnPathDomain)
	}
}

func TestCreateDomain_Conflict(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 505, Message: "A domain with this name already exists."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusConflict,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateDomain(&CreateDomainReq{Name: "duplicate.com"})
	if err == nil {
		t.Fatal("expected ErrExists, got nil")
	}
	if !errors.Is(err, ErrExists) {
		t.Errorf("expected errors.Is(err, ErrExists) to be true, got err=%v", err)
	}
}

func TestCreateDomain_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateDomain(&CreateDomainReq{Name: "bad.com"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var pe PostmarkErr
	if !errors.As(err, &pe) {
		t.Errorf("expected errors.As(err, &PostmarkErr) to be true, got err=%v", err)
	}
}

func TestCreateDomain_TransportError(t *testing.T) {
	transportErr := fmt.Errorf("dial tcp: connection refused")

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, transportErr
	})))

	_, err := api.CreateDomain(&CreateDomainReq{Name: "new.com"})
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
}

// ---- UpdateDomain --------------------------------------------------------------

func TestUpdateDomain_Success(t *testing.T) {
	want := DomainResp{
		ID:               7,
		Name:             "example.com",
		ReturnPathDomain: "new-bounces.example.com",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains/7") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		// Verify the request body is correctly serialised.
		var body UpdateDomainReq
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.ReturnPathDomain != "new-bounces.example.com" {
			t.Errorf("request body ReturnPathDomain = %q, want new-bounces.example.com", body.ReturnPathDomain)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.UpdateDomain(7, &UpdateDomainReq{ReturnPathDomain: "new-bounces.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ReturnPathDomain != "new-bounces.example.com" {
		t.Errorf("ReturnPathDomain = %q, want new-bounces.example.com", got.ReturnPathDomain)
	}
}

// TestUpdateDomain_EmptyReq verifies that calling UpdateDomain with a
// zero-value UpdateDomainReq returns an error immediately rather than
// silently sending an empty JSON object.
func TestUpdateDomain_EmptyReq(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not be made for an empty UpdateDomainReq")
		return nil, nil
	})))

	_, err := api.UpdateDomain(7, &UpdateDomainReq{})
	if err == nil {
		t.Fatal("expected an error for empty UpdateDomainReq, got nil")
	}
}

// TestUpdateDomain_NilReq verifies that passing a nil *UpdateDomainReq returns
// an error immediately rather than dereferencing a nil pointer.
func TestUpdateDomain_NilReq(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not be made for a nil UpdateDomainReq")
		return nil, nil
	})))

	_, err := api.UpdateDomain(7, nil)
	if err == nil {
		t.Fatal("expected an error for nil UpdateDomainReq, got nil")
	}
}

func TestUpdateDomain_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.UpdateDomain(9999, &UpdateDomainReq{ReturnPathDomain: "bounces.ghost.com"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestUpdateDomain_TransportError(t *testing.T) {
	transportErr := fmt.Errorf("dial tcp: connection refused")

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, transportErr
	})))

	_, err := api.UpdateDomain(7, &UpdateDomainReq{ReturnPathDomain: "bounces.example.com"})
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
}

// ---- DeleteDomain --------------------------------------------------------------

func TestDeleteDomain_Success(t *testing.T) {
	want := DeleteResp{Message: "Domain deleted."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains/99") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteDomain(99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Domain deleted." {
		t.Errorf("Message = %q, want 'Domain deleted.'", got.Message)
	}
}

func TestDeleteDomain_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.DeleteDomain(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestDeleteDomain_TransportError(t *testing.T) {
	transportErr := fmt.Errorf("dial tcp: connection refused")

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, transportErr
	})))

	_, err := api.DeleteDomain(99)
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
}

// ---- VerifyDomainDkim ----------------------------------------------------------

func TestVerifyDomainDkim_Success(t *testing.T) {
	want := DomainResp{
		ID:           42,
		Name:         "example.com",
		DKIMVerified: true,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains/42/verifyDkim") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.VerifyDomainDkim(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.DKIMVerified {
		t.Error("expected DKIMVerified to be true")
	}
}

func TestVerifyDomainDkim_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.VerifyDomainDkim(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestVerifyDomainDkim_TransportError(t *testing.T) {
	transportErr := fmt.Errorf("dial tcp: connection refused")

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, transportErr
	})))

	_, err := api.VerifyDomainDkim(42)
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
}

// ---- VerifyDomainReturnPath ----------------------------------------------------

func TestVerifyDomainReturnPath_Success(t *testing.T) {
	want := DomainResp{
		ID:                       42,
		Name:                     "example.com",
		ReturnPathDomainVerified: true,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains/42/verifyReturnPath") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.VerifyDomainReturnPath(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.ReturnPathDomainVerified {
		t.Error("expected ReturnPathDomainVerified to be true")
	}
}

func TestVerifyDomainReturnPath_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.VerifyDomainReturnPath(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestVerifyDomainReturnPath_TransportError(t *testing.T) {
	transportErr := fmt.Errorf("dial tcp: connection refused")

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, transportErr
	})))

	_, err := api.VerifyDomainReturnPath(42)
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
}

// ---- VerifyDomainSPF -----------------------------------------------------------

func TestVerifyDomainSPF_Success(t *testing.T) {
	want := DomainResp{
		ID:          42,
		Name:        "example.com",
		SPFVerified: true,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains/42/verifyspf") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.VerifyDomainSPF(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.SPFVerified {
		t.Error("expected SPFVerified to be true")
	}
}

func TestVerifyDomainSPF_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.VerifyDomainSPF(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestVerifyDomainSPF_TransportError(t *testing.T) {
	transportErr := fmt.Errorf("dial tcp: connection refused")

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, transportErr
	})))

	_, err := api.VerifyDomainSPF(42)
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
}

// ---- RotateDomainDKIM ----------------------------------------------------------

func TestRotateDomainDKIM_Success(t *testing.T) {
	want := DomainResp{
		ID:               42,
		Name:             "example.com",
		DKIMPendingHost:  "pm._domainkey.example.com",
		DKIMUpdateStatus: "Pending",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains/42/rotatedkim") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.RotateDomainDKIM(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DKIMUpdateStatus != "Pending" {
		t.Errorf("DKIMUpdateStatus = %q, want Pending", got.DKIMUpdateStatus)
	}
	if got.DKIMPendingHost != "pm._domainkey.example.com" {
		t.Errorf("DKIMPendingHost = %q, want pm._domainkey.example.com", got.DKIMPendingHost)
	}
}

func TestRotateDomainDKIM_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 0, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.RotateDomainDKIM(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestRotateDomainDKIM_TransportError(t *testing.T) {
	transportErr := fmt.Errorf("dial tcp: connection refused")

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, transportErr
	})))

	_, err := api.RotateDomainDKIM(42)
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
}

// ---- Account-Token header check ------------------------------------------------

func TestDomains_AccountTokenHeader(t *testing.T) {
	const token = "test-account-token"

	api := New(
		APITokenOpt(token),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("X-Postmark-Account-Token"); got != token {
				t.Errorf("X-Postmark-Account-Token = %q, want %q", got, token)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, DomainResp{ID: 1, Name: "example.com"}),
			}, nil
		})),
	)

	_, err := api.GetDomain(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
