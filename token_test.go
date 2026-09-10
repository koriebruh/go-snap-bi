package snap

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock is an injectable, manually-advanced clock for TokenManager.Now.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestParseExpiresIn(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{"typical value", "900", 900 * time.Second, false},
		{"empty", "", 0, true},
		{"not a number", "soon", 0, true},
		{"zero rejected", "0", 0, true},
		{"negative rejected", "-5", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExpiresIn(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseExpiresIn(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseExpiresIn(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestCacheTTL(t *testing.T) {
	tests := []struct {
		name      string
		expiresIn time.Duration
		want      time.Duration
	}{
		{"typical: margin fits comfortably", 900 * time.Second, 900*time.Second - tokenSafetyMargin},
		{"small expiresIn: margin clamped to half", 40 * time.Second, 20 * time.Second},
		{"expiresIn equal to margin: clamped to half, not zero/negative", tokenSafetyMargin, tokenSafetyMargin / 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cacheTTL(tt.expiresIn); got != tt.want {
				t.Errorf("cacheTTL(%v) = %v, want %v", tt.expiresIn, got, tt.want)
			}
			if got := cacheTTL(tt.expiresIn); got <= 0 {
				t.Errorf("cacheTTL(%v) = %v, must stay positive so caching isn't defeated", tt.expiresIn, got)
			}
		})
	}
}

func TestTokenManager_AccessTokenB2B_CachesAndRefetchesAfterExpiry(t *testing.T) {
	var reqCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got == "" || got[len(got)-len("/access-token/b2b"):] != "/access-token/b2b" {
			t.Errorf("request path = %q, want suffix /access-token/b2b", got)
		}
		var gotBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if got := gotBody["grantType"]; got != "client_credentials" {
			t.Errorf("request body[grantType] = %v, want client_credentials", got)
		}

		n := reqCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			_, _ = io.WriteString(w, `{"accessToken":"tok1","tokenType":"Bearer","expiresIn":"900"}`)
		} else {
			_, _ = io.WriteString(w, `{"accessToken":"tok2","tokenType":"Bearer","expiresIn":"900"}`)
		}
	}))
	defer server.Close()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	clock := newFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	m := &TokenManager{
		BaseURL:   server.URL,
		ClientKey: "client-key",
		Signer:    key,
		Now:       clock.Now,
	}

	tok1, err := m.AccessTokenB2B(context.Background())
	if err != nil {
		t.Fatalf("first AccessTokenB2B() error = %v", err)
	}
	if tok1 != "tok1" {
		t.Fatalf("first token = %q, want tok1", tok1)
	}
	if got := reqCount.Load(); got != 1 {
		t.Fatalf("request count after first call = %d, want 1", got)
	}

	// Before the computed expiry (900s - 30s safety margin): must not
	// re-fetch.
	clock.Advance(800 * time.Second)
	tok2, err := m.AccessTokenB2B(context.Background())
	if err != nil {
		t.Fatalf("second AccessTokenB2B() error = %v", err)
	}
	if tok2 != "tok1" {
		t.Errorf("second token = %q, want cached tok1", tok2)
	}
	if got := reqCount.Load(); got != 1 {
		t.Fatalf("request count after second call = %d, want 1 (should be cached)", got)
	}

	// Past the computed expiry (800 + 130 = 930s > 900 - 30 = 870s): must
	// re-fetch.
	clock.Advance(130 * time.Second)
	tok3, err := m.AccessTokenB2B(context.Background())
	if err != nil {
		t.Fatalf("third AccessTokenB2B() error = %v", err)
	}
	if tok3 != "tok2" {
		t.Errorf("third token = %q, want refetched tok2", tok3)
	}
	if got := reqCount.Load(); got != 2 {
		t.Fatalf("request count after third call = %d, want 2 (should have refetched)", got)
	}
}

func TestTokenManager_AccessTokenB2B_ConcurrentCallersShareOneFetch(t *testing.T) {
	var reqCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"accessToken":"tok1","tokenType":"Bearer","expiresIn":"900"}`)
	}))
	defer server.Close()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	m := &TokenManager{
		BaseURL:   server.URL,
		ClientKey: "client-key",
		Signer:    key,
	}

	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_, errs[i] = m.AccessTokenB2B(context.Background())
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("caller %d: AccessTokenB2B() error = %v", i, err)
		}
	}
	if got := reqCount.Load(); got != 1 {
		t.Errorf("request count from %d concurrent callers = %d, want 1", n, got)
	}
}

func TestTokenManager_AccessTokenB2B_SignsAsymmetricNeverSymmetric(t *testing.T) {
	var mu sync.Mutex
	var gotTimestamp, gotSignature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotTimestamp = r.Header.Get("X-Timestamp")
		gotSignature = r.Header.Get("X-Signature")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"accessToken":"tok1","tokenType":"Bearer","expiresIn":"900"}`)
	}))
	defer server.Close()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	const clientKey = "client-key"
	m := &TokenManager{
		BaseURL:   server.URL,
		ClientKey: clientKey,
		Signer:    key,
	}

	if _, err := m.AccessTokenB2B(context.Background()); err != nil {
		t.Fatalf("AccessTokenB2B() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotTimestamp == "" || gotSignature == "" {
		t.Fatal("request missing X-Timestamp or X-Signature")
	}

	stringToSign := BuildStringToSignAccessToken(clientKey, gotTimestamp)
	if err := VerifyAsymmetric(key.Public(), stringToSign, gotSignature); err != nil {
		t.Errorf("X-Signature does not verify as SignAsymmetric(%q): %v", stringToSign, err)
	}
}

func TestTokenManager_AccessTokenB2B_ErrorResponseCode(t *testing.T) {
	tests := []struct {
		name         string
		httpStatus   int
		responseCode string
		wantSentinel error
	}{
		{"400 bad request", http.StatusBadRequest, "4007300", ErrBadRequest},
		{"401 unauthorized", http.StatusUnauthorized, "4017301", ErrUnauthorized},
		{"500 internal server error", http.StatusInternalServerError, "5007300", ErrInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.httpStatus)
				_, _ = io.WriteString(w, `{"responseCode":"`+tt.responseCode+`","responseMessage":"failed"}`)
			}))
			defer server.Close()

			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("rsa.GenerateKey: %v", err)
			}
			m := &TokenManager{BaseURL: server.URL, ClientKey: "client-key", Signer: key}

			tok, err := m.AccessTokenB2B(context.Background())
			if err == nil {
				t.Fatal("AccessTokenB2B() error = nil, want non-nil for a non-2xx responseCode")
			}
			if tok != "" {
				t.Errorf("AccessTokenB2B() token = %q, want empty on error", tok)
			}
			if !errors.Is(err, tt.wantSentinel) {
				t.Errorf("AccessTokenB2B() error = %v, want errors.Is match against %v", err, tt.wantSentinel)
			}
		})
	}
}

// TestTokenManager_AccessTokenB2B_NonTwoXXWithEmptyResponseCode covers the
// case a well-formed responseCode is absent but the transport-level status
// still signals failure (e.g. a proxy/WAF error page) — must not be treated
// as success just because responseCode is empty.
func TestTokenManager_AccessTokenB2B_NonTwoXXWithEmptyResponseCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"message":"forbidden by upstream proxy"}`) // no responseCode field at all
	}))
	defer server.Close()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	m := &TokenManager{BaseURL: server.URL, ClientKey: "client-key", Signer: key}

	tok, err := m.AccessTokenB2B(context.Background())
	if err == nil {
		t.Fatal("AccessTokenB2B() error = nil, want non-nil for a 401 status with no responseCode field")
	}
	if tok != "" {
		t.Errorf("AccessTokenB2B() token = %q, want empty on error", tok)
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("AccessTokenB2B() error = %v, want errors.Is match against ErrUnauthorized", err)
	}
}

// TestTokenManager_AccessTokenB2B_WaiterCtxCancelDoesNotBlock proves a
// caller waiting on someone else's in-flight fetch returns as soon as its
// own ctx is cancelled, rather than blocking for the full fetch duration.
func TestTokenManager_AccessTokenB2B_WaiterCtxCancelDoesNotBlock(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // held open until the test explicitly lets it finish
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"accessToken":"tok1","tokenType":"Bearer","expiresIn":"900"}`)
	}))
	defer server.Close()
	defer close(release)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	m := &TokenManager{BaseURL: server.URL, ClientKey: "client-key", Signer: key}

	// Fetcher: starts the in-flight fetch, blocked on the server's <-release.
	go func() { _, _ = m.AccessTokenB2B(context.Background()) }()
	time.Sleep(20 * time.Millisecond) // let the fetcher acquire inFlight and reach the HTTP call

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := m.AccessTokenB2B(ctx)
		done <- err
	}()
	time.Sleep(20 * time.Millisecond) // let the waiter reach the select
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("waiter AccessTokenB2B() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not return promptly after its ctx was cancelled — mutex/fetch is blocking it")
	}
}

// TestTokenManager_AccessTokenB2B_InitiatorCtxCancelDoesNotPoisonWaiters is
// the regression test for the code/security review finding that the shared
// single-flight fetch used to run on the ctx of whichever caller happened to
// start it — so if that caller's own ctx was cancelled, every other waiter
// received that same spurious error instead of the token the server was
// still perfectly willing to provide. The fetch must now run on its own
// detached context, unaffected by any individual caller (initiator or
// waiter) giving up.
func TestTokenManager_AccessTokenB2B_InitiatorCtxCancelDoesNotPoisonWaiters(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"accessToken":"tok1","tokenType":"Bearer","expiresIn":"900"}`)
	}))
	defer server.Close()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	m := &TokenManager{BaseURL: server.URL, ClientKey: "client-key", Signer: key}

	// Initiator: its ctx will be cancelled while the fetch is still running.
	initiatorCtx, cancelInitiator := context.WithCancel(context.Background())
	initiatorDone := make(chan error, 1)
	go func() {
		_, err := m.AccessTokenB2B(initiatorCtx)
		initiatorDone <- err
	}()
	time.Sleep(20 * time.Millisecond) // let the initiator start the fetch

	// Waiter: healthy ctx, should get the real token once the server responds.
	waiterDone := make(chan struct {
		token string
		err   error
	}, 1)
	go func() {
		tok, err := m.AccessTokenB2B(context.Background())
		waiterDone <- struct {
			token string
			err   error
		}{tok, err}
	}()
	time.Sleep(20 * time.Millisecond) // let the waiter join the same fetch

	cancelInitiator()
	select {
	case err := <-initiatorDone:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("initiator AccessTokenB2B() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("initiator did not return promptly after its own ctx was cancelled")
	}

	close(release) // let the server (and thus the shared fetch) finish

	select {
	case res := <-waiterDone:
		if res.err != nil {
			t.Errorf("waiter AccessTokenB2B() error = %v, want nil (must not inherit initiator's cancellation)", res.err)
		}
		if res.token != "tok1" {
			t.Errorf("waiter AccessTokenB2B() token = %q, want %q", res.token, "tok1")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter never received the fetch result")
	}
}

func TestTokenManager_AccessTokenB2B2C_GrantTypes(t *testing.T) {
	tests := []struct {
		name       string
		grantType  GrantType
		code       string
		wantField  string
		wantOther  string // field that must be ABSENT from the request body
		respExtras string
	}{
		{
			name:      "authorization code",
			grantType: GrantTypeAuthorizationCode,
			code:      "the-auth-code",
			wantField: "authCode",
			wantOther: "refreshToken",
		},
		{
			name:      "refresh token",
			grantType: GrantTypeRefreshToken,
			code:      "the-refresh-token",
			wantField: "refreshToken",
			wantOther: "authCode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Path; got == "" || got[len(got)-len("/access-token/b2b2c"):] != "/access-token/b2b2c" {
					t.Errorf("request path = %q, want suffix /access-token/b2b2c", got)
				}
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Errorf("decode request body: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"accessToken":"tok1","tokenType":"Bearer","expiresIn":"1296000","refreshToken":"new-refresh-token"}`)
			}))
			defer server.Close()

			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("rsa.GenerateKey: %v", err)
			}
			m := &TokenManager{BaseURL: server.URL, ClientKey: "client-key", Signer: key}

			tok, err := m.AccessTokenB2B2C(context.Background(), tt.grantType, tt.code)
			if err != nil {
				t.Fatalf("AccessTokenB2B2C() error = %v", err)
			}

			if got, ok := gotBody[tt.wantField]; !ok || got != tt.code {
				t.Errorf("request body[%q] = %v, want %q", tt.wantField, got, tt.code)
			}
			if _, present := gotBody[tt.wantOther]; present {
				t.Errorf("request body unexpectedly contains %q for grant type %q", tt.wantOther, tt.grantType)
			}
			if got := gotBody["grantType"]; got != string(tt.grantType) {
				t.Errorf("request body[grantType] = %v, want %q", got, tt.grantType)
			}

			if tok.AccessToken != "tok1" {
				t.Errorf("AccessToken = %q, want tok1", tok.AccessToken)
			}
			if tok.TokenType != "Bearer" {
				t.Errorf("TokenType = %q, want Bearer", tok.TokenType)
			}
			if tok.ExpiresIn != 1296000*time.Second {
				t.Errorf("ExpiresIn = %v, want %v", tok.ExpiresIn, 1296000*time.Second)
			}
			if tok.RefreshToken != "new-refresh-token" {
				t.Errorf("RefreshToken = %q, want new-refresh-token", tok.RefreshToken)
			}
		})
	}
}

func TestTokenManager_AccessTokenB2B2C_ErrorResponseCode(t *testing.T) {
	tests := []struct {
		name         string
		httpStatus   int
		responseCode string
		wantSentinel error
	}{
		{"400 bad request", http.StatusBadRequest, "4007300", ErrBadRequest},
		{"401 unauthorized", http.StatusUnauthorized, "4017301", ErrUnauthorized},
		{"500 internal server error", http.StatusInternalServerError, "5007300", ErrInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.httpStatus)
				_, _ = io.WriteString(w, `{"responseCode":"`+tt.responseCode+`","responseMessage":"failed"}`)
			}))
			defer server.Close()

			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("rsa.GenerateKey: %v", err)
			}
			m := &TokenManager{BaseURL: server.URL, ClientKey: "client-key", Signer: key}

			tok, err := m.AccessTokenB2B2C(context.Background(), GrantTypeAuthorizationCode, "code")
			if err == nil {
				t.Fatal("AccessTokenB2B2C() error = nil, want non-nil for a non-2xx responseCode")
			}
			if tok != (Token{}) {
				t.Errorf("AccessTokenB2B2C() token = %+v, want zero value on error", tok)
			}
			if !errors.Is(err, tt.wantSentinel) {
				t.Errorf("AccessTokenB2B2C() error = %v, want errors.Is match against %v", err, tt.wantSentinel)
			}
		})
	}
}
