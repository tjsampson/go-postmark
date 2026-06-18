package postmark

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestSenderSignaturesCRUD exercises ListSenderSignatures, CreateSenderSignature,
// GetSenderSignature, UpdateSenderSignature, and DeleteSenderSignature using
// table-driven success and error sub-cases.
func TestSenderSignaturesCRUD(t *testing.T) {
	sigResp := SenderSignatureResp{ID: 55, EmailAddress: "sig@example.com", Name: "Sig"}
	listResp := ListSenderSignaturesResp{
		TotalCount: 2,
		SenderSignatures: []SenderSignatureResp{
			{ID: 1, EmailAddress: "sender1@example.com", Name: "Sender One"},
			{ID: 2, EmailAddress: "sender2@example.com", Name: "Sender Two"},
		},
	}
	deleteResp := DeleteResp{Message: "Signature removed."}

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
			name:           "ListSenderSignatures/success",
			wantMethod:     http.MethodGet,
			wantPathSuffix: "/senders",
			statusCode:     http.StatusOK,
			responseBody:   listResp,
			call:           func(api *API) (interface{}, error) { return api.ListSenderSignatures(10, 0) },
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*ListSenderSignaturesResp)
				if r.TotalCount != 2 {
					t.Errorf("TotalCount = %d, want 2", r.TotalCount)
				}
				if len(r.SenderSignatures) != 2 {
					t.Errorf("len(SenderSignatures) = %d, want 2", len(r.SenderSignatures))
				}
			},
		},
		{
			name:         "ListSenderSignatures/error",
			statusCode:   http.StatusInternalServerError,
			responseBody: PostmarkErr{ErrorCode: 500, Message: "internal error"},
			call:         func(api *API) (interface{}, error) { return api.ListSenderSignatures(10, 0) },
		},
		{
			name:           "CreateSenderSignature/success",
			wantMethod:     http.MethodPost,
			wantPathSuffix: "/senders",
			statusCode:     http.StatusOK,
			responseBody:   SenderSignatureResp{ID: 99, EmailAddress: "new@example.com", Name: "New Sender"},
			call: func(api *API) (interface{}, error) {
				return api.CreateSenderSignature(&CreateSenderSignatureReq{
					FromEmail: "new@example.com",
					Name:      "New Sender",
				})
			},
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*SenderSignatureResp)
				if r.ID != 99 || r.EmailAddress != "new@example.com" {
					t.Errorf("got ID=%d EmailAddress=%s", r.ID, r.EmailAddress)
				}
			},
		},
		{
			name:         "CreateSenderSignature/error",
			statusCode:   http.StatusUnprocessableEntity,
			responseBody: PostmarkErr{ErrorCode: 422, Message: "invalid email"},
			call: func(api *API) (interface{}, error) {
				return api.CreateSenderSignature(&CreateSenderSignatureReq{FromEmail: "bad"})
			},
		},
		{
			name:           "GetSenderSignature/success",
			wantMethod:     http.MethodGet,
			wantPathSuffix: "/senders/55",
			statusCode:     http.StatusOK,
			responseBody:   sigResp,
			call:           func(api *API) (interface{}, error) { return api.GetSenderSignature(55) },
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*SenderSignatureResp)
				if r.ID != 55 {
					t.Errorf("ID = %d, want 55", r.ID)
				}
			},
		},
		{
			name:         "GetSenderSignature/not_found",
			statusCode:   http.StatusNotFound,
			responseBody: PostmarkErr{ErrorCode: 404, Message: "Sender signature not found"},
			call:         func(api *API) (interface{}, error) { return api.GetSenderSignature(9999) },
		},
		{
			name:           "UpdateSenderSignature/success",
			wantMethod:     http.MethodPut,
			wantPathSuffix: "/senders/55",
			statusCode:     http.StatusOK,
			responseBody:   SenderSignatureResp{ID: 55, EmailAddress: "sig@example.com", Name: "Updated Sig"},
			call: func(api *API) (interface{}, error) {
				return api.UpdateSenderSignature(55, &UpdateSenderSignatureReq{Name: "Updated Sig"})
			},
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*SenderSignatureResp)
				if r.Name != "Updated Sig" {
					t.Errorf("Name = %q, want Updated Sig", r.Name)
				}
			},
		},
		{
			name:         "UpdateSenderSignature/not_found",
			statusCode:   http.StatusNotFound,
			responseBody: PostmarkErr{ErrorCode: 404, Message: "Sender signature not found"},
			call: func(api *API) (interface{}, error) {
				return api.UpdateSenderSignature(9999, &UpdateSenderSignatureReq{Name: "Ghost"})
			},
		},
		{
			name:           "DeleteSenderSignature/success",
			wantMethod:     http.MethodDelete,
			wantPathSuffix: "/senders/33",
			statusCode:     http.StatusOK,
			responseBody:   deleteResp,
			call:           func(api *API) (interface{}, error) { return api.DeleteSenderSignature(33) },
			checkOK: func(t *testing.T, got interface{}) {
				r := got.(*DeleteResp)
				if r.Message != "Signature removed." {
					t.Errorf("Message = %q", r.Message)
				}
			},
		},
		{
			name:         "DeleteSenderSignature/not_found",
			statusCode:   http.StatusNotFound,
			responseBody: PostmarkErr{ErrorCode: 404, Message: "Sender signature not found"},
			call:         func(api *API) (interface{}, error) { return api.DeleteSenderSignature(9999) },
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

// TestSenderSignatureHelpers exercises ResendSenderSignatureConfirmation,
// VerifySenderSignatureSPF, and RequestNewDKIMForSenderSignature using
// table-driven success and error sub-cases.
func TestSenderSignatureHelpers(t *testing.T) {
	sigResp := SenderSignatureResp{ID: 10, EmailAddress: "helper@example.com", SPFVerified: true}
	resendResp := DeleteResp{Message: "confirmation sent"}

	tests := []struct {
		name           string
		method         string
		pathSuffix     string
		returnsDelResp bool
		call           func(api *API) (interface{}, error)
	}{
		{
			name:           "ResendSenderSignatureConfirmation",
			method:         http.MethodPost,
			pathSuffix:     "/senders/10/resend",
			returnsDelResp: true,
			call: func(api *API) (interface{}, error) {
				return api.ResendSenderSignatureConfirmation(10)
			},
		},
		{
			name:       "VerifySenderSignatureSPF",
			method:     http.MethodPost,
			pathSuffix: "/senders/10/verifyspf",
			call: func(api *API) (interface{}, error) {
				return api.VerifySenderSignatureSPF(10)
			},
		},
		{
			name:       "RequestNewDKIMForSenderSignature",
			method:     http.MethodPost,
			pathSuffix: "/senders/10/requestnewdkim",
			call: func(api *API) (interface{}, error) {
				return api.RequestNewDKIMForSenderSignature(10)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name+"/success", func(t *testing.T) {
			var respBody interface{} = sigResp
			if tc.returnsDelResp {
				respBody = resendResp
			}

			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				if req.Method != tc.method {
					t.Errorf("method = %s, want %s", req.Method, tc.method)
				}
				if !strings.HasSuffix(req.URL.Path, tc.pathSuffix) {
					t.Errorf("path = %s, want suffix %s", req.URL.Path, tc.pathSuffix)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, respBody),
				}, nil
			})))

			got, err := tc.call(api)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Error("expected non-nil response")
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

// TestSenderSignatures_UnmarshalError verifies that a malformed JSON response
// causes sender signature methods to return a non-nil error.
func TestSenderSignatures_UnmarshalError(t *testing.T) {
	tests := []struct {
		name string
		call func(api *API) (interface{}, error)
	}{
		{
			name: "ListSenderSignatures",
			call: func(api *API) (interface{}, error) { return api.ListSenderSignatures(10, 0) },
		},
		{
			name: "CreateSenderSignature",
			call: func(api *API) (interface{}, error) {
				return api.CreateSenderSignature(&CreateSenderSignatureReq{FromEmail: "x@x.com", Name: "X"})
			},
		},
		{
			name: "GetSenderSignature",
			call: func(api *API) (interface{}, error) { return api.GetSenderSignature(1) },
		},
		{
			name: "UpdateSenderSignature",
			call: func(api *API) (interface{}, error) {
				return api.UpdateSenderSignature(1, &UpdateSenderSignatureReq{Name: "Y"})
			},
		},
		{
			name: "DeleteSenderSignature",
			call: func(api *API) (interface{}, error) { return api.DeleteSenderSignature(1) },
		},
		{
			name: "ResendSenderSignatureConfirmation",
			call: func(api *API) (interface{}, error) { return api.ResendSenderSignatureConfirmation(1) },
		},
		{
			name: "VerifySenderSignatureSPF",
			call: func(api *API) (interface{}, error) { return api.VerifySenderSignatureSPF(1) },
		},
		{
			name: "RequestNewDKIMForSenderSignature",
			call: func(api *API) (interface{}, error) { return api.RequestNewDKIMForSenderSignature(1) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
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
