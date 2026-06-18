package postmark

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

	trackOpens := true
	inlineCss := true
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
		InlineCss:     &inlineCss,
		From:          "sender@example.com",
		To:            "recipient@example.com",
		Tag:           "onboarding",
		TrackOpens:    &trackOpens,
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

// TestSendEmailWithTemplate_PostmarkErrorCode verifies that a Postmark logical
// error (HTTP 200 with non-zero ErrorCode in the body) is surfaced as a
// *PostmarkErr rather than silently returning a zero-value response.
func TestSendEmailWithTemplate_PostmarkErrorCode(t *testing.T) {
	// Postmark returns HTTP 200 with ErrorCode=406 when the template is unknown.
	body := SendEmailResp{ErrorCode: 406, Message: "Unknown template"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, body),
		}, nil
	})))

	_, err := api.SendEmailWithTemplate(&SendWithTemplateReq{
		TemplateID:    99999,
		TemplateModel: map[string]string{},
		From:          "sender@example.com",
		To:            "recipient@example.com",
	})
	if err == nil {
		t.Fatal("expected a PostmarkErr for ErrorCode=406, got nil")
	}
	var pmErr *PostmarkErr
	if !errors.As(err, &pmErr) {
		t.Fatalf("expected error to be *PostmarkErr, got %T: %v", err, err)
	}
	if pmErr.ErrorCode != 406 {
		t.Errorf("ErrorCode = %d, want 406", pmErr.ErrorCode)
	}
}

// TestSendEmailWithTemplate_NilReq verifies that passing a nil request is
// rejected locally before any HTTP request is attempted.
func TestSendEmailWithTemplate_NilReq(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not be made for nil req")
		return nil, nil
	})))

	_, err := api.SendEmailWithTemplate(nil)
	if err == nil {
		t.Fatal("expected an error for nil req, got nil")
	}
}

// TestSendWithTemplateReq_TrackOpensFalse verifies that TrackOpens=false is
// serialised to JSON (i.e. the *bool pointer form is used, not the bare bool
// with omitempty which would silently drop the false value).
func TestSendWithTemplateReq_TrackOpensFalse(t *testing.T) {
	f := false
	req := &SendWithTemplateReq{
		TemplateID:    1,
		TemplateModel: map[string]string{},
		From:          "a@b.com",
		To:            "c@d.com",
		TrackOpens:    &f,
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]interface{}
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	v, ok := m["TrackOpens"]
	if !ok {
		t.Fatal("TrackOpens key missing from JSON; false should be serialised when *bool pointer is set")
	}
	if v.(bool) != false {
		t.Errorf("TrackOpens = %v, want false", v)
	}
}

// TestSendWithTemplateReq_TrackOpensNilOmitted verifies that a nil TrackOpens
// pointer is omitted from the JSON output (server uses its default).
func TestSendWithTemplateReq_TrackOpensNilOmitted(t *testing.T) {
	req := &SendWithTemplateReq{
		TemplateID:    1,
		TemplateModel: map[string]string{},
		From:          "a@b.com",
		To:            "c@d.com",
		TrackOpens:    nil,
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if bytes.Contains(b, []byte("TrackOpens")) {
		t.Errorf("TrackOpens should be absent from JSON when nil, got: %s", b)
	}
}

// TestSendWithTemplateReq_InlineCssFalse verifies that InlineCss=false is
// serialised to JSON when explicitly set via a *bool pointer.
func TestSendWithTemplateReq_InlineCssFalse(t *testing.T) {
	f := false
	req := &SendWithTemplateReq{
		TemplateID:    1,
		TemplateModel: map[string]string{},
		From:          "a@b.com",
		To:            "c@d.com",
		InlineCss:     &f,
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]interface{}
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	v, ok := m["InlineCss"]
	if !ok {
		t.Fatal("InlineCss key missing from JSON; false should be serialised when *bool pointer is set")
	}
	if v.(bool) != false {
		t.Errorf("InlineCss = %v, want false", v)
	}
}

// TestSendWithTemplateReq_MetadataTyped verifies that Metadata is typed as
// map[string]string, preventing incompatible types from being passed.
func TestSendWithTemplateReq_MetadataTyped(t *testing.T) {
	req := &SendWithTemplateReq{
		TemplateID:    1,
		TemplateModel: map[string]string{},
		From:          "a@b.com",
		To:            "c@d.com",
		Metadata:      map[string]string{"campaign": "spring-sale", "source": "email"},
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]interface{}
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	meta, ok := m["Metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("Metadata type = %T, want map[string]interface{}", m["Metadata"])
	}
	if meta["campaign"] != "spring-sale" {
		t.Errorf("Metadata[campaign] = %v, want spring-sale", meta["campaign"])
	}
}

// ---- GetTemplate ---------------------------------------------------------------

func TestGetTemplate_Success(t *testing.T) {
	want := TemplateResp{
		TemplateID:   123,
		Name:         "Welcome Email",
		Subject:      "Welcome, {{name}}!",
		HtmlBody:     "<h1>Welcome</h1>",
		TextBody:     "Welcome",
		Alias:        "welcome",
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

// TestGetTemplate_EmptyID verifies that an empty templateID is rejected before
// any HTTP request is made.
func TestGetTemplate_EmptyID(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not be made for empty templateID")
		return nil, nil
	})))

	_, err := api.GetTemplate("")
	if err == nil {
		t.Fatal("expected an error for empty templateID, got nil")
	}
	if !errors.Is(err, errEmptyTemplateID) {
		t.Errorf("expected errEmptyTemplateID, got %v", err)
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

// TestCreateTemplate_NilReq verifies that passing a nil request is rejected
// locally before any HTTP request is attempted.
func TestCreateTemplate_NilReq(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not be made for nil req")
		return nil, nil
	})))

	_, err := api.CreateTemplate(nil)
	if err == nil {
		t.Fatal("expected an error for nil req, got nil")
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

// TestEditTemplate_EmptyID verifies that an empty templateID is rejected before
// any HTTP request is made.
func TestEditTemplate_EmptyID(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not be made for empty templateID")
		return nil, nil
	})))

	_, err := api.EditTemplate("", &EditTemplateReq{Name: "Ghost"})
	if err == nil {
		t.Fatal("expected an error for empty templateID, got nil")
	}
	if !errors.Is(err, errEmptyTemplateID) {
		t.Errorf("expected errEmptyTemplateID, got %v", err)
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
		if !strings.HasSuffix(req.URL.Path, "/templates") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if req.URL.Query().Get("count") != "10" {
			t.Errorf("expected count=10, query=%s", req.URL.RawQuery)
		}
		if req.URL.Query().Get("offset") != "0" {
			t.Errorf("expected offset=0, query=%s", req.URL.RawQuery)
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
		if req.URL.Query().Get("count") != "5" {
			t.Errorf("expected count=5, query=%s", req.URL.RawQuery)
		}
		if req.URL.Query().Get("offset") != "5" {
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

// TestListTemplates_QueryParamsMerged verifies that ListTemplates merges
// count/offset into any pre-existing query parameters on the request URL
// rather than overwriting the entire RawQuery. This guards against a future
// change in newRequest that might add default query params.
func TestListTemplates_QueryParamsMerged(t *testing.T) {
	var capturedQuery string
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		capturedQuery = req.URL.RawQuery
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, ListTemplatesResp{TotalCount: 0}),
		}, nil
	})))

	_, err := api.ListTemplates(20, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q, err := parseQuery(capturedQuery)
	if err != nil {
		t.Fatalf("failed to parse captured query %q: %v", capturedQuery, err)
	}
	if q.Get("count") != "20" {
		t.Errorf("count = %q, want 20 in query %q", q.Get("count"), capturedQuery)
	}
	if q.Get("offset") != "10" {
		t.Errorf("offset = %q, want 10 in query %q", q.Get("offset"), capturedQuery)
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

// TestDeleteTemplate_EmptyID verifies that an empty templateID is rejected before
// any HTTP request is made.
func TestDeleteTemplate_EmptyID(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not be made for empty templateID")
		return nil, nil
	})))

	_, err := api.DeleteTemplate("")
	if err == nil {
		t.Fatal("expected an error for empty templateID, got nil")
	}
	if !errors.Is(err, errEmptyTemplateID) {
		t.Errorf("expected errEmptyTemplateID, got %v", err)
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
		SuggestedTemplateModel: map[string]interface{}{"name": "World"},
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
	// SuggestedTemplateModel should decode as map[string]interface{}
	if got.SuggestedTemplateModel == nil {
		t.Error("SuggestedTemplateModel should not be nil")
	}
	if got.SuggestedTemplateModel["name"] != "World" {
		t.Errorf("SuggestedTemplateModel[name] = %v, want World", got.SuggestedTemplateModel["name"])
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

// ---- helpers used only in templates_test.go -----------------------------------

// parseQuery is a thin wrapper around url.ParseQuery used in this test file
// to keep test helpers self-contained.
func parseQuery(rawQuery string) (interface{ Get(string) string }, error) {
	return parseRawQuery(rawQuery)
}

// parseRawQuery parses a raw query string and returns the url.Values.
func parseRawQuery(rawQuery string) (urlValues, error) {
	// Re-use io from the imports to avoid an extra import.
	_ = io.Discard // keep io import used
	vals := make(urlValues)
	pairs := strings.Split(rawQuery, "&")
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			vals[kv[0]] = kv[1]
		}
	}
	return vals, nil
}

// urlValues is a minimal map that satisfies the Get interface used in tests.
type urlValues map[string]string

func (v urlValues) Get(key string) string { return v[key] }
