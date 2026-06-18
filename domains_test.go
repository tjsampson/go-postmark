package postmark

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestDomainsCRUD exercises ListDomains, CreateDomain, GetDomain, UpdateDomain,
// and DeleteDomain using table-driven success and error sub-cases.
func TestDomainsCRUD(t *testing.T) {
	domainResp := DomainResp{ID: 7, Name: "test.com", SPFVerified: true, ReturnPathDomain: "pm.test.com"}
	listResp := ListDomainsResp{
		TotalCount: 2,
		Domains:    []DomainResp{{ID: 1, Name: "example.com"}, {ID: 2, Name: "other.com"}},
	}
	deleteResp := DeleteResp{Message: "Domain deleted."}
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	tests := []struct {
		name           string
		wantMethod     string
		wantPathSuffix string
		statusCode     int
		responseBody   interface{}
		call           func(api *API) (interface{}, error)
		checkOK        func(t *testing.T, got interface{})
	}{
		{
			name:           "ListDomains/success",
			wantMethod:     http.MethodGet,
			wantPathSuffix: "/domains",
			statusCode:     http.StatusOK,
			responseBody:   listResp,
			call:           func(api *API) (interface{}, error) { return api.ListDomains(10, 0) },
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*ListDomainsResp)
				if r.TotalCount != 2 {
					t.Errorf("TotalCount = %d, want 2", r.TotalCount)
				}
				if len(r.Domains) != 2 {
					t.Errorf("len(Domains) = %d, want 2", len(r.Domains))
				}
			},
		},
		{
			name:         "ListDomains/error",
			statusCode:   http.StatusInternalServerError,
			responseBody: pmErr,
			call:         func(api *API) (interface{}, error) { return api.ListDomains(10, 0) },
		},
		{
			name:           "CreateDomain/success",
			wantMethod:     http.MethodPost,
			wantPathSuffix: "/domains",
			statusCode:     http.StatusOK,
			responseBody:   domainResp,
			call: func(api *API) (interface{}, error) {
				return api.CreateDomain(&CreateDomainReq{Name: "test.com"})
			},
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*DomainResp)
				if r.ID != 7 || r.Name != "test.com" {
					t.Errorf("got %+v, want ID=7 Name=test.com", r)
				}
			},
		},
		{
			name:         "CreateDomain/error",
			statusCode:   http.StatusUnprocessableEntity,
			responseBody: PostmarkErr{ErrorCode: 422, Message: "invalid domain"},
			call: func(api *API) (interface{}, error) {
				return api.CreateDomain(&CreateDomainReq{Name: ""})
			},
		},
		{
			name:           "GetDomain/success",
			wantMethod:     http.MethodGet,
			wantPathSuffix: "/domains/7",
			statusCode:     http.StatusOK,
			responseBody:   domainResp,
			call:           func(api *API) (interface{}, error) { return api.GetDomain(7) },
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*DomainResp)
				if r.ID != 7 || r.Name != "test.com" {
					t.Errorf("got %+v, want ID=7 Name=test.com", r)
				}
			},
		},
		{
			name:           "GetDomain/not_found",
			wantPathSuffix: "/domains/9999",
			statusCode:     http.StatusNotFound,
			responseBody:   PostmarkErr{ErrorCode: 404, Message: "Domain not found"},
			call:           func(api *API) (interface{}, error) { return api.GetDomain(9999) },
		},
		{
			name:           "UpdateDomain/success",
			wantMethod:     http.MethodPut,
			wantPathSuffix: "/domains/7",
			statusCode:     http.StatusOK,
			responseBody:   domainResp,
			call: func(api *API) (interface{}, error) {
				return api.UpdateDomain(7, &UpdateDomainReq{ReturnPathDomain: "pm.test.com"})
			},
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*DomainResp)
				if r.ReturnPathDomain != "pm.test.com" {
					t.Errorf("ReturnPathDomain = %q, want pm.test.com", r.ReturnPathDomain)
				}
			},
		},
		{
			name:         "UpdateDomain/not_found",
			statusCode:   http.StatusNotFound,
			responseBody: PostmarkErr{ErrorCode: 404, Message: "Domain not found"},
			call: func(api *API) (interface{}, error) {
				return api.UpdateDomain(9999, &UpdateDomainReq{ReturnPathDomain: "pm.test.com"})
			},
		},
		{
			name:           "DeleteDomain/success",
			wantMethod:     http.MethodDelete,
			wantPathSuffix: "/domains/5",
			statusCode:     http.StatusOK,
			responseBody:   deleteResp,
			call:           func(api *API) (interface{}, error) { return api.DeleteDomain(5) },
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*DeleteResp)
				if r.Message != "Domain deleted." {
					t.Errorf("Message = %q, want Domain deleted.", r.Message)
				}
			},
		},
		{
			name:         "DeleteDomain/not_found",
			statusCode:   http.StatusNotFound,
			responseBody: PostmarkErr{ErrorCode: 404, Message: "Domain not found"},
			call:         func(api *API) (interface{}, error) { return api.DeleteDomain(9999) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isError := tc.statusCode >= http.StatusBadRequest

			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				if tc.wantMethod != "" && req.Method != tc.wantMethod {
					t.Errorf("method = %s, want %s", req.Method, tc.wantMethod)
				}
				if tc.wantPathSuffix != "" && !strings.HasSuffix(req.URL.Path, tc.wantPathSuffix) {
					t.Errorf("path = %s, want suffix %s", req.URL.Path, tc.wantPathSuffix)
				}
				return &http.Response{
					StatusCode: tc.statusCode,
					Body:       jsonBody(t, tc.responseBody),
				}, nil
			})))

			got, err := tc.call(api)
			if isError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.statusCode == http.StatusNotFound && !errors.Is(err, ErrNotFound) {
					t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.checkOK != nil {
					tc.checkOK(t, got)
				}
			}
		})
	}
}

// TestDomainVerificationHelpers exercises the four verification/rotation
// helpers using table-driven success and error sub-cases.
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
		t.Run(tc.name+"/success", func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				if req.Method != tc.method {
					t.Errorf("method = %s, want %s", req.Method, tc.method)
				}
				if !strings.HasSuffix(req.URL.Path, tc.pathSuffix) {
					t.Errorf("path = %s, want suffix %s", req.URL.Path, tc.pathSuffix)
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

		t.Run(tc.name+"/not_found", func(t *testing.T) {
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

// TestDomains_UnmarshalError verifies that a malformed JSON response body
// causes the domain methods to return a non-nil error rather than silently
// succeeding with a zero-value struct.
func TestDomains_UnmarshalError(t *testing.T) {
	malformed := io.NopCloser(strings.NewReader(`{not valid json`))

	tests := []struct {
		name string
		call func(api *API) (interface{}, error)
	}{
		{
			name: "ListDomains",
			call: func(api *API) (interface{}, error) { return api.ListDomains(10, 0) },
		},
		{
			name: "CreateDomain",
			call: func(api *API) (interface{}, error) {
				return api.CreateDomain(&CreateDomainReq{Name: "x.com"})
			},
		},
		{
			name: "GetDomain",
			call: func(api *API) (interface{}, error) { return api.GetDomain(1) },
		},
		{
			name: "UpdateDomain",
			call: func(api *API) (interface{}, error) {
				return api.UpdateDomain(1, &UpdateDomainReq{ReturnPathDomain: "pm.x.com"})
			},
		},
		{
			name: "DeleteDomain",
			call: func(api *API) (interface{}, error) { return api.DeleteDomain(1) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Each sub-test needs its own fresh ReadCloser because reading is destructive.
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				_ = malformed // referenced to avoid lint; each test gets the same bad body
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{not valid json`)),
				}, nil
			})))

			_, err := tc.call(api)
			if err == nil {
				t.Fatal("expected unmarshal error, got nil")
			}
		})
	}
}

// TestDomains_InputValidation verifies that non-positive IDs and counts are
// rejected locally before any HTTP request is made.
func TestDomains_InputValidation(t *testing.T) {
	// neverCalled panics if the HTTP transport is invoked — confirming that
	// the guard returned before building a request.
	neverCalled := newTestClient(func(req *http.Request) (*http.Response, error) {
		panic("HTTP client must not be called for invalid inputs")
	})

	api := New(HTTPClientOpt(neverCalled))

	t.Run("ListDomains/zero_count", func(t *testing.T) {
		_, err := api.ListDomains(0, 0)
		if err == nil {
			t.Fatal("expected error for count=0, got nil")
		}
	})

	t.Run("ListDomains/negative_count", func(t *testing.T) {
		_, err := api.ListDomains(-1, 0)
		if err == nil {
			t.Fatal("expected error for count=-1, got nil")
		}
	})

	t.Run("GetDomain/zero_id", func(t *testing.T) {
		_, err := api.GetDomain(0)
		if err == nil {
			t.Fatal("expected error for domainID=0, got nil")
		}
	})

	t.Run("GetDomain/negative_id", func(t *testing.T) {
		_, err := api.GetDomain(-5)
		if err == nil {
			t.Fatal("expected error for domainID=-5, got nil")
		}
	})

	t.Run("UpdateDomain/zero_id", func(t *testing.T) {
		_, err := api.UpdateDomain(0, &UpdateDomainReq{ReturnPathDomain: "pm.x.com"})
		if err == nil {
			t.Fatal("expected error for domainID=0, got nil")
		}
	})

	t.Run("DeleteDomain/zero_id", func(t *testing.T) {
		_, err := api.DeleteDomain(0)
		if err == nil {
			t.Fatal("expected error for domainID=0, got nil")
		}
	})

	t.Run("VerifyDomainDKIM/zero_id", func(t *testing.T) {
		_, err := api.VerifyDomainDKIM(0)
		if err == nil {
			t.Fatal("expected error for domainID=0, got nil")
		}
	})

	t.Run("VerifyDomainReturnPath/zero_id", func(t *testing.T) {
		_, err := api.VerifyDomainReturnPath(0)
		if err == nil {
			t.Fatal("expected error for domainID=0, got nil")
		}
	})

	t.Run("VerifyDomainSPF/zero_id", func(t *testing.T) {
		_, err := api.VerifyDomainSPF(0)
		if err == nil {
			t.Fatal("expected error for domainID=0, got nil")
		}
	})

	t.Run("RotateDomainDKIM/zero_id", func(t *testing.T) {
		_, err := api.RotateDomainDKIM(0)
		if err == nil {
			t.Fatal("expected error for domainID=0, got nil")
		}
	})
}
