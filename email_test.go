package postmark

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ---- SendEmail -----------------------------------------------------------------

func TestSendEmail_Success(t *testing.T) {
	submittedAt, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")

	tests := []struct {
		name string
		req  *EmailReq
		want EmailResp
	}{
		{
			name: "plain text email",
			req: &EmailReq{
				From:     "sender@example.com",
				To:       "recipient@example.com",
				Subject:  "Hello",
				TextBody: "Hello, world!",
			},
			want: EmailResp{
				To:          "recipient@example.com",
				SubmittedAt: submittedAt,
				MessageID:   "abc-123",
				ErrorCode:   0,
				Message:     "OK",
			},
		},
		{
			name: "html email with attachments",
			req: &EmailReq{
				From:     "sender@example.com",
				To:       "recipient@example.com",
				Subject:  "HTML Email",
				HTMLBody: "<h1>Hello</h1>",
				Attachments: []Attachment{
					{Name: "file.txt", Content: "aGVsbG8=", ContentType: "text/plain"},
				},
			},
			want: EmailResp{
				To:          "recipient@example.com",
				SubmittedAt: submittedAt,
				MessageID:   "def-456",
				ErrorCode:   0,
				Message:     "OK",
			},
		},
		{
			name: "email with metadata and headers",
			req: &EmailReq{
				From:     "sender@example.com",
				To:       "recipient@example.com",
				Subject:  "Rich Email",
				TextBody: "Body",
				Headers: []Header{
					{Name: "X-Custom", Value: "custom-value"},
				},
				Metadata: map[string]string{"key": "value"},
			},
			want: EmailResp{
				To:          "recipient@example.com",
				SubmittedAt: submittedAt,
				MessageID:   "ghi-789",
				ErrorCode:   0,
				Message:     "OK",
			},
		},
		{
			name: "email with explicit TrackOpens false",
			req: &EmailReq{
				From:       "sender@example.com",
				To:         "recipient@example.com",
				Subject:    "No tracking",
				TextBody:   "Body",
				TrackOpens: boolPtr(false),
			},
			want: EmailResp{
				To:        "recipient@example.com",
				MessageID: "jkl-012",
				ErrorCode: 0,
				Message:   "OK",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("test-server-token"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					// Verify method and path.
					if req.Method != http.MethodPost {
						t.Errorf("expected POST, got %s", req.Method)
					}
					// Path must end with /email AND must NOT end with /email/batch.
					if !strings.HasSuffix(req.URL.Path, "/email") || strings.HasSuffix(req.URL.Path, "/email/batch") {
						t.Errorf("unexpected path: %s", req.URL.Path)
					}
					// Verify server token header is set (not account token).
					if got := req.Header.Get("X-Postmark-Server-Token"); got != "test-server-token" {
						t.Errorf("X-Postmark-Server-Token = %q, want %q", got, "test-server-token")
					}
					if req.Header.Get("X-Postmark-Account-Token") != "" {
						t.Errorf("unexpected X-Postmark-Account-Token header on email request")
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, tc.want),
					}, nil
				})),
			)

			got, err := api.SendEmail(tc.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.To != tc.want.To {
				t.Errorf("To = %q, want %q", got.To, tc.want.To)
			}
			if got.MessageID != tc.want.MessageID {
				t.Errorf("MessageID = %q, want %q", got.MessageID, tc.want.MessageID)
			}
			if got.ErrorCode != tc.want.ErrorCode {
				t.Errorf("ErrorCode = %d, want %d", got.ErrorCode, tc.want.ErrorCode)
			}
			if got.Message != tc.want.Message {
				t.Errorf("Message = %q, want %q", got.Message, tc.want.Message)
			}
		})
	}
}

// TestSendEmail_PostmarkErrorCode verifies that SendEmail returns a non-nil
// error when Postmark responds with HTTP 200 but a non-zero ErrorCode in the
// body. This is a common Postmark pattern for API-level validation failures.
func TestSendEmail_PostmarkErrorCode(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   EmailResp
		wantErrContain string
	}{
		{
			name: "invalid recipient",
			responseBody: EmailResp{
				ErrorCode: 300,
				Message:   "Invalid email request",
			},
			wantErrContain: "Invalid email request",
		},
		{
			name: "inactive recipient",
			responseBody: EmailResp{
				ErrorCode: 406,
				Message:   "You tried to send to a recipient that has been marked as inactive",
			},
			wantErrContain: "inactive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("test-server-token"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, tc.responseBody),
					}, nil
				})),
			)

			_, err := api.SendEmail(&EmailReq{
				From:    "sender@example.com",
				To:      "recipient@example.com",
				Subject: "Test",
			})

			if err == nil {
				t.Fatal("expected an error for non-zero ErrorCode, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrContain) {
				t.Errorf("expected error to contain %q, got %q", tc.wantErrContain, err.Error())
			}
			// The returned error must be (or wrap) a *PostmarkErr.
			var pmErr *PostmarkErr
			if !errors.As(err, &pmErr) {
				t.Errorf("expected error to be *PostmarkErr, got %T: %v", err, err)
			} else if pmErr.ErrorCode != tc.responseBody.ErrorCode {
				t.Errorf("PostmarkErr.ErrorCode = %d, want %d", pmErr.ErrorCode, tc.responseBody.ErrorCode)
			}
		})
	}
}

func TestSendEmail_APIError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   interface{}
		wantErrIs      error
		wantErrContain string
	}{
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			responseBody: PostmarkErr{
				ErrorCode: 500,
				Message:   "Internal server error",
			},
			wantErrContain: "Internal server error",
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			responseBody: PostmarkErr{
				ErrorCode: 401,
				Message:   "Unauthorized: Missing or incorrect API token",
			},
			wantErrContain: "Unauthorized",
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			responseBody: PostmarkErr{
				ErrorCode: 404,
				Message:   "Not found",
			},
			wantErrIs: ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("test-server-token"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tc.statusCode,
						Body:       jsonBody(t, tc.responseBody),
					}, nil
				})),
			)

			_, err := api.SendEmail(&EmailReq{
				From:    "sender@example.com",
				To:      "recipient@example.com",
				Subject: "Test",
			})

			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if tc.wantErrIs != nil && !errors.Is(err, tc.wantErrIs) {
				t.Errorf("expected errors.Is(err, %v) to be true, got err=%v", tc.wantErrIs, err)
			}
			if tc.wantErrContain != "" && !strings.Contains(err.Error(), tc.wantErrContain) {
				t.Errorf("expected error to contain %q, got %q", tc.wantErrContain, err.Error())
			}
		})
	}
}

// TestSendEmail_MissingServerToken verifies that SendEmail returns an error
// immediately (without making an HTTP request) when no server token is configured.
func TestSendEmail_MissingServerToken(t *testing.T) {
	api := New(
		// Deliberately omit ServerTokenOpt.
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP client should not be called when serverToken is empty")
			return nil, nil
		})),
	)

	_, err := api.SendEmail(&EmailReq{
		From:    "sender@example.com",
		To:      "recipient@example.com",
		Subject: "Test",
	})
	if err == nil {
		t.Fatal("expected an error for missing server token, got nil")
	}
	if !strings.Contains(err.Error(), "server token not configured") {
		t.Errorf("expected error to mention missing server token, got: %v", err)
	}
}

// ---- SendEmailBatch ------------------------------------------------------------

func TestSendEmailBatch_Success(t *testing.T) {
	tests := []struct {
		name string
		reqs []*EmailReq
		want BatchEmailResp
	}{
		{
			name: "single message batch",
			reqs: []*EmailReq{
				{
					From:     "sender@example.com",
					To:       "recipient@example.com",
					Subject:  "Hello",
					TextBody: "Hello, world!",
				},
			},
			want: BatchEmailResp{
				{
					To:        "recipient@example.com",
					MessageID: "batch-msg-1",
					ErrorCode: 0,
					Message:   "OK",
				},
			},
		},
		{
			name: "multiple messages batch",
			reqs: []*EmailReq{
				{From: "a@example.com", To: "b@example.com", Subject: "Msg 1", TextBody: "Body 1"},
				{From: "a@example.com", To: "c@example.com", Subject: "Msg 2", TextBody: "Body 2"},
				{From: "a@example.com", To: "d@example.com", Subject: "Msg 3", HTMLBody: "<p>Body 3</p>"},
			},
			want: BatchEmailResp{
				{To: "b@example.com", MessageID: "batch-1", ErrorCode: 0, Message: "OK"},
				{To: "c@example.com", MessageID: "batch-2", ErrorCode: 0, Message: "OK"},
				{To: "d@example.com", MessageID: "batch-3", ErrorCode: 0, Message: "OK"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("test-server-token"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					// Verify method and path.
					if req.Method != http.MethodPost {
						t.Errorf("expected POST, got %s", req.Method)
					}
					if !strings.HasSuffix(req.URL.Path, "/email/batch") {
						t.Errorf("unexpected path: %s", req.URL.Path)
					}
					// Verify server token header.
					if got := req.Header.Get("X-Postmark-Server-Token"); got != "test-server-token" {
						t.Errorf("X-Postmark-Server-Token = %q, want %q", got, "test-server-token")
					}
					if req.Header.Get("X-Postmark-Account-Token") != "" {
						t.Errorf("unexpected X-Postmark-Account-Token header on batch email request")
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, tc.want),
					}, nil
				})),
			)

			got, err := api.SendEmailBatch(tc.reqs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("batch response length = %d, want %d", len(got), len(tc.want))
			}
			for i, item := range got {
				if item.To != tc.want[i].To {
					t.Errorf("[%d] To = %q, want %q", i, item.To, tc.want[i].To)
				}
				if item.MessageID != tc.want[i].MessageID {
					t.Errorf("[%d] MessageID = %q, want %q", i, item.MessageID, tc.want[i].MessageID)
				}
				if item.ErrorCode != tc.want[i].ErrorCode {
					t.Errorf("[%d] ErrorCode = %d, want %d", i, item.ErrorCode, tc.want[i].ErrorCode)
				}
			}
		})
	}
}

// TestSendEmailBatch_EmptySlice asserts that SendEmailBatch returns an error
// immediately when an empty slice is supplied, without making any HTTP request.
func TestSendEmailBatch_EmptySlice(t *testing.T) {
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP client should not be called for an empty batch")
			return nil, nil
		})),
	)

	_, err := api.SendEmailBatch([]*EmailReq{})
	if err == nil {
		t.Fatal("expected an error for empty batch, got nil")
	}
	if !strings.Contains(err.Error(), "at least one message") {
		t.Errorf("expected error message to mention empty batch, got: %v", err)
	}
}

// TestSendEmailBatch_ExceedsMaxSize asserts that SendEmailBatch returns an
// error immediately when more than 500 messages are supplied, without making
// any HTTP request.
func TestSendEmailBatch_ExceedsMaxSize(t *testing.T) {
	// Build a slice of 501 minimal EmailReq values.
	reqs := make([]*EmailReq, 501)
	for i := range reqs {
		reqs[i] = &EmailReq{From: "a@example.com", To: "b@example.com", Subject: "Test"}
	}

	// The mock transport must never be called; if it is, fail loudly.
	api := New(
		ServerTokenOpt("test-server-token"),
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP client should not be called when batch size exceeds 500")
			return nil, nil
		})),
	)

	_, err := api.SendEmailBatch(reqs)
	if err == nil {
		t.Fatal("expected an error for batch size > 500, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("expected error message to mention batch size limit, got: %v", err)
	}
}

// TestSendEmailBatch_ExactlyMaxSize asserts that a batch of exactly 500
// messages is accepted (boundary condition: len == 500 must succeed).
func TestSendEmailBatch_ExactlyMaxSize(t *testing.T) {
	reqs := make([]*EmailReq, 500)
	for i := range reqs {
		reqs[i] = &EmailReq{From: "a@example.com", To: "b@example.com", Subject: "Test"}
	}

	// Build a matching response slice of 500 elements.
	wantResp := make(BatchEmailResp, 500)
	for i := range wantResp {
		wantResp[i] = EmailResp{To: "b@example.com", ErrorCode: 0, Message: "OK"}
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

	got, err := api.SendEmailBatch(reqs)
	if err != nil {
		t.Fatalf("unexpected error for batch of exactly 500: %v", err)
	}
	if len(got) != 500 {
		t.Errorf("batch response length = %d, want 500", len(got))
	}
}

// TestSendEmailBatch_MissingServerToken verifies that SendEmailBatch returns
// an error immediately (without making an HTTP request) when no server token
// is configured.
func TestSendEmailBatch_MissingServerToken(t *testing.T) {
	api := New(
		// Deliberately omit ServerTokenOpt.
		HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP client should not be called when serverToken is empty")
			return nil, nil
		})),
	)

	_, err := api.SendEmailBatch([]*EmailReq{
		{From: "a@example.com", To: "b@example.com", Subject: "Test"},
	})
	if err == nil {
		t.Fatal("expected an error for missing server token, got nil")
	}
	if !strings.Contains(err.Error(), "server token not configured") {
		t.Errorf("expected error to mention missing server token, got: %v", err)
	}
}

func TestSendEmailBatch_APIError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   interface{}
		wantErrIs      error
		wantErrContain string
	}{
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			responseBody: PostmarkErr{
				ErrorCode: 500,
				Message:   "Internal server error",
			},
			wantErrContain: "Internal server error",
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			responseBody: PostmarkErr{
				ErrorCode: 404,
				Message:   "Not found",
			},
			wantErrIs: ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("test-server-token"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tc.statusCode,
						Body:       jsonBody(t, tc.responseBody),
					}, nil
				})),
			)

			_, err := api.SendEmailBatch([]*EmailReq{
				{From: "a@example.com", To: "b@example.com", Subject: "Test"},
			})

			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if tc.wantErrIs != nil && !errors.Is(err, tc.wantErrIs) {
				t.Errorf("expected errors.Is(err, %v) to be true, got err=%v", tc.wantErrIs, err)
			}
			if tc.wantErrContain != "" && !strings.Contains(err.Error(), tc.wantErrContain) {
				t.Errorf("expected error to contain %q, got %q", tc.wantErrContain, err.Error())
			}
		})
	}
}

// ---- TrackOpens serialisation --------------------------------------------------

// TestTrackOpens_Serialisation verifies that an explicit false value for
// TrackOpens is serialised into the JSON request body (i.e. not silently
// dropped by omitempty), and that a nil value omits the field entirely.
func TestTrackOpens_Serialisation(t *testing.T) {
	tests := []struct {
		name       string
		trackOpens *bool
		wantInBody string
		wantAbsent string
	}{
		{
			name:       "explicit false is serialised",
			trackOpens: boolPtr(false),
			wantInBody: `"TrackOpens":false`,
		},
		{
			name:       "explicit true is serialised",
			trackOpens: boolPtr(true),
			wantInBody: `"TrackOpens":true`,
		},
		{
			name:       "nil omits field",
			trackOpens: nil,
			wantAbsent: `"TrackOpens"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := New(
				ServerTokenOpt("test-server-token"),
				HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
					// Read and inspect the serialised request body.
					body := make([]byte, 4096)
					n, _ := req.Body.Read(body)
					bodyStr := string(body[:n])

					if tc.wantInBody != "" && !strings.Contains(bodyStr, tc.wantInBody) {
						t.Errorf("expected body to contain %q, got: %s", tc.wantInBody, bodyStr)
					}
					if tc.wantAbsent != "" && strings.Contains(bodyStr, tc.wantAbsent) {
						t.Errorf("expected body NOT to contain %q, got: %s", tc.wantAbsent, bodyStr)
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       jsonBody(t, EmailResp{To: "b@example.com", Message: "OK"}),
					}, nil
				})),
			)

			_, err := api.SendEmail(&EmailReq{
				From:       "a@example.com",
				To:         "b@example.com",
				Subject:    "Test",
				TrackOpens: tc.trackOpens,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---- ServerTokenOpt ------------------------------------------------------------

func TestNew_WithServerTokenOpt(t *testing.T) {
	api := New(ServerTokenOpt("srv-token-xyz"))
	if api.serverToken != "srv-token-xyz" {
		t.Errorf("expected serverToken srv-token-xyz, got %s", api.serverToken)
	}
}

// TestNewServerRequest_UsesServerToken verifies that newServerRequest sets
// X-Postmark-Server-Token and does not set X-Postmark-Account-Token.
func TestNewServerRequest_UsesServerToken(t *testing.T) {
	api := New(ServerTokenOpt("my-server-token"), APITokenOpt("my-account-token"))

	req, err := api.newServerRequest(http.MethodPost, "email", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := req.Header.Get("X-Postmark-Server-Token"); got != "my-server-token" {
		t.Errorf("X-Postmark-Server-Token = %q, want %q", got, "my-server-token")
	}
	if got := req.Header.Get("X-Postmark-Account-Token"); got != "" {
		t.Errorf("X-Postmark-Account-Token should not be set, got %q", got)
	}
}

// TestNewServerRequest_EmptyToken verifies that newServerRequest returns an
// error (without building a request) when no server token has been configured.
func TestNewServerRequest_EmptyToken(t *testing.T) {
	api := New() // no ServerTokenOpt

	_, err := api.newServerRequest(http.MethodPost, "email", nil)
	if err == nil {
		t.Fatal("expected an error for empty server token, got nil")
	}
	if !strings.Contains(err.Error(), "server token not configured") {
		t.Errorf("expected error to mention missing server token, got: %v", err)
	}
}

// TestNewRequest_UsesAccountToken verifies that the existing newRequest helper
// still uses X-Postmark-Account-Token (unchanged behaviour).
func TestNewRequest_UsesAccountToken(t *testing.T) {
	api := New(APITokenOpt("my-account-token"), ServerTokenOpt("my-server-token"))

	req, err := api.newRequest(http.MethodGet, "servers/1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := req.Header.Get("X-Postmark-Account-Token"); got != "my-account-token" {
		t.Errorf("X-Postmark-Account-Token = %q, want %q", got, "my-account-token")
	}
	if got := req.Header.Get("X-Postmark-Server-Token"); got != "" {
		t.Errorf("X-Postmark-Server-Token should not be set on account request, got %q", got)
	}
}
