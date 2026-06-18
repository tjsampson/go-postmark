package postmark

import (
	"errors"
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
		if !strings.HasSuffix(req.URL.Path, "/domains") {
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
