# SNAP Conformance

Specification: SNAP Technical Specification v1.0.2 (September 2024),
published by ASPI (Asosiasi Sistem Pembayaran Indonesia) on the
[ASPI SNAP Developer Site](https://apidevportal.aspi-indonesia.or.id/api-services).

This document states **implementation coverage against SNAP Technical
Specification v1.0.2** — it is a self-reported engineering artifact,
not an official SNAP compliance certification. Nothing here should be
read as "SNAP compliant" or "SNAP certified": those determinations are
ASPI's alone to make. "PASS" below means this project's own test suite
(request/response wire-shape round-trips, HTTP status handling, and
signature round-trip/property tests — see the signing note below)
passes for that category, not that ASPI or Bank Indonesia have
reviewed or endorsed the implementation.

See [SPEC_COVERAGE.md](./SPEC_COVERAGE.md) for the endpoint-by-endpoint
traceability table backing the summary below, and
[SECURITY.md](./SECURITY.md) for the security-specific checks that run
alongside these tests.

## Coverage

| Category          | Service Codes  | Endpoints | Status |
|--------------------|-----------------|-----------|--------|
| Registrasi          | 01–10, 81       | 11        | PASS   |
| Informasi Saldo     | 11              | 1         | PASS   |
| Riwayat Transaksi   | 12–14           | 3         | PASS   |
| Transfer Kredit     | 15–53, 75–78    | 43        | PASS   |
| Transfer Debit      | 54–72, 79–80    | 21        | PASS   |
| Keamanan            | 73–74           | 2         | PASS   |
| Administrasi        | —               | 0         | N/A — no API endpoints published for this category |

79 payment/registration endpoint bindings plus 2 Access Token
(Keamanan) endpoints in the root package, covering every Service Code
currently published across all 7 portal categories.

## Test Coverage

Every endpoint binding has, at minimum:

- A **wire round-trip test**: a worked JSON example (from the portal's
  own documentation, where available) unmarshals into the Go type and
  re-marshals to the same shape.
- A **mandatory-field test**: fields the specification marks Mandatory
  serialize even from a zero-value struct (no `omitempty` dropping
  them); Optional/Conditional fields correctly omit when unset.
- A **transport test suite** (via `httptest`): a successful 2xx
  response parses correctly; a non-2xx `responseCode` is an error; a
  non-2xx HTTP status with a 2xx-shaped body is still an error (HTTP
  status is authoritative, per `CheckResponseStatus`); a 2xx status
  with no `responseCode` in the body is an error.

Signing (`SignSymmetric`, `SignAsymmetric`), token lifecycle
(`TokenManager`), and inbound-request verification (`ServerVerifier`)
in the root package have their own dedicated test suites, run with
`go test -race` to catch concurrency issues in token caching/refresh.
`SignSymmetric`'s test suite includes one known-answer test against an
HMAC-SHA512 value computed independently (via Python's stdlib
hmac/hashlib), not a value published in the SNAP specification itself
— no test in this repo checks output against an ASPI/BI-published
signing example. `SignAsymmetric` (RSA/PKCS#1v1.5) has no known-answer
test at all; it is covered by round-trip and property tests (sign then
verify, determinism, key-strength floor, hostile-input handling), not
validated against an external reference value.

## What this does NOT cover

- **Live-endpoint / sandbox testing.** Tests validate this project's
  own request construction and response parsing against the
  specification's documented shapes and worked examples — they do not
  call ASPI's sandbox or any live PJP endpoint. No live-traffic
  interoperability test exists yet.
- **External validation of the cryptographic signing implementation.**
  As noted above, the HMAC known-answer test is independently computed
  rather than spec-sourced, and RSA signing has no known-answer test
  at all — a consistent bug in the shared string-to-sign construction
  used by both signing and verification would not necessarily be
  caught by these tests, since sign/verify are largely tested against
  each other rather than against an authoritative external value.
- **Every field-level ambiguity in the source specification.** Where
  the portal's own documentation is internally inconsistent (a
  documented HTTP method contradicting a worked example, a field-name
  casing mismatch between a table and its example), this project
  records the ambiguity and states which interpretation it implemented.
  These are flagged, not silently resolved, in each package's own doc
  comments and SPEC_COVERAGE.md.
- **Official ASPI review.** This project has not been submitted to, or
  reviewed by, ASPI or Bank Indonesia.
