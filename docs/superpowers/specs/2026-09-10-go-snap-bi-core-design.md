# go-snap-bi: Phase 1 — Core Security & Transport Design

Status: approved by user, pending spec self-review
Scope: **Phase 1 only** — shared primitives used by every SNAP endpoint. No
per-service (balance inquiry, transfer, VA, QRIS, direct debit, ...) request/
response bindings. Those are separate follow-up specs (Phase 2+), one per
service group.

## Why this project is split into phases

SNAP (Standar Nasional Open API Pembayaran) covers 60+ distinct endpoints
across ~14 service groups (Registrasi, Informasi Saldo, Riwayat Transaksi,
Transfer Kredit with 10 sub-groups including Virtual Account and QR/MPM,
Transfer Debit with 4 sub-groups including Direct Debit and QR/CPM). Building
all of it as one spec/plan violates the project's own decomposition rule —
the endpoints are mechanical and independent, but the security envelope
underneath every single one of them is common, small, and is where all the
real risk (crypto correctness, timing, signature format) lives.

Phase 1 delivers that shared envelope as a complete, independently useful,
independently testable package. Phase 2+ specs will each add one service
group's typed bindings on top of it — additive, non-breaking, parallelizable.

## Source of truth and versioning

Three separate version axes exist in SNAP. Conflating them is the most likely
way this package silently breaks on a future BI/ASPI revision, so they are
tracked separately everywhere in code and docs:

| Axis | Current value | Where it comes from |
|---|---|---|
| Standard document version | **v1.0.2** (September 2024) | "Pedoman Standar Teknis dan Keamanan SNAP.pdf" — the security/technical envelope document this design implements |
| Per-service API version | **1.0** | The `Version` field in each service's "Informasi Umum" table on the ASPI SNAP Developer Site (apidevportal.aspi-indonesia.or.id) — e.g. Balance Inquiry, Transaction Status Inquiry |
| URI path version segment | **v1.0** | §2.1.11 of the standard: path format `/{domain}/{version}/{service-group}/{product-type}`, version expressed as `/v{major}.{minor}/` |

No machine-readable spec (OpenAPI/Swagger/Postman) is published. The
"Aplikasi Pengujian" sandbox on the dev portal requires signup and was not
accessible during research. Every endpoint's field-level schema must be
hand-transcribed from the portal's "Guides" tab (confirmed format: Parameter /
Data Type / Mandatory / Length / Description tables, consistent with the PDF's
style) during that endpoint's own Phase 2+ spec — not pre-fetched now.

Known v1.0.2 changelog deltas worth encoding as compatibility notes:
- B2B access-token request body field renamed `granttype` → `grantType`
  (camelCase) as of v1.0.2. Only implement the current name.
- `X-EXTERNAL-ID` description changed "Numeric String" → "Alphanumeric" for
  B2B in v1.0.2, but the B2B2C transaction header table (page 19 of the PDF)
  still says "Numeric String". This is treated as a spec inconsistency, not
  silently resolved: the field type in code is a plain `string` with no
  numeric-only validation, for both B2B and B2B2C.
- The word "Private Key" was removed from §1.7.1/§1.7.2 (API Public Key
  receipt/delivery method sections) in v1.0.2. Cosmetic wording change only,
  does not affect the asymmetric signing implementation (see below).

## Scope decisions (from brainstorming)

- **Client AND server.** The package provides both: primitives to call out to
  a Penyedia Layanan (bank/PJP) as a client, and primitives to verify
  incoming requests as a Penyedia Layanan. These are inverse operations
  sharing the same crypto core.
- **Standard + per-PJP quirks layer.** Real PJPs deviate from the written
  standard in small ways (timestamp millisecond handling, path prefixes,
  extra required fields). The package implements the standard exactly by
  default, with named override points for known deviations — not a strict
  standard-only implementation.
- **Single flat package** (`package snap`, module `github.com/koriebruh/go-snap-bi`).
  No `client`/`server` subpackage split — deferred until real growth forces
  it (YAGNI; the whole package is small in Phase 1).
- **Quirks expressed as interface hooks**, not a declarative config struct.
  A small `Profile` interface with sane standard-compliant defaults; a
  PJP-specific profile embeds the default and overrides only the method(s)
  that differ.

## Components

### 1. Signing primitives

Two algorithms, kept as **separate functions** — the `stringToSign` formulas
genuinely differ (symmetric includes `AccessToken`, asymmetric does not), so
a single function with a bool flag would hide that difference and invite
mistakes.

```go
func SignSymmetric(clientSecret, stringToSign string) string
func SignAsymmetric(signer crypto.Signer, stringToSign string) (string, error)

func VerifySymmetric(clientSecret, stringToSign, signature string) bool
func VerifyAsymmetric(pub crypto.PublicKey, stringToSign, signature string) error

func BuildStringToSignAccessToken(clientID, timestamp string) string
func BuildStringToSignTransaction(method, endpointURL, accessToken string, body []byte, timestamp string, symmetric bool) string

func ParseRSAPrivateKeyPEM(pem []byte) (crypto.Signer, error) // convenience only
```

- `SignAsymmetric` takes `crypto.Signer`, not a raw `*rsa.PrivateKey` or PEM
  bytes. This lets a caller plug in a PEM-loaded key today and an HSM/KMS-
  backed signer later without an API change. `ParseRSAPrivateKeyPEM` is
  shipped as a convenience helper, not a requirement.
- Algorithms per spec: `HMAC_SHA512` (symmetric, 512-bit) and `SHA256withRSA`
  (asymmetric, RSA key ≥256 bits). Both formulas hash-encode the request body
  as `Lowercase(HexEncode(SHA256(minify(RequestBody))))`.
- **`minify(RequestBody)` correctness constraint**: the exact bytes hashed
  must be the exact bytes sent on the wire. The implementation marshals the
  request body to JSON exactly once and reuses those bytes both for hashing
  and for the HTTP request body — never re-marshals for hashing separately,
  since Go map key ordering is non-deterministic across separate
  `json.Marshal` calls and would desync signature from payload.
- Timestamp layout (`yyyy-MM-ddTHH:mm:ss.SSSTZD`) is a named constant
  (`DefaultTimestampLayout`), not inlined — the spec's own worked examples
  omit milliseconds inconsistently, and real PJPs vary here. Overridable via
  `Profile.TimestampLayout()`.
- The doc renders the asymmetric formula as `SHA256withRSA(clientSecret,
  stringToSign)` — verified as a documentation artifact (the same wording
  the v1.0.2 changelog touched), not a real requirement to use clientSecret
  for RSA signing. Implemented with the RSA private key, as RSA signing
  requires.

### 2. Token lifecycle (client side)

```go
type TokenManager struct { /* client key, signer, http client, base URL, cached token, mutex */ }

func (m *TokenManager) AccessTokenB2B(ctx context.Context) (string, error)
func (m *TokenManager) AccessTokenB2B2C(ctx context.Context, grantType GrantType, code string) (Token, error)
```

- `AccessTokenB2B`: `client_credentials` grant, always asymmetric-signed per
  spec regardless of the partner's agreed transaction-signing mode. Caches
  the token and auto-refetches based on the server-returned `expiresIn`
  (spec value: 900s / 15 minutes) minus a safety margin — never a hardcoded
  TTL. Mutex-guarded for concurrent use.
- `AccessTokenB2B2C`: `authorization_code` or `refresh_token` grant (spec
  TTL: 15 days). Per-end-user; the caller supplies the auth code from their
  own consent flow (out of scope) and owns customer-session scoping — this
  method only executes the token exchange.
- **Transaction-level signing mode** (symmetric-with-token vs
  asymmetric-without-token) is a **per-partner agreement fixed at
  registration time**, per the standard's own wording — not something
  inferred per-request. It is a config field on `TokenManager` /
  `ServerVerifier`, set once per integration.

### 3. Server-side verification

```go
type KeyStore interface {
    PublicKey(clientKey string) (crypto.PublicKey, error)
    ClientSecret(clientKey string) (string, error)
}

type ServerVerifier struct { /* KeyStore, SignatureMode, timestamp tolerance */ }

func (v *ServerVerifier) VerifyAccessTokenRequest(req IncomingRequest) error
func (v *ServerVerifier) VerifyTransactionRequest(req IncomingRequest) error
```

- `KeyStore` is caller-implemented — key/secret storage is an application
  concern (DB, vault, etc.), not something this package should own.
- `VerifyAccessTokenRequest` validates the asymmetric signature on the token
  endpoint (always asymmetric per spec).
- `VerifyTransactionRequest` validates per the pre-agreed
  `SignatureMode` (symmetric or asymmetric) and checks `X-TIMESTAMP` falls
  within a **configurable** freshness window — the spec does not pin a
  tolerance value, so this must not be hardcoded.

### 4. Header assembly and quirk hooks

```go
type Profile interface {
    TimestampLayout() string
    BuildPath(serviceGroup, productType string) string
}

var DefaultProfile Profile // implements standard SNAP behavior exactly
```

- `HeaderBuilder` assembles the mandatory header set per request type:
  `Content-Type`, `X-TIMESTAMP`, `X-CLIENT-KEY`, `X-SIGNATURE`,
  `X-PARTNER-ID`, `X-EXTERNAL-ID`, `CHANNEL-ID`, `ORIGIN`, plus
  B2B2C-only fields (`Authorization-Customer`, `X-IP-ADDRESS`,
  `X-DEVICE-ID`, `X-LATITUDE`, `X-LONGITUDE`) when applicable.
- Timestamp formatting and URI path construction (§2.1.11:
  `/{domain}/{version}/{service-group}/{product-type}`) are delegated to
  `Profile`, so a bank-specific deviation is expressed as a small struct
  embedding `DefaultProfile` and overriding one method — not a fork of the
  header builder.

### 5. Transport

```go
type Transport struct { /* *http.Client, Profile, signer/verifier config */ }

type Envelope struct {
    ResponseCode    string
    ResponseMessage string
    Raw             json.RawMessage
}

func (t *Transport) Do(ctx context.Context, req OutgoingRequest) (Envelope, error)
```

- Wraps an injectable `*http.Client` (default: sane timeout). No retry/
  circuit-breaker logic in Phase 1 — YAGNI until a concrete need appears.
- Builds headers, signs, sends, and parses the response into a generic
  `Envelope`. Phase 1 has no per-service typed response structs yet (that's
  Phase 2 work per service group) — callers unmarshal `Envelope.Raw`
  themselves for now.

### 6. Response codes

```go
func ParseResponseCode(code string) (httpStatus int, serviceCode, caseCode string, err error)

var (
    ErrBadRequest   = errors.New("snap: bad request")
    ErrUnauthorized = errors.New("snap: unauthorized")
    ErrForbidden    = errors.New("snap: forbidden")
    // ... one sentinel per HTTP status class actually used by SNAP
)
```

- `responseCode` is confirmed structured as `HTTPStatus(3) + ServiceCode(2)
  + CaseCode(2)` = 7 characters, generic across all services (verified on
  the ASPI dev portal's "Response Code" tab, consistent with observed
  service codes 11/36/53/73/74 in the technical standard).
- Sentinel errors are matched by HTTP-status class via `errors.Is`, since
  case codes are numerous per class but the HTTP class is stable and is
  what most callers actually branch on.

## Testing strategy

Standard `go test`, table-driven, stdlib only — no mocking libraries or test
frameworks.

- **Signing**: round-trip sign→verify for both symmetric and asymmetric.
  `stringToSign` construction checked against the standard's worked
  examples. Correctness relies on stdlib `crypto/hmac` and `crypto/rsa`
  being used correctly, not on reimplementing cryptographic primitives.
- **`ParseResponseCode`**: table-driven across HTTP-class boundaries
  (2xx/4xx/5xx) and malformed-code inputs.
- **`HeaderBuilder`**: table-driven, default `Profile` vs. a stub override,
  confirming a quirk hook actually changes the assembled headers.
- **`TokenManager`**: `httptest.Server` mocking the access-token endpoint;
  an injectable `func() time.Time` clock field (not a full clock interface —
  YAGNI) to test cache-expiry/refetch deterministically without sleeping.
- **`ServerVerifier`**: round-trip test — a client-side signer signs a
  request, the server-side verifier (with a fake `KeyStore`) validates it —
  proving the two halves are genuinely inverse operations.

## Out of scope for Phase 1 (explicitly deferred)

- Any per-service request/response types (balance inquiry, transfer,
  virtual account, QRIS/MPM/CPM, direct debit, registration, transaction
  history, etc.) — each becomes its own Phase 2+ spec.
- A registry of known PJP `Profile` implementations (e.g. built-in BCA/
  Mandiri/BNI/BRI profiles) — Phase 1 ships the hook mechanism only; actual
  per-bank profiles are added as they're needed and verified against a real
  sandbox.
- Retry, circuit-breaking, or rate-limiting in `Transport`.
- Fraud Detection System (FDS) integration — the standard describes FDS as
  the Penyedia Layanan's own system requirement, not part of the Open API
  surface this package wraps.
