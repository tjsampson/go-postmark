package postmark

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestListDomains_Success(t *testing.T) {
	want := ListDomainsResp{
		TotalCount: 2,
		Domains: []DomainResp{
			{ID: 1, Name: "example.com"},
			{ID: 2, Name: "other.com"},
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

func TestListDomains_Error(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

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

func TestCreateDomain_Success(t *testing.T) {
	want := DomainResp{ID: 42, Name: "newdomain.com"}

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
	if got.ID != 42 || got.Name != "newdomain.com" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCreateDomain_Error(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 422, Message: "invalid domain"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateDomain(&CreateDomainReq{Name: ""})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestGetDomain_Success(t *testing.T) {
	want := DomainResp{ID: 7, Name: "test.com", SPFVerified: true}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains/7") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetDomain(7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 7 || got.Name != "test.com" {
		t.Errorf("got %+v, want %+v", got, want)
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
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

func TestUpdateDomain_Success(t *testing.T) {
	want := DomainResp{ID: 7, Name: "test.com", ReturnPathDomain: "pm.test.com"}

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

	got, err := api.UpdateDomain(7, &UpdateDomainReq{ReturnPathDomain: "pm.test.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ReturnPathDomain != "pm.test.com" {
		t.Errorf("ReturnPathDomain = %q, want pm.test.com", got.ReturnPathDomain)
	}
}

func TestUpdateDomain_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Domain not found"}),
		}, nil
	})))

	_, err := api.UpdateDomain(9999, &UpdateDomainReq{ReturnPathDomain: "pm.test.com"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

func TestDeleteDomain_Success(t *testing.T) {
	want := DeleteResp{Message: "Domain deleted."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/domains/5") {
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
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

// Table-driven tests for domain verification helpers.
func TestDomainVerificationHelpers(t *testing.T) {
	domainResp := DomainResp{ID: 10, Name: "example.com", DKIMVerified: true}

	tests := []struct {
		name       string
		method     string
		pathSuffix string
		call       func(api *API) (*DomainResp, error)
	}{
		{
			name:       "VerifyDomainDKIM",
			method:     http.MethodPut,
			pathSuffix: "/domains/10/verifyDkim",
			call:       func(api *API) (*DomainResp, error) { return api.VerifyDomainDKIM(10) },
		},
		{
			name:       "VerifyDomainReturnPath",
			method:     http.MethodPut,
			pathSuffix: "/domains/10/verifyReturnPath",
			call:       func(api *API) (*DomainResp, error) { return api.VerifyDomainReturnPath(10) },
		},
		{
			name:       "VerifyDomainSPF",
			method:     http.MethodPost,
			pathSuffix: "/domains/10/verifyspf",
			call:       func(api *API) (*DomainResp, error) { return api.VerifyDomainSPF(10) },
		},
		{
			name:       "RotateDomainDKIM",
			method:     http.MethodPut,
			pathSuffix: "/domains/10/rotatedkim",
			call:       func(api *API) (*DomainResp, error) { return api.RotateDomainDKIM(10) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name+"_Success", func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				if req.Method != tc.method {
					t.Errorf("expected %s, got %s", tc.method, req.Method)
				}
				if !strings.HasSuffix(req.URL.Path, tc.pathSuffix) {
					t.Errorf("unexpected path: %s (want suffix %s)", req.URL.Path, tc.pathSuffix)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, domainResp),
				}, nil
			})))

			got, err := tc.call(api)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != domainResp.ID {
				t.Errorf("ID = %d, want %d", got.ID, domainResp.ID)
			}
		})

		t.Run(tc.name+"_Error", func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "not found"}),
				}, nil
			})))

			_, err := tc.call(api)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
			}
		})
	}
}
