package snap

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// GrantType identifies the SNAP OAuth-style grant used for an access-token
// request.
type GrantType string

// Grant types defined by the standard. Values match the wire format exactly
// (note the inconsistent casing is the standard's, not a typo).
const (
	GrantTypeClientCredentials GrantType = "client_credentials"
	GrantTypeAuthorizationCode GrantType = "AUTHORIZATION_CODE"
	GrantTypeRefreshToken      GrantType = "REFRESH_TOKEN"
)

// Token is the result of a B2B2C access-token exchange.
type Token struct {
	AccessToken  string
	TokenType    string
	ExpiresIn    time.Duration
	RefreshToken string // set for B2B2C only
}

// tokenSafetyMargin is subtracted from the server-reported expiresIn so a
// cached B2B token is refetched slightly before it actually expires, rather
// than exactly at the boundary.
const tokenSafetyMargin = 30 * time.Second

// TokenManager fetches and (for B2B) caches SNAP access tokens. Access-token
// requests are always asymmetric-signed, both B2B and B2B2C, regardless of
// the signing mode agreed for transaction requests.
type TokenManager struct {
	BaseURL    string
	ClientKey  string
	Signer     crypto.Signer
	HTTPClient *http.Client
	Profile    Profile
	Now        func() time.Time // optional; nil means time.Now

	mu          sync.Mutex
	cachedToken string
	expiresAt   time.Time
}

func (m *TokenManager) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func (m *TokenManager) profile() Profile {
	if m.Profile != nil {
		return m.Profile
	}
	return DefaultProfile{}
}

func (m *TokenManager) httpClient() *http.Client {
	if m.HTTPClient != nil {
		return m.HTTPClient
	}
	return &http.Client{Timeout: defaultHTTPTimeout}
}

// accessTokenHeaders builds the smaller header set used by access-token
// requests (no X-PARTNER-ID/X-EXTERNAL-ID/CHANNEL-ID), signed via
// SignAsymmetric per the standard, never SignSymmetric.
func (m *TokenManager) accessTokenHeaders(timestamp string) (http.Header, error) {
	stringToSign := BuildStringToSignAccessToken(m.ClientKey, timestamp)
	signature, err := SignAsymmetric(m.Signer, stringToSign)
	if err != nil {
		return nil, fmt.Errorf("snap: token manager: %w", err)
	}
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-TIMESTAMP", timestamp)
	h.Set("X-CLIENT-KEY", m.ClientKey)
	h.Set("X-SIGNATURE", signature)
	return h, nil
}

// accessTokenResponse is the shared decode shape for both B2B and B2B2C
// access-token responses; expiresIn is transmitted as a quoted string
// (the standard's own "900" example).
type accessTokenResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	AccessToken     string `json:"accessToken"`
	TokenType       string `json:"tokenType"`
	ExpiresIn       string `json:"expiresIn"`
	RefreshToken    string `json:"refreshToken"`
}

// envelopeError returns a non-nil error if responseCode is present and
// signals a non-2xx outcome (or is present but malformed) — an error
// response must never be silently ignored. An empty responseCode (the
// normal shape of a successful access-token response) is not an error.
func envelopeError(responseCode string) error {
	if responseCode == "" {
		return nil
	}
	httpStatus, _, _, err := ParseResponseCode(responseCode)
	if err != nil {
		return fmt.Errorf("snap: token manager: response carries malformed response code: %w", err)
	}
	if httpStatus >= 200 && httpStatus < 300 {
		return nil
	}
	return ResponseCodeError(responseCode)
}

// doAccessTokenRequest POSTs body to path and decodes the response.
func (m *TokenManager) doAccessTokenRequest(ctx context.Context, path string, body []byte, timestamp string) (accessTokenResponse, error) {
	headers, err := m.accessTokenHeaders(timestamp)
	if err != nil {
		return accessTokenResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return accessTokenResponse{}, fmt.Errorf("snap: token manager: build request: %w", err)
	}
	for k, v := range headers {
		req.Header[k] = v
	}

	resp, err := m.httpClient().Do(req)
	if err != nil {
		return accessTokenResponse{}, fmt.Errorf("snap: token manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return accessTokenResponse{}, fmt.Errorf("snap: token manager: read response body: %w", err)
	}

	var parsed accessTokenResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return accessTokenResponse{}, fmt.Errorf("snap: token manager: decode response body: %w", err)
	}
	if err := envelopeError(parsed.ResponseCode); err != nil {
		return accessTokenResponse{}, err
	}
	if parsed.AccessToken == "" {
		return accessTokenResponse{}, errors.New("snap: token manager: response has no accessToken")
	}
	return parsed, nil
}

func parseExpiresIn(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, errors.New("snap: token manager: response has no expiresIn")
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("snap: token manager: parse expiresIn: %w", err)
	}
	return time.Duration(seconds) * time.Second, nil
}

// AccessTokenB2B returns a cached client_credentials access token, fetching
// (and caching) a new one if none is cached or the cached one is at or past
// its computed expiry. Guarded by a mutex so concurrent callers share one
// in-flight fetch's result rather than each firing a request.
func (m *TokenManager) AccessTokenB2B(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	if m.cachedToken != "" && now.Before(m.expiresAt) {
		return m.cachedToken, nil
	}

	timestamp := now.Format(m.profile().TimestampLayout())
	body := []byte(`{"grantType":"client_credentials"}`)
	parsed, err := m.doAccessTokenRequest(ctx, m.profile().BuildPath("access-token", "b2b"), body, timestamp)
	if err != nil {
		return "", err
	}
	expiresIn, err := parseExpiresIn(parsed.ExpiresIn)
	if err != nil {
		return "", err
	}

	m.cachedToken = parsed.AccessToken
	m.expiresAt = now.Add(expiresIn - tokenSafetyMargin)
	return m.cachedToken, nil
}

// AccessTokenB2B2C exchanges an authorization code or refresh token for a
// per-end-user Token. Not cached — the caller owns customer-session scoping.
func (m *TokenManager) AccessTokenB2B2C(ctx context.Context, grantType GrantType, code string) (Token, error) {
	var reqBody struct {
		GrantType    GrantType `json:"grantType"`
		AuthCode     string    `json:"authCode,omitempty"`
		RefreshToken string    `json:"refreshToken,omitempty"`
	}
	reqBody.GrantType = grantType
	switch grantType {
	case GrantTypeAuthorizationCode:
		reqBody.AuthCode = code
	case GrantTypeRefreshToken:
		reqBody.RefreshToken = code
	default:
		return Token{}, fmt.Errorf("snap: access token b2b2c: unsupported grant type %q", grantType)
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return Token{}, fmt.Errorf("snap: access token b2b2c: encode request body: %w", err)
	}

	timestamp := m.now().Format(m.profile().TimestampLayout())
	parsed, err := m.doAccessTokenRequest(ctx, m.profile().BuildPath("access-token", "b2b2c"), body, timestamp)
	if err != nil {
		return Token{}, err
	}
	expiresIn, err := parseExpiresIn(parsed.ExpiresIn)
	if err != nil {
		return Token{}, err
	}

	return Token{
		AccessToken:  parsed.AccessToken,
		TokenType:    parsed.TokenType,
		ExpiresIn:    expiresIn,
		RefreshToken: parsed.RefreshToken,
	}, nil
}
