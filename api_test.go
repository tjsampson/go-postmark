package postmark

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// roundTripFunc is a convenience type that lets a plain function satisfy
// the http.RoundTripper interface, enabling lightweight HTTP mocking in tests.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// newTestClient returns an *http.Client whose transport is replaced by fn,
// so no real network connections are made during tests.
func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

// jsonBody is a helper that serialises v and wraps it in an io.ReadCloser.
func jsonBody(t *testing.T, v interface{}) io.ReadCloser {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("jsonBody: %v", err)
	}
	return io.NopCloser(strings.NewReader(string(b)))
}

// ---- New / Option tests --------------------------------------------------------

func TestNew_Defaults(t *testing.T) {
	api := New()
	if api.baseHost != "https://api.postmarkapp.com" {
		t.Errorf("unexpected baseHost: %s", api.baseHost)
	}
	if api.timeout != 10*time.Second {
		t.Errorf("unexpected timeout: %v", api.timeout)
	}
	if api.client == nil {
		t.Error("client must not be nil")
	}
}

func TestNew_WithAPITokenOpt(t *testing.T) {
	api := New(APITokenOpt("tok-123"))
	if api.token != "tok-123" {
		t.Errorf("expected token tok-123, got %s", api.token)
	}
}

func TestNew_WithTimeoutOpt(t *testing.T) {
	api := New(TimeoutOpt(5 * time.Second))
	if api.timeout != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v", api.timeout)
	}
}

func TestNew_WithHTTPClientOpt(t *testing.T) {
	custom := &http.Client{Timeout: 3 * time.Second}
	api := New(HTTPClientOpt(custom))
	if api.client != custom {
		t.Error("expected the custom http.Client to be set")
	}
}

// TestTimeoutOpt_AppliedToClient verifies that TimeoutOpt propagates the
// timeout to the underlying *http.Client and does not mutate the default
// client singleton.
func TestTimeoutOpt_AppliedToClient(t *testing.T) {
	api := New(TimeoutOpt(42 * time.Second))

	hc, ok := api.client.(*http.Client)
	if !ok {
		t.Fatal("expected api.client to be *http.Client")
	}
	if hc.Timeout != 42*time.Second {
		t.Errorf("expected client timeout 42s, got %v", hc.Timeout)
	}

	// The default client singleton must remain unmodified (10 s).
	defaultClient := &http.Client{Timeout: defaultTimeOut}
	if defaultClient.Timeout != 10*time.Second {
		t.Errorf("default timeout should still be 10s, got %v", defaultClient.Timeout)
	}

	// More directly: verify that calling New with TimeoutOpt does not affect
	// a freshly-created instance that uses the default timeout.
	api2 := New()
	hc2, ok := api2.client.(*http.Client)
	if !ok {
		t.Fatal("expected api2.client to be *http.Client")
	}
	if hc2.Timeout != 10*time.Second {
		t.Errorf("default client timeout should be 10s after TimeoutOpt on another instance, got %v", hc2.Timeout)
	}
}

// ---- PostmarkErr ---------------------------------------------------------------

func TestPostmarkErr_Error(t *testing.T) {
	pe := PostmarkErr{ErrorCode: 422, Message: "Invalid data"}
	want := "Invalid data Error Code=422"
	if pe.Error() != want {
		t.Errorf("Error() = %q, want %q", pe.Error(), want)
	}
}

func TestPostmarkErr_Code(t *testing.T) {
	pe := PostmarkErr{ErrorCode: 404}
	if pe.Code() != 404 {
		t.Errorf("Code() = %d, want 404", pe.Code())
	}
}

func TestNewError(t *testing.T) {
	pe := NewError(409, "already exists: %s", "my-server")
	if pe.ErrorCode != 409 {
		t.Errorf("ErrorCode = %d, want 409", pe.ErrorCode)
	}
	if pe.Message != "already exists: my-server" {
		t.Errorf("Message = %q", pe.Message)
	}
}

// ---- CreateServer --------------------------------------------------------------

func TestCreateServer_Success(t *testing.T) {
	want := ServerResp{ID: 1, Name: "Test Server", Color: "blue"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/servers") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.CreateServer(&CreateServerReq{Name: "Test Server", Color: "blue"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCreateServer_APIError(t *testing.T) {
	pmErr := PostmarkErr{ErrorCode: 500, Message: "server error"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       jsonBody(t, pmErr),
		}, nil
	})))

	_, err := api.CreateServer(&CreateServerReq{Name: "Bad"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ---- ReadServer ----------------------------------------------------------------

func TestReadServer_Success(t *testing.T) {
	want := ServerResp{ID: 42, Name: "My Server"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ReadServer("42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("ID = %d, want 42", got.ID)
	}
}

// ---- ListServers ---------------------------------------------------------------

func TestListServers_Success(t *testing.T) {
	want := ListServerResp{TotalCount: 2, Servers: []ServerResp{{ID: 1}, {ID: 2}}}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.RawQuery, "count=10") {
			t.Errorf("expected count param, query=%s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.ListServers("10", "0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
}

// ---- DeleteServer --------------------------------------------------------------

func TestDeleteServer_Success(t *testing.T) {
	want := DeleteResp{Message: "Server deleted."}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.DeleteServer("99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message != "Server deleted." {
		t.Errorf("Message = %q", got.Message)
	}
}

// ---- UpdateServer --------------------------------------------------------------

func TestUpdateServer_Success(t *testing.T) {
	want := ServerResp{ID: 7, Name: "Renamed"}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       jsonBody(t, want),
		}, nil
	})))

	got, err := api.UpdateServer("7", &UpdateServerReq{Name: "Renamed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Renamed" {
		t.Errorf("Name = %q, want Renamed", got.Name)
	}
}

// ---- Do / body-close -----------------------------------------------------------

// trackingReadCloser is an io.ReadCloser that records whether Close was called.
type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (t *trackingReadCloser) Close() error {
	t.closed = true
	return nil
}

// TestDo_BodyIsClosed verifies that Do closes the HTTP response body after
// reading it, preventing connection leaks in the net/http transport pool.
func TestDo_BodyIsClosed(t *testing.T) {
	tracker := &trackingReadCloser{
		Reader: strings.NewReader(`{}`),
	}

	api := New(HTTPClientOpt(newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       tracker,
		}, nil
	})))

	req, err := http.NewRequest(http.MethodGet, "https://api.postmarkapp.com/servers/1", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	_, err = api.Do(req)
	if err != nil {
		t.Fatalf("unexpected error from Do: %v", err)
	}

	if !tracker.closed {
		t.Error("expected resp.Body.Close() to be called, but it was not")
	}
}
