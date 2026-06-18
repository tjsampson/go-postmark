package postmark

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// ---- SendEmailWithTemplate -----------------------------------------------------

func TestSendEmailWithTemplate_Success(t *testing.T) {
	want := EmailResp{
		To:        "recipient@example.com",
		MessageID: "msg-id-123",
		ErrorCode: 0,
		Message:   "OK",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/email/withTemplate") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.SendEmailWithTemplate(&SendEmailWithTemplateReq{
		TemplateID:    1,
		TemplateModel: TemplateModel{"name": "Alice"},
		From:          "sender@example.com",
		To:            "recipient@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MessageID != want.MessageID {
		t.Errorf("MessageID = %q, want %q", got.MessageID, want.MessageID)
	}
	if got.To != want.To {
		t.Errorf("To = %q, want %q", got.To, want.To)
	}
}

func TestSendEmailWithTemplate_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 300, Message: "Invalid template ID"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.SendEmailWithTemplate(&SendEmailWithTemplateReq{
		TemplateID:    9999,
		TemplateModel: TemplateModel{},
		From:          "sender@example.com",
		To:            "recipient@example.com",
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestSendEmailWithTemplate_InBodyErrorCode verifies that when Postmark returns
// HTTP 200 but with a non-zero ErrorCode in the body (e.g. ErrorCode 406 for
// individual message failures), SendEmailWithTemplate surfaces it as a Go error.
func TestSendEmailWithTemplate_InBodyErrorCode(t *testing.T) {
	body := EmailResp{
		To:        "recipient@example.com",
		ErrorCode: 406,
		Message:   "You tried to send to a recipient that has been marked as inactive.",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, body),
		}, nil
	})))

	_, err := api.SendEmailWithTemplate(&SendEmailWithTemplateReq{
		TemplateID: 1,
		From:       "sender@example.com",
		To:         "inactive@example.com",
	})
	if err == nil {
		t.Fatal("expected error for non-zero ErrorCode in 200 body, got nil")
	}
	var pmErr PostmarkErr
	if !errors.As(err, &pmErr) {
		t.Errorf("expected PostmarkErr, got %T: %v", err, err)
	}
	if pmErr.ErrorCode != 406 {
		t.Errorf("ErrorCode = %d, want 406", pmErr.ErrorCode)
	}
}

func TestSendEmailWithTemplate_WithAlias(t *testing.T) {
	want := EmailResp{
		To:        "recipient@example.com",
		MessageID: "msg-alias-456",
		ErrorCode: 0,
		Message:   "OK",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.SendEmailWithTemplate(&SendEmailWithTemplateReq{
		TemplateAlias: "welcome-email",
		TemplateModel: TemplateModel{"name": "Bob"},
		From:          "sender@example.com",
		To:            "recipient@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MessageID != want.MessageID {
		t.Errorf("MessageID = %q, want %q", got.MessageID, want.MessageID)
	}
}

// TestSendEmailWithTemplate_TrackOpensFalse verifies that explicitly setting
// TrackOpens to false is serialised into the JSON body (not silently dropped).
// This requires TrackOpens to be *bool rather than bool with omitempty.
func TestSendEmailWithTemplate_TrackOpensFalse(t *testing.T) {
	trackOpens := false

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		var body map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		val, ok := body["TrackOpens"]
		if !ok {
			t.Error("TrackOpens field missing from JSON body; *bool with omitempty should include false")
		} else if val != false {
			t.Errorf("TrackOpens = %v, want false", val)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: jsonBody(t, EmailResp{
				To:        "recipient@example.com",
				MessageID: "msg-track-789",
				ErrorCode: 0,
				Message:   "OK",
			}),
		}, nil
	})))

	_, err := api.SendEmailWithTemplate(&SendEmailWithTemplateReq{
		TemplateID:  1,
		From:        "sender@example.com",
		To:          "recipient@example.com",
		TrackOpens:  &trackOpens,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- SendEmailBatchWithTemplates -----------------------------------------------

func TestSendEmailBatchWithTemplates_Success(t *testing.T) {
	// The Postmark batch-with-templates API wraps results in {"Messages":[...]}.
	// The mock must return this envelope so the unmarshal logic is exercised correctly.
	wantMessages := []EmailResp{
		{To: "a@example.com", MessageID: "batch-1", ErrorCode: 0, Message: "OK"},
		{To: "b@example.com", MessageID: "batch-2", ErrorCode: 0, Message: "OK"},
	}
	envelope := batchWithTemplatesResp{Messages: wantMessages}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/email/batchWithTemplates") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, envelope),
		}, nil
	})))

	got, err := api.SendEmailBatchWithTemplates(&BatchWithTemplatesReq{
		Messages: []SendEmailWithTemplateReq{
			{TemplateID: 1, TemplateModel: TemplateModel{}, From: "sender@example.com", To: "a@example.com"},
			{TemplateID: 2, TemplateModel: TemplateModel{}, From: "sender@example.com", To: "b@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].MessageID != "batch-1" {
		t.Errorf("got[0].MessageID = %q, want batch-1", got[0].MessageID)
	}
	if got[1].MessageID != "batch-2" {
		t.Errorf("got[1].MessageID = %q, want batch-2", got[1].MessageID)
	}
	if got[0].To != "a@example.com" {
		t.Errorf("got[0].To = %q, want a@example.com", got[0].To)
	}
	if got[1].To != "b@example.com" {
		t.Errorf("got[1].To = %q, want b@example.com", got[1].To)
	}
}

func TestSendEmailBatchWithTemplates_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.SendEmailBatchWithTemplates(&BatchWithTemplatesReq{
		Messages: []SendEmailWithTemplateReq{},
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetTemplate ---------------------------------------------------------------

func TestGetTemplate_Success(t *testing.T) {
	want := TemplateResp{
		TemplateID: 42,
		Name:       "Welcome Email",
		Subject:    "Welcome to our service",
		Active:     true,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/templates/42") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetTemplate("42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateID != want.TemplateID {
		t.Errorf("TemplateID = %d, want %d", got.TemplateID, want.TemplateID)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}

func TestGetTemplate_ByAlias(t *testing.T) {
	want := TemplateResp{
		TemplateID: 10,
		Name:       "Password Reset",
		Alias:      "password-reset",
		Active:     true,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(req.URL.Path, "/templates/password-reset") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetTemplate("password-reset")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Alias != "password-reset" {
		t.Errorf("Alias = %q, want password-reset", got.Alias)
	}
}

func TestGetTemplate_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 1101, Message: "Template not found"}),
		}, nil
	})))

	_, err := api.GetTemplate("99999")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- CreateTemplate ------------------------------------------------------------

func TestCreateTemplate_Success(t *testing.T) {
	want := TemplateResp{
		TemplateID: 55,
		Name:       "New Template",
		Subject:    "Hello {{name}}",
		Active:     true,
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/templates") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.CreateTemplate(&CreateTemplateReq{
		Name:     "New Template",
		Subject:  "Hello {{name}}",
		HtmlBody: "<p>Hello {{name}}</p>",
		TextBody: "Hello {{name}}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateID != want.TemplateID {
		t.Errorf("TemplateID = %d, want %d", got.TemplateID, want.TemplateID)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}

func TestCreateTemplate_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 1105, Message: "Template alias already in use"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateTemplate(&CreateTemplateReq{
		Name:  "Duplicate",
		Alias: "existing-alias",
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- UpdateTemplate ------------------------------------------------------------

func TestUpdateTemplate_Success(t *testing.T) {
	want := TemplateResp{
		TemplateID: 7,
		Name:       "Updated Template",
		Subject:    "New Subject",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/templates/7") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.UpdateTemplate("7", &UpdateTemplateReq{
		Name:    "Updated Template",
		Subject: "New Subject",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Updated Template" {
		t.Errorf("Name = %q, want Updated Template", got.Name)
	}
}

func TestUpdateTemplate_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 1101, Message: "Template not found"}),
		}, nil
	})))

	_, err := api.UpdateTemplate("9999", &UpdateTemplateReq{Name: "Ghost"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestUpdateTemplate_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.UpdateTemplate("5", &UpdateTemplateReq{Name: "Bad"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- ListTemplates -------------------------------------------------------------

func TestListTemplates_Success(t *testing.T) {
	want := ListTemplatesResp{
		TotalCount: 3,
		Templates: []TemplateResp{
			{TemplateID: 1, Name: "Template A"},
			{TemplateID: 2, Name: "Template B"},
			{TemplateID: 3, Name: "Template C"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.RawQuery, "Count=10") {
			t.Errorf("expected Count param, query=%s", req.URL.RawQuery)
		}
		if !strings.Contains(req.URL.RawQuery, "Offset=0") {
			t.Errorf("expected Offset param, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListTemplates(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", got.TotalCount)
	}
	if len(got.Templates) != 3 {
		t.Errorf("len(Templates) = %d, want 3", len(got.Templates))
	}
}

func TestListTemplates_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.ListTemplates(10, 0)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestListTemplates_Pagination(t *testing.T) {
	want := ListTemplatesResp{
		TotalCount: 100,
		Templates:  []TemplateResp{{TemplateID: 51, Name: "Template 51"}},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.RawQuery, "Count=1") {
			t.Errorf("expected Count=1, query=%s", req.URL.RawQuery)
		}
		if !strings.Contains(req.URL.RawQuery, "Offset=50") {
			t.Errorf("expected Offset=50, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListTemplates(1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 100 {
		t.Errorf("TotalCount = %d, want 100", got.TotalCount)
	}
}

// ---- DeleteTemplate ------------------------------------------------------------

func TestDeleteTemplate_Success(t *testing.T) {
	want := DeleteTemplateResp{ErrorCode: 0, Message: "Template removed."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/templates/99") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteTemplate("99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Template removed." {
		t.Errorf("Message = %q, want Template removed.", got.Message)
	}
}

func TestDeleteTemplate_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 1101, Message: "Template not found"}),
		}, nil
	})))

	_, err := api.DeleteTemplate("9999")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestDeleteTemplate_ByAlias(t *testing.T) {
	want := DeleteTemplateResp{ErrorCode: 0, Message: "Template removed."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(req.URL.Path, "/templates/my-alias") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteTemplate("my-alias")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ErrorCode != 0 {
		t.Errorf("ErrorCode = %d, want 0", got.ErrorCode)
	}
}
