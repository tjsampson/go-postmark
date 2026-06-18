package postmark

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestListSenderSignatures_Success(t *testing.T) {
	want := ListSenderSignaturesResp{
		TotalCount: 2,
		SenderSignatures: []SenderSignatureResp{
			{ID: 1, EmailAddress: "sender1@example.com", Name: "Sender One"},
			{ID: 2, EmailAddress: "sender2@example.com", Name: "Sender Two"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/senders") {
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

	got, err := api.ListSenderSignatures(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
	if len(got.SenderSignatures) != 2 {
		t.Errorf("len(SenderSignatures) = %d, want 2", len(got.SenderSignatures))
	}
}

func TestListSenderSignatures_Error(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.ListSenderSignatures(10, 0)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestCreateSenderSignature_Success(t *testing.T) {
	want := SenderSignatureResp{ID: 99, EmailAddress: "new@example.com", Name: "New Sender"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/senders") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.CreateSenderSignature(&CreateSenderSignatureReq{
		FromEmail: "new@example.com",
		Name:      "New Sender",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 99 || got.EmailAddress != "new@example.com" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCreateSenderSignature_Error(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 422, Message: "invalid email"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateSenderSignature(&CreateSenderSignatureReq{FromEmail: "bad"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestGetSenderSignature_Success(t *testing.T) {
	want := SenderSignatureResp{ID: 55, EmailAddress: "sig@example.com", Name: "Sig"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/senders/55") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetSenderSignature(55)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 55 {
		t.Errorf("ID = %d, want 55", got.ID)
	}
}

func TestGetSenderSignature_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Sender signature not found"}),
		}, nil
	})))

	_, err := api.GetSenderSignature(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

func TestUpdateSenderSignature_Success(t *testing.T) {
	want := SenderSignatureResp{ID: 55, EmailAddress: "sig@example.com", Name: "Updated Sig"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/senders/55") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.UpdateSenderSignature(55, &UpdateSenderSignatureReq{Name: "Updated Sig"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Updated Sig" {
		t.Errorf("Name = %q, want Updated Sig", got.Name)
	}
}

func TestUpdateSenderSignature_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Sender signature not found"}),
		}, nil
	})))

	_, err := api.UpdateSenderSignature(9999, &UpdateSenderSignatureReq{Name: "Ghost"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

func TestDeleteSenderSignature_Success(t *testing.T) {
	want := DeleteResp{Message: "Signature removed."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/senders/33") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteSenderSignature(33)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Signature removed." {
		t.Errorf("Message = %q", got.Message)
	}
}

func TestDeleteSenderSignature_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Sender signature not found"}),
		}, nil
	})))

	_, err := api.DeleteSenderSignature(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound), got %v", err)
	}
}

// Table-driven tests for sender signature helper operations.
func TestSenderSignatureHelpers(t *testing.T) {
	sigResp := SenderSignatureResp{ID: 10, EmailAddress: "helper@example.com", SPFVerified: true}
	deleteRespVal := DeleteResp{Message: "confirmation sent"}

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
		t.Run(tc.name+"_Success", func(t *testing.T) {
			var respBody interface{} = sigResp
			if tc.returnsDelResp {
				respBody = deleteRespVal
			}

			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				if req.Method != tc.method {
					t.Errorf("expected %s, got %s", tc.method, req.Method)
				}
				if !strings.HasSuffix(req.URL.Path, tc.pathSuffix) {
					t.Errorf("unexpected path: %s (want suffix %s)", req.URL.Path, tc.pathSuffix)
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
