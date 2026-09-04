package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func envelope(data any) map[string]any {
	return map[string]any{
		"code":        200,
		"status":      "ok",
		"response_id": "SUCCESS",
		"message":     "",
		"data":        data,
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

func TestClientBasicAuthHeader(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Basic YWRtaW46cGZzZW5zZQ==" { // admin:pfsense
			t.Errorf("Authorization = %q", got)
		}
		writeJSON(t, w, 200, envelope([]any{}))
	})
	c, err := New(Config{URL: srv.URL, Username: "admin", Password: "pfsense"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.List(context.Background(), "/api/v2/firewall/aliases", nil); err != nil {
		t.Fatal(err)
	}
}

func TestClientAPIKeyHeader(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "sekret-key" {
			t.Errorf("X-API-Key = %q", got)
		}
		writeJSON(t, w, 200, envelope([]any{}))
	})
	c, err := New(Config{URL: srv.URL, APIKey: "sekret-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.List(context.Background(), "/api/v2/firewall/aliases", nil); err != nil {
		t.Fatal(err)
	}
}

func TestClientJWTAuthAndRefresh(t *testing.T) {
	var jwtCalls int
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/auth/jwt":
			jwtCalls++
			writeJSON(t, w, 200, envelope(map[string]any{"token": "test-jwt"}))
		case strings.HasPrefix(r.URL.Path, "/api/v2/"):
			if got := r.Header.Get("Authorization"); got != "Bearer test-jwt" {
				t.Errorf("Authorization = %q", got)
			}
			writeJSON(t, w, 200, envelope([]any{}))
		}
	})
	c, err := New(Config{URL: srv.URL, Username: "admin", Password: "pfsense", UseJWT: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// First call issues a JWT, second uses the cache.
	for i := 0; i < 2; i++ {
		if _, err := c.List(ctx, "/api/v2/firewall/aliases", nil); err != nil {
			t.Fatal(err)
		}
	}
	if jwtCalls != 1 {
		t.Errorf("expected 1 JWT call (cached), got %d", jwtCalls)
	}
	// Simulate an expired token -> refresh path.
	c.invalidateJWT()
	if _, err := c.List(ctx, "/api/v2/firewall/aliases", nil); err != nil {
		t.Fatal(err)
	}
	if jwtCalls != 2 {
		t.Errorf("expected 2 JWT calls after invalidate, got %d", jwtCalls)
	}
}

func TestClientJWTRefreshOn401(t *testing.T) {
	var jwtCalls int
	var allow bool
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/auth/jwt":
			jwtCalls++
			allow = true
			writeJSON(t, w, 200, envelope(map[string]any{"token": "fresh-jwt"}))
		case !allow:
			writeJSON(t, w, 401, map[string]any{"code": 401, "status": "unauthorized", "response_id": "AUTHENTICATION_FAILED", "message": "nope"})
		default:
			if got := r.Header.Get("Authorization"); got != "Bearer fresh-jwt" {
				t.Errorf("Authorization = %q", got)
			}
			writeJSON(t, w, 200, envelope([]any{}))
		}
	})
	c, err := New(Config{URL: srv.URL, Username: "admin", Password: "pfsense", UseJWT: true})
	if err != nil {
		t.Fatal(err)
	}
	// Prime the (stale) token by issuing once, then force server to reject it.
	c.jwtMu.Lock()
	c.jwtToken = "stale-jwt"
	c.jwtAt = time.Now()
	c.jwtMu.Unlock()
	if _, err := c.List(context.Background(), "/api/v2/firewall/aliases", nil); err != nil {
		t.Fatal(err)
	}
	if jwtCalls != 1 {
		t.Errorf("expected a refresh JWT call, got %d", jwtCalls)
	}
}

func TestClientForcesSynchronousApply(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode create body: %v", err)
			}
			if body["async"] != false {
				t.Errorf("create async = %#v, want false", body["async"])
			}
		case "/delete":
			if got := r.URL.Query().Get("async"); got != "false" {
				t.Errorf("delete async = %q, want false", got)
			}
		case "/apply":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode apply body: %v", err)
			}
			if body["async"] != false {
				t.Errorf("apply async = %#v, want false", body["async"])
			}
		}
		writeJSON(t, w, http.StatusOK, envelope(map[string]any{"id": 1}))
	})
	c, err := New(Config{URL: srv.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Create(context.Background(), "/create", map[string]any{"apply": true}); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(context.Background(), "/delete", Query{}.Set("apply", "true")); err != nil {
		t.Fatal(err)
	}
	if err := c.Apply(context.Background(), "/apply", nil); err != nil {
		t.Fatal(err)
	}
}

func TestClientAPIError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 422, map[string]any{
			"code": 422, "status": "unprocessable entity", "response_id": "VALIDATION_ERROR", "message": "Field `name` is required.",
		})
	})
	c, _ := New(Config{URL: srv.URL, APIKey: "k"})
	_, err := c.Create(context.Background(), "/api/v2/firewall/alias", map[string]any{})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.ResponseID != "VALIDATION_ERROR" || apiErr.Code != 422 {
		t.Errorf("unexpected APIError: %+v", apiErr)
	}
}

func TestClientGetNotFound(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 404, map[string]any{
			"code": 404, "status": "not found", "response_id": "NOT_FOUND", "message": "Object not found",
		})
	})
	c, _ := New(Config{URL: srv.URL, APIKey: "k"})
	_, err := c.Get(context.Background(), "/api/v2/firewall/alias", Query{}.Set("id", "99"))
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestClientListDecodesArray(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name__exact") != "webservers" {
			t.Errorf("missing query filter, got %v", r.URL.Query())
		}
		writeJSON(t, w, 200, envelope([]map[string]any{{"name": "webservers", "id": 0}}))
	})
	c, _ := New(Config{URL: srv.URL, APIKey: "k"})
	items, err := c.List(context.Background(), "/api/v2/firewall/aliases", Query{}.Set("name__exact", "webservers"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	var obj map[string]any
	if err := json.Unmarshal(items[0], &obj); err != nil {
		t.Fatal(err)
	}
	if obj["name"] != "webservers" {
		t.Errorf("unexpected object: %v", obj)
	}
}

func TestClientGetDecodesObject(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, envelope(map[string]any{"name": "webservers", "id": 3, "type": "host"}))
	})
	c, _ := New(Config{URL: srv.URL, APIKey: "k"})
	raw, err := c.Get(context.Background(), "/api/v2/firewall/alias", Query{}.Set("id", "3"))
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["name"] != "webservers" || obj["type"] != "host" {
		t.Errorf("unexpected object: %v", obj)
	}
}

func TestClientSerializesMutations(t *testing.T) {
	var active atomic.Int64
	var overlap atomic.Bool
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if active.Add(1) > 1 {
			overlap.Store(true)
		}
		time.Sleep(10 * time.Millisecond)
		active.Add(-1)
		writeJSON(t, w, http.StatusOK, envelope(map[string]any{"id": 1}))
	})
	c, err := New(Config{URL: srv.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}

	const workers = 12
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			ctx := context.Background()
			var callErr error
			switch i % 4 {
			case 0:
				_, callErr = c.Create(ctx, "/api/v2/test", map[string]any{"id": i})
			case 1:
				_, callErr = c.Update(ctx, "/api/v2/test", map[string]any{"id": i})
			case 2:
				callErr = c.Delete(ctx, "/api/v2/test", Query{}.Set("id", fmt.Sprint(i)))
			case 3:
				callErr = c.Apply(ctx, "/api/v2/test/apply", nil)
			}
			errs <- callErr
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if overlap.Load() {
		t.Fatal("mutation requests overlapped")
	}
}

func TestClientAllowsReadsDuringMutation(t *testing.T) {
	mutationStarted := make(chan struct{})
	releaseMutation := make(chan struct{})
	readStarted := make(chan struct{})
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			close(mutationStarted)
			<-releaseMutation
			writeJSON(t, w, http.StatusOK, envelope(map[string]any{"id": 1}))
			return
		}
		close(readStarted)
		writeJSON(t, w, http.StatusOK, envelope(map[string]any{"id": 1}))
	})
	c, err := New(Config{URL: srv.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}

	mutationDone := make(chan error, 1)
	go func() {
		_, err := c.Create(context.Background(), "/api/v2/test", map[string]any{"id": 1})
		mutationDone <- err
	}()
	<-mutationStarted

	readDone := make(chan error, 1)
	go func() {
		_, err := c.Get(context.Background(), "/api/v2/test", nil)
		readDone <- err
	}()
	select {
	case <-readStarted:
	case <-time.After(time.Second):
		close(releaseMutation)
		t.Fatal("read was blocked by an in-flight mutation")
	}
	close(releaseMutation)
	if err := <-mutationDone; err != nil {
		t.Fatal(err)
	}
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
}

func TestClientMutationWaitHonorsContext(t *testing.T) {
	mutationStarted := make(chan struct{})
	releaseMutation := make(chan struct{})
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		close(mutationStarted)
		<-releaseMutation
		writeJSON(t, w, http.StatusOK, envelope(map[string]any{"id": 1}))
	})
	c, err := New(Config{URL: srv.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}

	firstDone := make(chan error, 1)
	go func() {
		_, err := c.Create(context.Background(), "/api/v2/test", map[string]any{"id": 1})
		firstDone <- err
	}()
	<-mutationStarted

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = c.Update(ctx, "/api/v2/test", map[string]any{"id": 1})
	if !errors.Is(err, context.DeadlineExceeded) {
		close(releaseMutation)
		t.Fatalf("queued mutation error = %v, want context deadline exceeded", err)
	}

	close(releaseMutation)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
}

func TestNewRejectsBadURL(t *testing.T) {
	if _, err := New(Config{URL: "ftp://nope"}); err == nil {
		t.Fatal("expected error for non-http(s) URL")
	}
}
