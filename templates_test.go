package postmark

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// ---- ListTemplates -------------------------------------------------------------

func TestListTemplates_Success(t *testing.T) {
	want := ListTemplatesResp{
		TotalCount: 2,
		Templates: []TemplateResp{
			{TemplateID: 1, Name: "Welcome"},
			{TemplateID: 2, Name: "Reset Password"},
		},
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/templates") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if !strings.Contains(req.URL.RawQuery, "count=10") {
				t.Errorf("expected count param, query=%s", req.URL.RawQuery)
			}
			if !strings.Contains(req.URL.RawQuery, "offset=0") {
				t.Errorf("expected offset param, query=%s", req.URL.RawQuery)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.ListTemplates(10, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
	if len(got.Templates) != 2 {
		t.Errorf("len(Templates) = %d, want 2", len(got.Templates))
	}
}

func TestListTemplates_WithLayoutTemplate(t *testing.T) {
	want := ListTemplatesResp{TotalCount: 1, Templates: []TemplateResp{{TemplateID: 5, Name: "Promo"}}}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.RawQuery, "layoutTemplate=base-layout") {
				t.Errorf("expected layoutTemplate param, query=%s", req.URL.RawQuery)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.ListTemplates(5, 0, "base-layout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
}

func TestListTemplates_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.ListTemplates(10, 0, "")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestListTemplates_CountZero verifies that a count of 0 returns a local error
// immediately, rather than forwarding an invalid request to Postmark.
func TestListTemplates_CountZero(t *testing.T) {
	api := New(ServerTokenOpt("test-server-token"))

	_, err := api.ListTemplates(0, 0, "")
	if err == nil {
		t.Fatal("expected error for count=0, got nil")
	}
	if !strings.Contains(err.Error(), "count must be >= 1") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestListTemplates_CountNegative verifies that a negative count returns a local error.
func TestListTemplates_CountNegative(t *testing.T) {
	api := New(ServerTokenOpt("test-server-token"))

	_, err := api.ListTemplates(-5, 0, "")
	if err == nil {
		t.Fatal("expected error for count=-5, got nil")
	}
	if !strings.Contains(err.Error(), "count must be >= 1") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestListTemplates_OffsetNegative verifies that a negative offset returns a local error.
func TestListTemplates_OffsetNegative(t *testing.T) {
	api := New(ServerTokenOpt("test-server-token"))

	_, err := api.ListTemplates(10, -1, "")
	if err == nil {
		t.Fatal("expected error for offset=-1, got nil")
	}
	if !strings.Contains(err.Error(), "offset must be >= 0") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---- CreateTemplate ------------------------------------------------------------

func TestCreateTemplate_Success(t *testing.T) {
	want := TemplateResp{
		TemplateID: 42,
		Name:       "My Template",
		Alias:      "my-template",
		Subject:    "Hello {{name}}",
		HtmlBody:   "<p>Hello {{name}}</p>",
		Active:     true,
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/templates") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.CreateTemplate(&CreateTemplateReq{
		Name:     "My Template",
		Alias:    "my-template",
		Subject:  "Hello {{name}}",
		HtmlBody: "<p>Hello {{name}}</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateID != 42 {
		t.Errorf("TemplateID = %d, want 42", got.TemplateID)
	}
	if got.Name != "My Template" {
		t.Errorf("Name = %q, want %q", got.Name, "My Template")
	}
}

func TestCreateTemplate_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 422, Message: "validation error"}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.CreateTemplate(&CreateTemplateReq{Name: "Bad"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetTemplate ---------------------------------------------------------------

func TestGetTemplate_ByID_Success(t *testing.T) {
	want := TemplateResp{
		TemplateID: 99,
		Name:       "Invoice",
		Subject:    "Your invoice #{{id}}",
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/templates/99") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.GetTemplate("99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateID != 99 {
		t.Errorf("TemplateID = %d, want 99", got.TemplateID)
	}
}

func TestGetTemplate_ByAlias_Success(t *testing.T) {
	want := TemplateResp{
		TemplateID: 77,
		Name:       "Welcome Email",
		Alias:      "welcome-email",
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if !strings.HasSuffix(req.URL.Path, "/templates/welcome-email") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.GetTemplate("welcome-email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Alias != "welcome-email" {
		t.Errorf("Alias = %q, want %q", got.Alias, "welcome-email")
	}
}

func TestGetTemplate_NotFound(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: 1101, Message: "Template not found"}),
			}, nil
		})),
	)

	_, err := api.GetTemplate("9999")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- UpdateTemplate ------------------------------------------------------------

func TestUpdateTemplate_Success(t *testing.T) {
	want := TemplateResp{
		TemplateID: 42,
		Name:       "Updated Template",
		Subject:    "New Subject",
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPut {
				t.Errorf("expected PUT, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/templates/42") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.UpdateTemplate("42", &UpdateTemplateReq{Name: "Updated Template", Subject: "New Subject"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Updated Template" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated Template")
	}
}

func TestUpdateTemplate_NotFound(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: 1101, Message: "Template not found"}),
			}, nil
		})),
	)

	_, err := api.UpdateTemplate("9999", &UpdateTemplateReq{Name: "Ghost"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- DeleteTemplate ------------------------------------------------------------

func TestDeleteTemplate_Success(t *testing.T) {
	want := DeleteResp{Message: "Template removed."}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/templates/55") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.DeleteTemplate("55")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Template removed." {
		t.Errorf("Message = %q, want %q", got.Message, "Template removed.")
	}
}

func TestDeleteTemplate_NotFound(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       jsonBody(t, PostmarkErr{ErrorCode: 1101, Message: "Template not found"}),
			}, nil
		})),
	)

	_, err := api.DeleteTemplate("9999")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestDeleteTemplate_ByAlias(t *testing.T) {
	want := DeleteResp{Message: "Template removed."}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/templates/my-alias") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.DeleteTemplate("my-alias")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Template removed." {
		t.Errorf("Message = %q, want %q", got.Message, "Template removed.")
	}
}

// ---- SendEmailWithTemplate -----------------------------------------------------

func TestSendEmailWithTemplate_ByID_Success(t *testing.T) {
	want := SendEmailResp{
		To:        "recipient@example.com",
		MessageID: "msg-abc-123",
		Message:   "OK",
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/email/withTemplate") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateID:    101,
		TemplateModel: map[string]interface{}{"name": "Alice"},
		From:          "sender@example.com",
		To:            "recipient@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MessageID != "msg-abc-123" {
		t.Errorf("MessageID = %q, want %q", got.MessageID, "msg-abc-123")
	}
	if got.To != "recipient@example.com" {
		t.Errorf("To = %q, want %q", got.To, "recipient@example.com")
	}
}

func TestSendEmailWithTemplate_ByAlias_Success(t *testing.T) {
	want := SendEmailResp{
		To:        "user@example.com",
		MessageID: "msg-xyz-456",
		Message:   "OK",
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, want),
			}, nil
		})),
	)

	got, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateAlias: "welcome-email",
		TemplateModel: map[string]interface{}{"name": "Bob"},
		From:          "hello@example.com",
		To:            "user@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MessageID != "msg-xyz-456" {
		t.Errorf("MessageID = %q, want %q", got.MessageID, "msg-xyz-456")
	}
}

func TestSendEmailWithTemplate_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 1101, Message: "Template not found"}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateID:    9999,
		TemplateModel: map[string]interface{}{},
		From:          "sender@example.com",
		To:            "recipient@example.com",
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestSendEmailWithTemplate_TrackOpensFalse verifies that explicitly setting
// TrackOpens to false serialises the field into the JSON request body rather
// than omitting it. This is important because omitting the field causes Postmark
// to apply its server-side default instead of honouring the caller's intent to
// disable open tracking.
func TestSendEmailWithTemplate_TrackOpensFalse(t *testing.T) {
	trackOpens := false

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			// Read and decode the request body to verify TrackOpens is present.
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			var decoded map[string]interface{}
			if err = json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}

			val, ok := decoded["TrackOpens"]
			if !ok {
				t.Error("TrackOpens key must be present in request body when set to false; got omitted")
			} else if val != false {
				t.Errorf("TrackOpens = %v, want false", val)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{}),
			}, nil
		})),
	)

	_, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateID:    1,
		TemplateModel: map[string]interface{}{},
		From:          "sender@example.com",
		To:            "recipient@example.com",
		TrackOpens:    &trackOpens,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSendEmailWithTemplate_TrackOpensTrue verifies that setting TrackOpens to
// true also serialises the field correctly.
func TestSendEmailWithTemplate_TrackOpensTrue(t *testing.T) {
	trackOpens := true

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			var decoded map[string]interface{}
			if err = json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}

			val, ok := decoded["TrackOpens"]
			if !ok {
				t.Error("TrackOpens key must be present in request body when set to true")
			} else if val != true {
				t.Errorf("TrackOpens = %v, want true", val)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{}),
			}, nil
		})),
	)

	_, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateID:    1,
		TemplateModel: map[string]interface{}{},
		From:          "sender@example.com",
		To:            "recipient@example.com",
		TrackOpens:    &trackOpens,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSendEmailWithTemplate_TrackOpensNil verifies that leaving TrackOpens as
// nil omits the field entirely, deferring to the Postmark server default.
func TestSendEmailWithTemplate_TrackOpensNil(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			var decoded map[string]interface{}
			if err = json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}

			if _, ok := decoded["TrackOpens"]; ok {
				t.Error("TrackOpens key must be absent in request body when TrackOpens is nil")
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{}),
			}, nil
		})),
	)

	_, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateID:    1,
		TemplateModel: map[string]interface{}{},
		From:          "sender@example.com",
		To:            "recipient@example.com",
		// TrackOpens deliberately omitted (nil)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSendEmailWithTemplate_TemplateModelNilOmitted verifies that a nil
// TemplateModel is omitted from the JSON body entirely (omitempty), rather
// than serialised as "TemplateModel":null which Postmark may reject.
func TestSendEmailWithTemplate_TemplateModelNilOmitted(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			var decoded map[string]interface{}
			if err = json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}

			if _, ok := decoded["TemplateModel"]; ok {
				t.Error("TemplateModel key must be absent in request body when TemplateModel is nil (omitempty)")
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, SendEmailResp{}),
			}, nil
		})),
	)

	_, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateID: 1,
		From:       "sender@example.com",
		To:         "recipient@example.com",
		// TemplateModel deliberately nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- SendEmailBatchWithTemplates -----------------------------------------------

func TestSendEmailBatchWithTemplates_Success(t *testing.T) {
	// Use an exported-field struct (batchWithTemplatesAPIResp) to build the
	// mock response body. This avoids depending on custom marshal logic.
	wantResp := batchWithTemplatesAPIResp{
		TotalSent:   2,
		TotalFailed: 0,
		Messages: []SendEmailResp{
			{To: "alice@example.com", MessageID: "msg-001", Message: "OK"},
			{To: "bob@example.com", MessageID: "msg-002", Message: "OK"},
		},
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/email/batchWithTemplates") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			if req.Header.Get("X-Postmark-Server-Token") == "" {
				t.Error("expected X-Postmark-Server-Token header to be set")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, wantResp),
			}, nil
		})),
	)

	got, err := api.SendEmailBatchWithTemplates(&BatchWithTemplatesReq{
		Messages: []SendWithTemplateReq{
			{
				TemplateAlias: "welcome-email",
				TemplateModel: map[string]interface{}{"name": "Alice"},
				From:          "sender@example.com",
				To:            "alice@example.com",
			},
			{
				TemplateAlias: "welcome-email",
				TemplateModel: map[string]interface{}{"name": "Bob"},
				From:          "sender@example.com",
				To:            "bob@example.com",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(responses) = %d, want 2", len(got))
	}
	if got[0].MessageID != "msg-001" {
		t.Errorf("got[0].MessageID = %q, want %q", got[0].MessageID, "msg-001")
	}
	if got[1].MessageID != "msg-002" {
		t.Errorf("got[1].MessageID = %q, want %q", got[1].MessageID, "msg-002")
	}
}

func TestSendEmailBatchWithTemplates_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal error"}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       jsonBody(t, pmErr),
			}, nil
		})),
	)

	_, err := api.SendEmailBatchWithTemplates(&BatchWithTemplatesReq{
		Messages: []SendWithTemplateReq{
			{
				TemplateAlias: "welcome-email",
				TemplateModel: map[string]interface{}{},
				From:          "sender@example.com",
				To:            "recipient@example.com",
			},
		},
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSendEmailBatchWithTemplates_EmptyBatch(t *testing.T) {
	wantResp := batchWithTemplatesAPIResp{
		TotalSent:   0,
		TotalFailed: 0,
		Messages:    []SendEmailResp{},
	}

	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, wantResp),
			}, nil
		})),
	)

	got, err := api.SendEmailBatchWithTemplates(&BatchWithTemplatesReq{
		Messages: []SendWithTemplateReq{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 responses, got %d", len(got))
	}
}

// ---- Empty server token --------------------------------------------------------

// TestServerTokenEmpty verifies that all methods return a descriptive local error
// immediately when no server token has been configured, rather than silently
// issuing a request with an empty X-Postmark-Server-Token header.
func TestServerTokenEmpty(t *testing.T) {
	// Deliberately do NOT set ServerTokenOpt; POSTMARK_SERVER_TOKEN is unset in CI.
	// We also do not inject an HTTP client, so any accidental network call would fail.
	api := &API{
		baseHost:    "https://api.postmarkapp.com",
		serverToken: "", // explicitly empty
	}

	t.Run("ListTemplates", func(t *testing.T) {
		_, err := api.ListTemplates(1, 0, "")
		if err == nil {
			t.Fatal("expected error for empty server token, got nil")
		}
		if !strings.Contains(err.Error(), "server token is empty") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("CreateTemplate", func(t *testing.T) {
		_, err := api.CreateTemplate(&CreateTemplateReq{Name: "T"})
		if err == nil {
			t.Fatal("expected error for empty server token, got nil")
		}
		if !strings.Contains(err.Error(), "server token is empty") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("GetTemplate", func(t *testing.T) {
		_, err := api.GetTemplate("1")
		if err == nil {
			t.Fatal("expected error for empty server token, got nil")
		}
		if !strings.Contains(err.Error(), "server token is empty") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("UpdateTemplate", func(t *testing.T) {
		_, err := api.UpdateTemplate("1", &UpdateTemplateReq{})
		if err == nil {
			t.Fatal("expected error for empty server token, got nil")
		}
		if !strings.Contains(err.Error(), "server token is empty") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("DeleteTemplate", func(t *testing.T) {
		_, err := api.DeleteTemplate("1")
		if err == nil {
			t.Fatal("expected error for empty server token, got nil")
		}
		if !strings.Contains(err.Error(), "server token is empty") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("SendEmailWithTemplate", func(t *testing.T) {
		_, err := api.SendEmailWithTemplate(&SendWithTemplateReq{TemplateID: 1, From: "a@b.c", To: "x@y.z"})
		if err == nil {
			t.Fatal("expected error for empty server token, got nil")
		}
		if !strings.Contains(err.Error(), "server token is empty") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("SendEmailBatchWithTemplates", func(t *testing.T) {
		_, err := api.SendEmailBatchWithTemplates(&BatchWithTemplatesReq{Messages: []SendWithTemplateReq{}})
		if err == nil {
			t.Fatal("expected error for empty server token, got nil")
		}
		if !strings.Contains(err.Error(), "server token is empty") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// ---- Header verification -------------------------------------------------------

// TestTemplates_UsesServerTokenHeader verifies that all template methods set
// the X-Postmark-Server-Token header (from the dedicated serverToken field) and
// NOT the X-Postmark-Account-Token header.
func TestTemplates_UsesServerTokenHeader(t *testing.T) {
	const serverToken = "server-tok-xyz"

	checkHeaders := func(t *testing.T, req *http.Request) {
		t.Helper()
		if req.Header.Get("X-Postmark-Server-Token") != serverToken {
			t.Errorf("X-Postmark-Server-Token = %q, want %q", req.Header.Get("X-Postmark-Server-Token"), serverToken)
		}
		if req.Header.Get("X-Postmark-Account-Token") != "" {
			t.Errorf("X-Postmark-Account-Token should NOT be set for template endpoints, got %q", req.Header.Get("X-Postmark-Account-Token"))
		}
	}

	t.Run("ListTemplates", func(t *testing.T) {
		api := New(ServerTokenOpt(serverToken), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, ListTemplatesResp{})}, nil
		})))
		api.ListTemplates(1, 0, "") //nolint:errcheck
	})

	t.Run("CreateTemplate", func(t *testing.T) {
		api := New(ServerTokenOpt(serverToken), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, TemplateResp{})}, nil
		})))
		api.CreateTemplate(&CreateTemplateReq{Name: "T"}) //nolint:errcheck
	})

	t.Run("GetTemplate", func(t *testing.T) {
		api := New(ServerTokenOpt(serverToken), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, TemplateResp{})}, nil
		})))
		api.GetTemplate("1") //nolint:errcheck
	})

	t.Run("UpdateTemplate", func(t *testing.T) {
		api := New(ServerTokenOpt(serverToken), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, TemplateResp{})}, nil
		})))
		api.UpdateTemplate("1", &UpdateTemplateReq{}) //nolint:errcheck
	})

	t.Run("DeleteTemplate", func(t *testing.T) {
		api := New(ServerTokenOpt(serverToken), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, DeleteResp{})}, nil
		})))
		api.DeleteTemplate("1") //nolint:errcheck
	})

	t.Run("SendEmailWithTemplate", func(t *testing.T) {
		api := New(ServerTokenOpt(serverToken), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, SendEmailResp{})}, nil
		})))
		api.SendEmailWithTemplate(&SendWithTemplateReq{TemplateID: 1, TemplateModel: map[string]interface{}{}, From: "a@b.c", To: "x@y.z"}) //nolint:errcheck
	})

	t.Run("SendEmailBatchWithTemplates", func(t *testing.T) {
		api := New(ServerTokenOpt(serverToken), HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			checkHeaders(t, req)
			return &http.Response{StatusCode: http.StatusOK, Body: jsonBody(t, batchWithTemplatesAPIResp{Messages: []SendEmailResp{}})}, nil
		})))
		api.SendEmailBatchWithTemplates(&BatchWithTemplatesReq{Messages: []SendWithTemplateReq{}}) //nolint:errcheck
	})
}

// ---- URL encoding --------------------------------------------------------------

// TestGetTemplate_PathEscaping verifies that special characters in idOrAlias
// are properly URL-encoded so that template aliases containing '/', '?', or '#'
// do not corrupt the request path.
func TestGetTemplate_PathEscaping(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			// url.PathEscape encodes '/' as '%2F'; the raw path must not contain a literal '/'.
			if strings.Contains(req.URL.RawPath, "//") {
				t.Errorf("unescaped slash in path: %s", req.URL.RawPath)
			}
			if !strings.Contains(req.URL.EscapedPath(), "slash%2Falias") {
				t.Errorf("expected 'slash%%2Falias' in escaped path, got %s", req.URL.EscapedPath())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       jsonBody(t, TemplateResp{}),
			}, nil
		})),
	)

	api.GetTemplate("slash/alias") //nolint:errcheck
}
