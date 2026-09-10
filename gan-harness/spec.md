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

## Later slices (not in scope for this GAN loop — do not implement now)

- Slice 3: `Transport`, `Envelope`, `TokenManager` (B2B + B2B2C).
- Slice 4: `KeyStore`, `ServerVerifier`.
