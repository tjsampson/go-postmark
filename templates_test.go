package postmark

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// ---- SendEmailWithTemplate -----------------------------------------------------

func TestSendEmailWithTemplate_Success(t *testing.T) {
	want := SendEmailResp{
		To:        "recipient@example.com",
		MessageID: "abc-123",
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

	got, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateID:    42,
		TemplateModel: map[string]string{"name": "World"},
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

func TestSendEmailWithTemplate_WithAlias(t *testing.T) {
	want := SendEmailResp{
		To:        "recipient@example.com",
		MessageID: "msg-alias-001",
		Message:   "OK",
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

	got, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateAlias: "welcome-email",
		TemplateModel: map[string]interface{}{"user": "Alice"},
		InlineCss:     true,
		From:          "sender@example.com",
		To:            "recipient@example.com",
		Tag:           "onboarding",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MessageID != want.MessageID {
		t.Errorf("MessageID = %q, want %q", got.MessageID, want.MessageID)
	}
}

func TestSendEmailWithTemplate_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 422, Message: "Invalid template model"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateID:    1,
		TemplateModel: nil,
		From:          "sender@example.com",
		To:            "recipient@example.com",
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- GetTemplate ---------------------------------------------------------------

func TestGetTemplate_Success(t *testing.T) {
	want := TemplateResp{
		TemplateID: 123,
		Name:       "Welcome Email",
		Subject:    "Welcome, {{name}}!",
		HtmlBody:   "<h1>Welcome</h1>",
		TextBody:   "Welcome",
		Alias:      "welcome",
		TemplateType: "Standard",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/templates/123") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.GetTemplate("123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateID != want.TemplateID {
		t.Errorf("TemplateID = %d, want %d", got.TemplateID, want.TemplateID)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Alias != want.Alias {
		t.Errorf("Alias = %q, want %q", got.Alias, want.Alias)
	}
}

func TestGetTemplate_ByAlias(t *testing.T) {
	want := TemplateResp{
		TemplateID: 456,
		Name:       "Password Reset",
		Alias:      "password-reset",
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

	_, err := api.GetTemplate("9999")
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
		TemplateID:   789,
		Name:         "New Template",
		Subject:      "Hello {{name}}",
		HtmlBody:     "<p>Hello {{name}}</p>",
		TextBody:     "Hello {{name}}",
		Alias:        "new-template",
		TemplateType: "Standard",
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
		Alias:    "new-template",
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
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateTemplate(&CreateTemplateReq{Name: "Bad"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- EditTemplate --------------------------------------------------------------

func TestEditTemplate_Success(t *testing.T) {
	want := TemplateResp{
		TemplateID: 100,
		Name:       "Updated Template",
		Subject:    "Updated Subject",
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/templates/100") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.EditTemplate("100", &EditTemplateReq{
		Name:    "Updated Template",
		Subject: "Updated Subject",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Subject != want.Subject {
		t.Errorf("Subject = %q, want %q", got.Subject, want.Subject)
	}
}

func TestEditTemplate_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 1101, Message: "Template not found"}),
		}, nil
	})))

	_, err := api.EditTemplate("9999", &EditTemplateReq{Name: "Ghost"})
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

// ---- ListTemplates -------------------------------------------------------------

func TestListTemplates_Success(t *testing.T) {
	want := ListTemplatesResp{
		TotalCount: 3,
		Templates: []TemplateListItem{
			{TemplateID: 1, Name: "Template A", Active: true},
			{TemplateID: 2, Name: "Template B", Active: true},
			{TemplateID: 3, Name: "Template C", Active: false},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/templates") {
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

func TestListTemplates_WithOffset(t *testing.T) {
	want := ListTemplatesResp{
		TotalCount: 10,
		Templates:  []TemplateListItem{{TemplateID: 6, Name: "Template F"}},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.RawQuery, "count=5") {
			t.Errorf("expected count=5, query=%s", req.URL.RawQuery)
		}
		if !strings.Contains(req.URL.RawQuery, "offset=5") {
			t.Errorf("expected offset=5, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListTemplates(5, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 10 {
		t.Errorf("TotalCount = %d, want 10", got.TotalCount)
	}
}

// ---- DeleteTemplate ------------------------------------------------------------

func TestDeleteTemplate_Success(t *testing.T) {
	want := DeleteResp{Message: "Template deleted."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/templates/55") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteTemplate("55")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Template deleted." {
		t.Errorf("Message = %q, want 'Template deleted.'", got.Message)
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

// ---- ValidateTemplate ----------------------------------------------------------

func TestValidateTemplate_Success(t *testing.T) {
	want := ValidateTemplateResp{
		AllContentIsValid: true,
		HtmlBody: TemplateValidationResult{
			ContentIsValid:  true,
			RenderedContent: "<h1>Hello World</h1>",
		},
		TextBody: TemplateValidationResult{
			ContentIsValid:  true,
			RenderedContent: "Hello World",
		},
		Subject: TemplateValidationResult{
			ContentIsValid:  true,
			RenderedContent: "Hello World",
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/templates/validate") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ValidateTemplate(&ValidateTemplateReq{
		Subject:  "Hello {{name}}",
		HtmlBody: "<h1>Hello {{name}}</h1>",
		TextBody: "Hello {{name}}",
		TestRenderModel: map[string]string{
			"name": "World",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.AllContentIsValid {
		t.Error("expected AllContentIsValid to be true")
	}
	if got.HtmlBody.RenderedContent != "<h1>Hello World</h1>" {
		t.Errorf("HtmlBody.RenderedContent = %q", got.HtmlBody.RenderedContent)
	}
}

func TestValidateTemplate_WithErrors(t *testing.T) {
	want := ValidateTemplateResp{
		AllContentIsValid: false,
		HtmlBody: TemplateValidationResult{
			ContentIsValid: false,
			ValidationErrors: []TemplateValidationError{
				{Message: "Unclosed tag", Line: 1, CharacterPosition: 5},
			},
		},
		Subject: TemplateValidationResult{
			ContentIsValid: true,
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ValidateTemplate(&ValidateTemplateReq{
		Subject:  "Hello",
		HtmlBody: "{{#each unclosed",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AllContentIsValid {
		t.Error("expected AllContentIsValid to be false")
	}
	if len(got.HtmlBody.ValidationErrors) != 1 {
		t.Errorf("expected 1 validation error, got %d", len(got.HtmlBody.ValidationErrors))
	}
	if got.HtmlBody.ValidationErrors[0].Message != "Unclosed tag" {
		t.Errorf("ValidationError.Message = %q", got.HtmlBody.ValidationErrors[0].Message)
	}
}

// ---- PushTemplate --------------------------------------------------------------

func TestPushTemplate_Success(t *testing.T) {
	want := PushTemplateResp{
		TotalCount: 2,
		Templates: []PushTemplateChange{
			{Action: "Create", TemplateID: 10, Alias: "welcome", Name: "Welcome Email"},
			{Action: "Update", TemplateID: 11, Alias: "reset", Name: "Password Reset"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/templates/push") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.PushTemplate(&PushTemplateReq{
		SourceServerID:      1,
		DestinationServerID: 2,
		PerformChanges:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
	if len(got.Templates) != 2 {
		t.Errorf("len(Templates) = %d, want 2", len(got.Templates))
	}
	if got.Templates[0].Action != "Create" {
		t.Errorf("Templates[0].Action = %q, want Create", got.Templates[0].Action)
	}
}

func TestPushTemplate_DryRun(t *testing.T) {
	want := PushTemplateResp{
		TotalCount: 1,
		Templates: []PushTemplateChange{
			{Action: "Update", TemplateID: 20, Alias: "promo", Name: "Promo Email"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.PushTemplate(&PushTemplateReq{
		SourceServerID:      3,
		DestinationServerID: 4,
		PerformChanges:      false, // dry run
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
}

func TestPushTemplate_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "internal server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.PushTemplate(&PushTemplateReq{
		SourceServerID:      99,
		DestinationServerID: 100,
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
