package postmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestListInboundRules_Success(t *testing.T) {
	want := ListInboundRulesResp{
		TotalCount: 2,
		InboundRules: []InboundRuleResp{
			{ID: 1, Rule: "rule1.example.com"},
			{ID: 2, Rule: "rule2.example.com"},
		},
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "triggers/inboundrules") {
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

	got, err := api.ListInboundRules(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
	if len(got.InboundRules) != 2 {
		t.Errorf("len(InboundRules) = %d, want 2", len(got.InboundRules))
	}
	if got.InboundRules[0].Rule != "rule1.example.com" {
		t.Errorf("InboundRules[0].Rule = %q, want rule1.example.com", got.InboundRules[0].Rule)
	}
}

func TestListInboundRules_Pagination(t *testing.T) {
	tests := []struct {
		name   string
		count  int
		offset int
	}{
		{name: "first page", count: 5, offset: 0},
		{name: "second page", count: 5, offset: 5},
		{name: "large page", count: 100, offset: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				countParam := fmt.Sprintf("count=%d", tc.count)
				offsetParam := fmt.Sprintf("offset=%d", tc.offset)
				if !strings.Contains(req.URL.RawQuery, countParam) {
					t.Errorf("expected %s in query, got %s", countParam, req.URL.RawQuery)
				}
				if !strings.Contains(req.URL.RawQuery, offsetParam) {
					t.Errorf("expected %s in query, got %s", offsetParam, req.URL.RawQuery)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, ListInboundRulesResp{}),
				}, nil
			})))
			_, err := api.ListInboundRules(tc.count, tc.offset)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestListInboundRules_APIError(t *testing.T) {
	wantErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, wantErr),
		}, nil
	})))

	_, err := api.ListInboundRules(10, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, &wantErr) {
		t.Errorf("expected errors.Is(err, PostmarkErr{500}), got err=%v", err)
	}
}

func TestCreateInboundRule_Success(t *testing.T) {
	want := InboundRuleResp{ID: 42, Rule: "newrule.example.com"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "triggers/inboundrules") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["Rule"] != "newrule.example.com" {
			t.Errorf("expected Rule=newrule.example.com, got %v", body["Rule"])
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.CreateInboundRule("newrule.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("ID = %d, want 42", got.ID)
	}
	if got.Rule != "newrule.example.com" {
		t.Errorf("Rule = %q, want newrule.example.com", got.Rule)
	}
}

func TestCreateInboundRule_Various(t *testing.T) {
	tests := []struct {
		name string
		rule string
	}{
		{name: "domain rule", rule: "example.com"},
		{name: "subdomain rule", rule: "mail.example.com"},
		{name: "wildcard rule", rule: "*.example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expectedRule := tc.rule
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				if body["Rule"] != expectedRule {
					t.Errorf("expected Rule=%q, got %v", expectedRule, body["Rule"])
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, InboundRuleResp{ID: 1, Rule: expectedRule}),
				}, nil
			})))
			got, err := api.CreateInboundRule(tc.rule)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Rule != tc.rule {
				t.Errorf("Rule = %q, want %q", got.Rule, tc.rule)
			}
		})
	}
}

func TestCreateInboundRule_APIError(t *testing.T) {
	wantErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, wantErr),
		}, nil
	})))

	_, err := api.CreateInboundRule("bad.example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, &wantErr) {
		t.Errorf("expected errors.Is(err, PostmarkErr{500}), got err=%v", err)
	}
}

func TestDeleteInboundRule_Success(t *testing.T) {
	want := DeleteResp{Message: "Trigger deleted."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "triggers/inboundrules/42") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteInboundRule(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Trigger deleted." {
		t.Errorf("Message = %q, want Trigger deleted.", got.Message)
	}
}

func TestDeleteInboundRule_NotFound(t *testing.T) {
	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       jsonBody(t, PostmarkErr{ErrorCode: 404, Message: "trigger not found"}),
		}, nil
	})))

	_, err := api.DeleteInboundRule(9999)
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true, got err=%v", err)
	}
}

func TestDeleteInboundRule_PathContainsTriggerID(t *testing.T) {
	tests := []struct {
		name      string
		triggerID int64
		wantPath  string
	}{
		{name: "trigger id 1", triggerID: 1, wantPath: "/triggers/inboundrules/1"},
		{name: "trigger id 999", triggerID: 999, wantPath: "/triggers/inboundrules/999"},
		{name: "large trigger id", triggerID: 123456789, wantPath: "/triggers/inboundrules/123456789"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != tc.wantPath {
					t.Errorf("path = %s, want %s", req.URL.Path, tc.wantPath)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       jsonBody(t, DeleteResp{Message: "Trigger deleted."}),
				}, nil
			})))
			_, err := api.DeleteInboundRule(tc.triggerID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
