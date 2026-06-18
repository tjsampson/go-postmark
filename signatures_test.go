package postmark

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// ---- ListSignatures -----------------------------------------------------------

func TestListSignatures_Success(t *testing.T) {
	want := ListSignaturesResp{
		TotalCount: 2,
		SenderSignatures: []SignatureListEntry{
			{ID: 1, EmailAddress: "sender@example.com", Name: "Sender One"},
			{ID: 2, EmailAddress: "sender2@example.com", Name: "Sender Two"},
		},
	}

	api := New(APITokenOpt("acct-tok"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
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
		if got := req.Header.Get("X-Postmark-Account-Token"); got != "acct-tok" {
			t.Errorf("X-Postmark-Account-Token = %q, want acct-tok", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListSignatures(10, 0)
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

func TestListSignatures_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.ListSignatures(10, 0)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetSignature -------------------------------------------------------------

func TestGetSignature_Success(t *testing.T) {
	want := SignatureResp{
		ID:           42,
		EmailAddress: "hello@example.com",
		Name:         "Hello Sender",
		Confirmed:    true,
	}

	api := New(APITokenOpt("acct-tok"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/senders/42") {
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

	got, err := api.GetSignature(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("ID = %d, want 42", got.ID)
	}
	if got.EmailAddress != "hello@example.com" {
		t.Errorf("EmailAddress = %q", got.EmailAddress)
	}
}

func TestGetSignature_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Sender signature not found"}),
		}, nil
	})))

	_, err := api.GetSignature(9999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- CreateSignature ----------------------------------------------------------

func TestCreateSignature_Success(t *testing.T) {
	want := SignatureResp{
		ID:           10,
		EmailAddress: "new@example.com",
		Name:         "New Sender",
	}

	api := New(APITokenOpt("acct-tok"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/senders") {
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
		var sent CreateSignatureReq
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if sent.FromEmail != "new@example.com" {
			t.Errorf("request body FromEmail = %q, want new@example.com", sent.FromEmail)
		}
		if sent.Name != "New Sender" {
			t.Errorf("request body Name = %q, want New Sender", sent.Name)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.CreateSignature(&CreateSignatureReq{
		FromEmail: "new@example.com",
		Name:      "New Sender",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 10 {
		t.Errorf("ID = %d, want 10", got.ID)
	}
	if got.Name != "New Sender" {
		t.Errorf("Name = %q, want New Sender", got.Name)
	}
}

func TestCreateSignature_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.CreateSignature(&CreateSignatureReq{FromEmail: "bad@bad.com", Name: "Bad"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- EditSignature ------------------------------------------------------------

func TestEditSignature_Success(t *testing.T) {
	want := SignatureResp{
		ID:   7,
		Name: "Updated Sender",
	}

	api := New(APITokenOpt("acct-tok"), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/senders/7") {
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
		var sent EditSignatureReq
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if sent.Name != "Updated Sender" {
			t.Errorf("request body Name = %q, want Updated Sender", sent.Name)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.EditSignature(7, &EditSignatureReq{Name: "Updated Sender"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Updated Sender" {
		t.Errorf("Name = %q, want Updated Sender", got.Name)
	}
}

func TestEditSignature_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Signature not found"}),
		}, nil
	})))

	_, err := api.EditSignature(9999, &EditSignatureReq{Name: "Ghost"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- DeleteSignature ----------------------------------------------------------

func TestDeleteSignature_Success(t *testing.T) {
	want := DeleteResp{Message: "Signature sender deleted."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/senders/5") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteSignature(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Signature sender deleted." {
		t.Errorf("Message = %q", got.Message)
	}
}

func TestDeleteSignature_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Signature not found"}),
		}, nil
	})))

	_, err := api.DeleteSignature(9999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- ResendSignatureConfirmation ----------------------------------------------

func TestResendSignatureConfirmation_Success(t *testing.T) {
	want := ResendResp{Message: "Confirmation email has been re-sent."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/senders/8/resend") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ResendSignatureConfirmation(8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Confirmation email has been re-sent." {
		t.Errorf("Message = %q", got.Message)
	}
}

func TestResendSignatureConfirmation_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "Sender signature not found"}),
		}, nil
	})))

	_, err := api.ResendSignatureConfirmation(9999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- RotateSignatureDKIM -----------------------------------------------------

func TestRotateSignatureDKIM_Success(t *testing.T) {
	want := SignatureResp{
		ID:              9,
		DKIMPendingHost: "new-dkim._domainkey.example.com",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/senders/9/rotateDkim") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.RotateSignatureDKIM(9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DKIMPendingHost != "new-dkim._domainkey.example.com" {
		t.Errorf("DKIMPendingHost = %q", got.DKIMPendingHost)
	}
}

func TestRotateSignatureDKIM_APIError(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 500, Message: "server error"}),
		}, nil
	})))

	_, err := api.RotateSignatureDKIM(9)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
