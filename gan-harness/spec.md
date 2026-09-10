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

## CURRENT SLICE: Slice 1 — Signing core

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

## Later slices (not in scope for this GAN loop — do not implement now)

- Slice 2: `Profile` interface + `DefaultProfile`, `HeaderBuilder`,
  `ParseResponseCode` + sentinel errors.
- Slice 3: `Transport`, `Envelope`, `TokenManager` (B2B + B2B2C).
- Slice 4: `KeyStore`, `ServerVerifier`.
