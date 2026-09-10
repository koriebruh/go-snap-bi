# GAN Harness Spec — go-snap-bi Phase 1

Source of truth: `docs/superpowers/specs/2026-09-10-go-snap-bi-core-design.md`.
This file scopes each generator/evaluator build loop to one slice at a time.
Do not implement slices beyond the one named as CURRENT.

## Global constraints (apply to every slice)

- Module `github.com/koriebruh/go-snap-bi`, single flat `package snap` at
  repo root. No subpackages.
- Go stdlib only. No third-party dependencies for crypto, HTTP, or testing.
- Table-driven `go test`, stdlib `testing` only — no mocking libraries or
  test frameworks.
- No retry/circuit-breaker/rate-limit logic anywhere (explicitly out of
  scope in the design doc).
- No per-service (balance inquiry, transfer, VA, QRIS, direct debit, ...)
  types or endpoints — Phase 1 is the security/transport envelope only.

## DONE: Slice 1 — Signing core

Implemented and passed GAN eval (9.33/10) + code/security review on branch
`main` (commits `ca2026b`, `bacc861`). Kept here for reference only — do not
re-implement.

Implement, in package `snap`:

- `SignSymmetric(clientSecret, stringToSign string) string` — HMAC-SHA512,
  lowercase hex-encoded output.
- `VerifySymmetric(clientSecret, stringToSign, signature string) bool` —
  constant-time comparison.
- `SignAsymmetric(signer crypto.Signer, stringToSign string) (string, error)`
  — SHA256withRSA (`crypto.SHA256`, PKCS#1 v1.5), lowercase hex-encoded
  output. Must accept any `crypto.Signer`, not a concrete `*rsa.PrivateKey`.
- `VerifyAsymmetric(pub crypto.PublicKey, stringToSign, signature string) error`
  — inverse of the above using `rsa.VerifyPKCS1v15`.
- `BuildStringToSignAccessToken(clientID, timestamp string) string` —
  formula: `clientID + "|" + timestamp`.
- `BuildStringToSignTransaction(method, endpointURL, accessToken string, body []byte, timestamp string, symmetric bool) string`
  — symmetric formula: `HTTPMethod + ":" + EndpointUrl + ":" + AccessToken + ":" + Lowercase(HexEncode(SHA256(body))) + ":" + TimeStamp`.
  Asymmetric formula: same but **without** the `AccessToken` segment:
  `HTTPMethod + ":" + EndpointUrl + ":" + Lowercase(HexEncode(SHA256(body))) + ":" + TimeStamp`.
  `body` is the already-minified (single-marshaled) request body bytes — this
  function does not marshal JSON itself, it only hashes the bytes it is
  given. Empty request body → hash of an empty byte slice, per spec note
  ("dalam hal tidak terdapat Request Body maka digunakan string kosong").
- `ParseRSAPrivateKeyPEM(pemBytes []byte) (crypto.Signer, error)` — parses a
  PKCS#1 or PKCS#8 PEM-encoded RSA private key, returns it as `crypto.Signer`.
  Convenience helper; return a clear error for non-RSA keys or malformed PEM.
- `DefaultTimestampLayout` — exported `const string` for the Go time layout
  equivalent to `yyyy-MM-ddTHH:mm:ss.SSSTZD` (e.g.
  `"2006-01-02T15:04:05.000-07:00"`).

### Required tests (table-driven, stdlib `testing`)

- Round-trip: sign with `SignSymmetric`, verify with `VerifySymmetric` — true
  for correct signature, false for tampered signature or tampered
  `stringToSign`.
- Round-trip: sign with `SignAsymmetric` using a generated `*rsa.PrivateKey`
  (`rsa.GenerateKey` in test setup, >=2048 bits for test speed), verify with
  `VerifyAsymmetric` using the corresponding public key — success for
  correct signature, error for tampered signature or wrong key.
- `BuildStringToSignTransaction`: table test proving the symmetric formula
  includes the access token and the asymmetric formula does not, given
  identical other inputs — assert the two outputs differ only by the
  `AccessToken` segment.
- `BuildStringToSignTransaction` with empty `body`: assert it uses the SHA256
  hash of an empty byte slice, not an error or panic.
- `ParseRSAPrivateKeyPEM`: valid PKCS#1 PEM, valid PKCS#8 PEM, malformed PEM
  (error expected), non-RSA key type (error expected).

## CURRENT SLICE: Slice 2 — Header assembly + response codes

Implement, in package `snap` (builds on Slice 1's `DefaultTimestampLayout`,
`SignSymmetric`/`SignAsymmetric`, `BuildStringToSignTransaction`):

- `Profile` interface:
  ```go
  type Profile interface {
      TimestampLayout() string
      BuildPath(serviceGroup, productType string) string
  }
  ```
  `TimestampLayout()` default returns `DefaultTimestampLayout` (from Slice 1).
  `BuildPath(serviceGroup, productType string) string` default implements
  the standard's URI path format (design doc §"Header assembly and quirk
  hooks", spec §2.1.11): `/{domain}/{version}/{service-group}/{product-type}`.
  Since domain and version are integration-specific (not global constants),
  `DefaultProfile` should be a struct with `Domain` and `Version` fields
  (e.g. `Version` defaults to `"v1.0"` per the URI version segment format
  `/v{major}.{minor}/`), not a zero-field singleton — a caller must be able
  to construct `DefaultProfile{Domain: "openapi.example.com", Version: "v1.0"}`.
  A PJP-specific profile embeds `DefaultProfile` and overrides one method.

- `HeaderBuilder` — a struct or set of functions that assemble the mandatory
  header set for a request, given: HTTP method, full endpoint URL, request
  body bytes, whether it's a B2B or B2B2C request, an `Authorization`
  bearer token (and `Authorization-Customer` for B2B2C), `clientKey`,
  `partnerID`, `externalID`, `channelID`, and a `Profile`. Produces a
  `http.Header` (or equivalent map) containing:
  - Always: `Content-Type: application/json`, `X-TIMESTAMP` (formatted via
    `Profile.TimestampLayout()`), `X-CLIENT-KEY`, `X-SIGNATURE` (computed via
    `BuildStringToSignTransaction` + `SignSymmetric`/`SignAsymmetric` from
    Slice 1), `X-PARTNER-ID`, `X-EXTERNAL-ID`, `CHANNEL-ID`.
  - `Authorization: Bearer <token>` for transaction requests (both B2B and
    B2B2C).
  - `ORIGIN` — optional, include only if a non-empty origin is supplied.
  - B2B2C only, **mandatory**: `Authorization-Customer`, `X-DEVICE-ID` —
    return an error if either is empty on a B2B2C request, per the
    standard (§2.1.6.b), rather than omitting them.
  - B2B2C only, genuinely optional (omitted when empty): `X-IP-ADDRESS`,
    `X-LATITUDE`, `X-LONGITUDE`.
  Do not hardcode which signing function to call — accept a `symmetric bool`
  parameter (or equivalent) matching `BuildStringToSignTransaction`'s own
  parameter, and the corresponding secret/signer.

- `ParseResponseCode(code string) (httpStatus int, serviceCode, caseCode string, err error)`
  — splits a 7-character `responseCode` into `HTTPStatus(3) + ServiceCode(2)
  + CaseCode(2)`. Returns a clear error for any code not exactly 7 digits or
  containing non-digit characters.

- Sentinel errors for the response code's HTTP-status class, one per class
  actually documented in the standard's response-code table: `ErrBadRequest`
  (400), `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrNotFound` (404),
  `ErrInternalServerError` (500), `ErrServiceUnavailable` (503), `ErrTimeout`
  (504). Plus a function `ResponseCodeError(code string) error` that parses
  the code and returns the matching sentinel wrapped with the raw code via
  `%w`/`fmt.Errorf`, so callers can `errors.Is(err, snap.ErrUnauthorized)`
  regardless of the specific case code. For an HTTP class with no matching
  sentinel, return a generic wrapped error, not a panic or nil.

### Required tests (table-driven, stdlib `testing`)

- `HeaderBuilder`: table test for a B2B request (asserts no B2B2C-only
  headers present), a B2B2C request (asserts B2B2C-only headers present when
  supplied, absent when not), and one test proving a custom `Profile`
  (stub embedding `DefaultProfile`, overriding `TimestampLayout` to a fixed
  string) changes the resulting `X-TIMESTAMP` header — proves the hook
  actually takes effect.
- `DefaultProfile.BuildPath`: table test covering the standard path shape
  with a couple of different service-group/product-type combinations.
- `ParseResponseCode`: table test across at least one code per HTTP class
  documented in the standard (200, 400, 401, 403, 404, 500, 503, 504), plus
  malformed inputs (wrong length, non-digit characters, empty string) —
  all must return a non-nil `err`, not panic.
- `ResponseCodeError`: table test asserting `errors.Is` matches the correct
  sentinel for each HTTP class, and that the raw code string is still
  recoverable from the error message (via `%w` wrapping, not swallowed).

## DONE: Slice 2 — Header assembly + response codes

Implemented, GAN-evaluated (9.00/10), code/security reviewed and fixed on
branch `feat/phase1-core` (commits `13e15ea`, `ad0036d`, `1b7f9ab`,
`6af2f4a`). Kept above for reference only — do not re-implement.

## CURRENT SLICE: Slice 3 — Transport, Envelope, TokenManager

Implement, in package `snap` (builds on Slice 1's signing functions and
Slice 2's `HeaderBuilder`/`Profile`):

- `Envelope` struct:
  ```go
  type Envelope struct {
      ResponseCode    string
      ResponseMessage string
      Raw             json.RawMessage
  }
  ```
  `Raw` holds the full decoded response body so a caller can unmarshal it
  into a service-specific type in a later phase; Phase 1 has no
  per-service types yet.

- `Transport` struct + `Do` method:
  ```go
  type Transport struct {
      HTTPClient *http.Client // optional; nil means a client with a sane default timeout (e.g. 30s)
  }

  func (t *Transport) Do(ctx context.Context, hb HeaderBuilder) (Envelope, error)
  ```
  Reuses Slice 2's `HeaderBuilder` directly instead of inventing a parallel
  request type — `hb.EndpointURL` is already the full URL, `hb.Method` the
  HTTP verb, `hb.Body` the exact wire bytes. `Do` calls `hb.Build()` to get
  the signed headers (propagating its error unchanged), builds an
  `*http.Request` via `http.NewRequestWithContext(ctx, hb.Method,
  hb.EndpointURL, bytes.NewReader(hb.Body))`, copies the built headers onto
  it, executes it via `HTTPClient` (or the default), reads and closes the
  response body, and unmarshals it into `Envelope` — `responseCode` and
  `responseMessage` extracted into their named fields, the whole decoded
  body also kept as `Raw`. A non-2xx HTTP status is not itself a Go `error`
  from `Do` — the caller inspects `Envelope.ResponseCode` via
  `ResponseCodeError` (Slice 2) to decide; only transport-level failures
  (network error, non-JSON body, context cancellation) are returned as
  `error`.

- `GrantType` + `Token`:
  ```go
  type GrantType string

  const (
      GrantTypeClientCredentials GrantType = "client_credentials"
      GrantTypeAuthorizationCode GrantType = "AUTHORIZATION_CODE"
      GrantTypeRefreshToken      GrantType = "REFRESH_TOKEN"
  )

  type Token struct {
      AccessToken  string
      TokenType    string
      ExpiresIn    time.Duration
      RefreshToken string // set for B2B2C only
  }
  ```

- `TokenManager` struct + methods:
  ```go
  type TokenManager struct {
      BaseURL    string        // e.g. "https://openapi.example.com"
      ClientKey  string
      Signer     crypto.Signer // access-token requests are always asymmetric-signed, both B2B and B2B2C
      HTTPClient *http.Client  // optional; nil means a sane default
      Profile    Profile       // optional; nil means DefaultProfile{}
      Now        func() time.Time // optional; nil means time.Now, injectable for deterministic tests

      // unexported: cached B2B token + expiry + sync.Mutex
  }

  func (m *TokenManager) AccessTokenB2B(ctx context.Context) (string, error)
  func (m *TokenManager) AccessTokenB2B2C(ctx context.Context, grantType GrantType, code string) (Token, error)
  ```
  - Access-token requests use a different, simpler header set than
    transaction requests (no `X-PARTNER-ID`/`X-EXTERNAL-ID`/`CHANNEL-ID`):
    `Content-Type`, `X-TIMESTAMP`, `X-CLIENT-KEY`, `X-SIGNATURE`. Do not
    route this through `HeaderBuilder` (which is shaped for transaction
    requests) — build this smaller header set directly.
  - `stringToSign` for the access-token request is
    `BuildStringToSignAccessToken(clientKey, timestamp)` (Slice 1), always
    signed via `SignAsymmetric` (Slice 1) — never `SignSymmetric`, per the
    standard, regardless of what signing mode is used for transaction
    requests.
  - `AccessTokenB2B`: POSTs `{"grantType":"client_credentials"}` to
    `BaseURL + Profile.BuildPath("access-token", "b2b")`. On success, parses
    `accessToken`, `tokenType`, `expiresIn` (seconds, per the standard's
    `"900"` example) from the response body, caches the token keyed by
    nothing else (one `TokenManager` = one client credential = one cached
    token), and computes an internal expiry as `m.now() + expiresIn -
    safetyMargin` where `safetyMargin` is a small fixed constant (e.g. 30s)
    — not the raw `expiresIn` treated as exact. A call before that computed
    expiry returns the cached token with zero HTTP calls; a call at or past
    it re-fetches. Guarded by a mutex for concurrent callers.
  - `AccessTokenB2B2C`: POSTs `{"grantType":<grantType>,"authCode":<code>}`
    (when `grantType == GrantTypeAuthorizationCode`) or
    `{"grantType":<grantType>,"refreshToken":<code>}` (when
    `grantType == GrantTypeRefreshToken`) to `BaseURL +
    Profile.BuildPath("access-token", "b2b2c")`. Parses `accessToken`,
    `tokenType`, `expiresIn`, `refreshToken` from the response and returns
    a `Token` — no caching (per-end-user; the caller owns session scoping,
    per the design doc).
  - A non-nil `responseCode`/`responseMessage` error response (per Slice
    2's `ParseResponseCode`/`ResponseCodeError`) is surfaced as the
    returned `error`, not silently ignored.

### Required tests (table-driven, stdlib `testing`)

- `Transport.Do`: using `httptest.Server`, a test that returns a JSON body
  with `responseCode`/`responseMessage`/extra fields, asserting `Envelope`
  correctly extracts the named fields and preserves the rest in `Raw`
  (unmarshal `Raw` into a small anonymous struct in the test to confirm).
  Also a test asserting the request the server actually received carries
  the signed headers from `HeaderBuilder` (e.g. check `X-Signature` is
  non-empty and `X-Timestamp` is present in what the test server recorded).
- `TokenManager.AccessTokenB2B`: using `httptest.Server` with an injectable
  `Now`, a test proving (a) the first call hits the server and caches the
  result, (b) a second call before the computed expiry does NOT hit the
  server again (assert on a request counter), (c) advancing the injected
  clock past the computed expiry causes a third call to hit the server
  again. Also a test asserting the request signature was computed via
  `SignAsymmetric`/`BuildStringToSignAccessToken`, not the symmetric path.
- `TokenManager.AccessTokenB2B2C`: using `httptest.Server`, one test per
  grant type (`GrantTypeAuthorizationCode`, `GrantTypeRefreshToken`)
  asserting the correct body field (`authCode` vs `refreshToken`) is sent,
  and the returned `Token` (including `RefreshToken`) is parsed correctly.
- Both `TokenManager` methods: a test where the mock server returns a
  non-2xx `responseCode` (e.g. `"401xxxx"`-shaped per Slice 2), asserting
  a non-nil `error` is returned rather than a "successful" empty token.

## DONE: Slice 3 — Transport, Envelope, TokenManager

Implemented, GAN-evaluated (8.5/10), code/security reviewed across two
rounds and fixed on branch `feat/phase1-core` (commits `8748748`, `d92d734`,
`76e69d3`). Kept above for reference only — do not re-implement.

## CURRENT SLICE: Slice 4 — KeyStore, ServerVerifier (final Phase 1 slice)

Implement, in package `snap` (builds on Slice 1's verify functions and
Slice 2's response-code helpers; this is the "Penyedia Layanan verifies an
incoming request" side, the inverse of Slice 3's client-side signing):

- `KeyStore` interface — caller-implemented, since key/secret storage is an
  application concern (DB, vault, etc.), not something this package owns:
  ```go
  type KeyStore interface {
      PublicKey(clientKey string) (crypto.PublicKey, error)
      ClientSecret(clientKey string) (string, error)
  }
  ```

- `SignatureMode` — the transaction-level signing mode agreed with a
  partner at registration time (per the standard's own wording; this is
  config, not a per-request runtime choice):
  ```go
  type SignatureMode int

  const (
      SignatureModeSymmetric SignatureMode = iota // default zero value
      SignatureModeAsymmetric
  )
  ```

- `IncomingRequest` — the fields a `ServerVerifier` needs, extracted by the
  caller from whatever HTTP framework they use (this package doesn't parse
  `*http.Request` itself, to stay framework-agnostic):
  ```go
  type IncomingRequest struct {
      Method      string // HTTP method, e.g. "POST"
      EndpointURL string // full endpoint URL exactly as used in the signing formula
      Body        []byte // exact request body bytes as received
      Timestamp   string // X-TIMESTAMP header value
      ClientKey   string // X-CLIENT-KEY header value
      Signature   string // X-SIGNATURE header value
      AccessToken string // Authorization header's token, without the "Bearer " prefix; required for symmetric transaction verification, ignored otherwise
  }
  ```

- `ServerVerifier`:
  ```go
  type ServerVerifier struct {
      KeyStore        KeyStore
      Mode            SignatureMode    // transaction-level mode; access-token requests are always asymmetric regardless of this
      TimestampWindow time.Duration    // freshness tolerance; zero means no freshness check (explicit opt-in, not hardcoded)
      Profile         Profile          // optional; nil means DefaultProfile{}, used only for TimestampLayout when parsing Timestamp
      Now             func() time.Time // optional; nil means time.Now, injectable for tests
  }

  func (v *ServerVerifier) VerifyAccessTokenRequest(req IncomingRequest) error
  func (v *ServerVerifier) VerifyTransactionRequest(req IncomingRequest) error
  ```
  - Both methods first check timestamp freshness when `TimestampWindow > 0`:
    parse `req.Timestamp` using `v.profile().TimestampLayout()` (falling
    back to `DefaultTimestampLayout` via `DefaultProfile{}`), return an
    error if it fails to parse or if `|now - parsedTime| > TimestampWindow`.
  - `VerifyAccessTokenRequest`: always asymmetric, regardless of `Mode`
    (matches Slice 3's `TokenManager`, which always signs access-token
    requests asymmetrically). Looks up the public key via
    `KeyStore.PublicKey(req.ClientKey)`, builds
    `BuildStringToSignAccessToken(req.ClientKey, req.Timestamp)` (Slice 1),
    verifies via `VerifyAsymmetric` (Slice 1). A `KeyStore` lookup failure
    is returned as-is (wrapped with context), not swallowed.
  - `VerifyTransactionRequest`: builds
    `BuildStringToSignTransaction(req.Method, req.EndpointURL,
    req.AccessToken, req.Body, req.Timestamp, v.Mode ==
    SignatureModeSymmetric)` (Slice 1). If `v.Mode == SignatureModeSymmetric`,
    looks up the secret via `KeyStore.ClientSecret(req.ClientKey)` and
    verifies via `VerifySymmetric` (Slice 1), converting its `bool` return
    into an error (e.g. a new `ErrSignatureMismatch` sentinel) rather than
    silently returning `false`. If `v.Mode == SignatureModeAsymmetric`,
    looks up the public key via `KeyStore.PublicKey` and verifies via
    `VerifyAsymmetric`.
  - Neither method interprets a request as valid just because a `KeyStore`
    lookup succeeded — the signature check itself is always performed and
    is what determines pass/fail.

### Required tests (table-driven, stdlib `testing`)

- Round-trip: use Slice 1's `SignSymmetric`/`SignAsymmetric` to construct a
  correctly-signed `IncomingRequest` (both an access-token-shaped one and a
  transaction-shaped one, in both `SignatureMode`s), verify it succeeds via
  `ServerVerifier` with a fake in-memory `KeyStore`. Then tamper one field
  at a time (signature, body, timestamp, client key pointing at a
  different key/secret) and assert verification fails for each.
- `KeyStore` lookup failure (unknown `clientKey`): assert the error is
  returned, not swallowed into a generic "verification failed", and that no
  panic occurs.
- Timestamp freshness: a request signed with a timestamp far outside
  `TimestampWindow` fails even with an otherwise-correct signature; a
  request within the window succeeds; `TimestampWindow == 0` accepts a
  wildly-old timestamp (documents the explicit opt-out).
- `VerifyAccessTokenRequest` ignores `Mode` entirely — a table test with
  both `SignatureModeSymmetric` and `SignatureModeAsymmetric` set on the
  same `ServerVerifier`, both correctly verifying an asymmetrically-signed
  access-token request, proves `Mode` has no effect on this method.

## DONE: Slice 4 — KeyStore, ServerVerifier (final Phase 1 slice)

Implemented, GAN-evaluated (9.17/10), code/security reviewed and fixed on
branch `feat/phase1-core` (commits `7773403`, `7dd0f43`, `808d91a`) —
including closing a CRITICAL empty-secret HMAC auth-bypass caught in
security review, plus aligning `SignatureMode`'s zero value with
`HeaderBuilder.Symmetric`'s and making `TimestampWindow` secure-by-default.

## Phase 1 status: COMPLETE

All four slices (signing core, header assembly + response codes, transport
+ token lifecycle, server-side verification) are implemented, evaluated, and
reviewed on branch `feat/phase1-core`. Next step is opening a PR into `main`
and merging, per the design doc's scope for this package's first shippable
release.
