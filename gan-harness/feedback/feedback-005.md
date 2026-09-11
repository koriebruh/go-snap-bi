# Evaluation — Slice 3 (Transport, Envelope, TokenManager) — commit 8748748

## Verification performed

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `gofmt -l .` — no output (all files formatted).
- `go test ./... -race -v -count=1` — all tests pass, including every
  required table case from `spec.md`'s Slice 3 "Required tests" section.
- `go test ./... -race -run 'TokenManager|TransportDo' -count=10` — pass,
  no flakiness.
- `go test ./... -race -run 'TokenManager|TransportDo' -count=20 -shuffle=on`
  — pass, no flakiness, no ordering sensitivity.
- `go.mod` has no `require` block — stdlib-only confirmed. Test files import
  only `context`, `crypto/rand`, `crypto/rsa`, `encoding/json`, `errors`,
  `io`, `net/http`, `net/http/httptest`, `sync`, `sync/atomic`, `testing`,
  `time` — no testify/gomock/third-party test helpers.
- `grep`'d for `KeyStore`/`ServerVerifier` and per-service identifiers
  (balance, transfer, VA, QRIS, direct debit) across `.go` files — zero
  matches outside test fixture strings/comments (e.g. `EndpointURL:
  ".../v1.0/transfer-va"` is just realistic test data, not a per-service
  type). Slice 4 and per-service scope creep: none found.
- Exported surface of `transport.go`/`token.go` matches spec exactly:
  `Envelope`, `Transport`, `Transport.Do`, `GrantType` + 3 constants,
  `Token`, `TokenManager` + `AccessTokenB2B`/`AccessTokenB2B2C`. No extra
  exported helpers, no invented config structs.

## Point-by-point findings from the assigned checklist

1. **Caching genuinely works, not just "no error."** Read
   `TestTokenManager_AccessTokenB2B_CachesAndRefetchesAfterExpiry`
   (`token_test.go:40-115`): it asserts `reqCount` == 1 after the first
   call, == 1 again after a call at `+800s` (before the `900s - 30s`
   safety-margin expiry), and == 2 after a call at `+930s` (past it). This
   is a genuine request-count assertion, not an error-only check. Confirmed
   by re-running with `-count=20 -shuffle=on`, no flake.

2. **Access-token requests are unconditionally asymmetric.** `TokenManager`
   has no `Symmetric` field at all, and `accessTokenHeaders` (`token.go:82`)
   has exactly one code path: `SignAsymmetric(m.Signer, ...)`. This is
   structurally guaranteed, not just tested — there is no branch to get
   wrong. `TestTokenManager_AccessTokenB2B_SignsAsymmetricNeverSymmetric`
   additionally recomputes `BuildStringToSignAccessToken` from the
   timestamp the server actually received and verifies it via
   `VerifyAsymmetric`, so the test would catch a formula regression, not
   just a "signature present" check.

3. **`Transport.Do` error propagation is correct.**
   `TestTransportDo_PropagatesHeaderBuildErrorUnchanged` calls `hb.Build()`
   itself with a malformed `HeaderBuilder` (empty `ExternalID`) and asserts
   `Do`'s returned error string is byte-identical to `Build`'s own error —
   proving `Do` doesn't wrap/mutate it (`transport.go:40-43` returns `err`
   directly, unwrapped). `TestTransportDo_NonTwoXXStatusIsNotAnError` posts
   a 401 HTTP status with `responseCode: "4017301"` and asserts `Do`
   returns `nil` error while `env.ResponseCode` carries the code for the
   caller to run through `ResponseCodeError` — matches spec exactly.
   Conversely, `TokenManager`'s own methods (`envelopeError`,
   `token.go:112-124`) do turn a non-2xx `responseCode` into a returned
   error, and this is exercised by
   `TestTokenManager_AccessTokenB2B_ErrorResponseCode` /
   `..._AccessTokenB2B2C_ErrorResponseCode` across 400/401/500, each
   asserting `errors.Is` against the correct sentinel.

4. **Body size bound is real, not fabricated.** `transport.go:18`:
   `const maxResponseBytes = 10 << 20 // 10 MiB`, applied via
   `io.LimitReader(resp.Body, maxResponseBytes)` in both `transport.go:63`
   and `token.go:147`. Reasonable bound, actually wired into both read
   paths, not just declared and unused.

5. **No Slice 4 / per-service leakage** — confirmed by grep, see above.

## Two real findings (from independent review, not in the original checklist)

### Major — B2B mutex is held across the entire HTTP round-trip

`AccessTokenB2B` (`token.go:180-203`) does `m.mu.Lock(); defer
m.mu.Unlock()` and holds that lock through `doAccessTokenRequest`, i.e.
through the full network call and the default 30s client timeout. This
technically satisfies the spec's "guarded by a mutex for concurrent
callers," and `TestTokenManager_AccessTokenB2B_ConcurrentCallersShareOneFetch`
passes — but the test can't discriminate a coarse lock from a proper
singleflight-style refresh, since both produce exactly one HTTP request for
N concurrent callers. The actual cost: a slow or hung token endpoint blocks
*every* caller, including ones whose own `ctx` was already cancelled, for
up to the full client timeout, with no way out via `ctx.Done()`. Fix:
release the lock before the HTTP call, re-check the cache after
re-acquiring it (double-checked locking), so only the fetch itself races
and losers reuse the winner's result rather than blocking on the mutex for
the full request duration. (`x/sync/singleflight` is off-limits per the
stdlib-only constraint — hand-roll the recheck, it's ~5 lines.)

### Major — non-2xx responses with an empty/unparseable `responseCode` are not surfaced as auth/HTTP errors

`envelopeError` (`token.go:112-114`) treats an empty `responseCode` as
success (`return nil`), and `doAccessTokenRequest` never inspects
`resp.StatusCode` anywhere. If a gateway returns HTTP 401 with a body that
doesn't carry the `responseCode` field (e.g. a differently-shaped error
body, or a proxy/WAF error page), `parsed.ResponseCode` is `""`,
`envelopeError` passes it as non-error, and the code falls through to the
`AccessToken == ""` check (`token.go:159`), producing the generic error
`"snap: token manager: response has no accessToken"` instead of anything
that lets a caller `errors.Is(err, snap.ErrUnauthorized)`. This is
distinct from — and does not contradict — the `Transport.Do` behavior
(where ignoring HTTP status is correct and spec-mandated); for
`TokenManager` the spec explicitly requires error responses to be
"surfaced as the returned error," and an HTTP-level auth failure with a
non-standard body currently isn't surfaced as one. No required test
exercises this path (all `ErrorResponseCode` tests send a well-formed
`responseCode` alongside the non-2xx status). Fix: in
`doAccessTokenRequest`, when `parsed.ResponseCode == ""` and
`resp.StatusCode` is not 2xx, fall back to a status-code-derived error
instead of silently treating it as fine.

## Minor issues

1. **Unguarded shared reads in `TestTokenManager_AccessTokenB2B_SignsAsymmetricNeverSymmetric`** (`token_test.go:159-190`): `gotTimestamp`/`gotSignature` are written by the httptest handler goroutine and read by the test goroutine with no mutex, unlike the equivalent pattern in `transport_test.go:59` which correctly uses one. In practice this doesn't race in `go test -race` because the HTTP response read-out forces a happens-before edge, but it's an inconsistent, fragile pattern next to a sibling file that does it correctly. Fix: mirror `transport_test.go`'s mutex-guarded capture for consistency.
2. **`parseExpiresIn` only accepts `expiresIn` as a JSON string.** `accessTokenResponse.ExpiresIn` is typed `string`, so a provider sending an unquoted JSON number (`"expiresIn": 900`) fails the whole response decode with a generic `"decode response body"` error rather than a targeted one. Spec cites the standard's own `"900"` (quoted) example, so this is defensible as in-spec, but no test covers the unquoted-number case and the failure mode is opaque. Low priority; note for Slice 4+ hardening if provider variance shows up in practice.

## What improved since last iteration

Slice 3 correctly reuses Slice 1/2 primitives end-to-end (`HeaderBuilder`,
`BuildStringToSignAccessToken`, `SignAsymmetric`, `ParseResponseCode`,
`ResponseCodeError`) rather than duplicating logic — no parallel signing or
header-building path was invented for the token manager. The
`TestTransportDo_SendsSignedHeaders` test's pattern of recomputing the
expected `stringToSign` from what the server actually received and
verifying it, rather than just asserting a non-empty header, is genuinely
strong test design and should be the template going forward.

## Scores

| Dimension | Score | Notes |
|---|---|---|
| Correctness | 9/10 | Build/vet/gofmt clean; race-free across 30 repeated/shuffled runs; every required test case present and genuinely discriminating (request-count assertions, signature recomputation, `errors.Is` checks). Docked 1 for the empty-`responseCode`/non-2xx gap in `doAccessTokenRequest`. |
| Security | 8/10 | No secret material logged/leaked; access-token signing is structurally (not just behaviorally) asymmetric-only. Docked for the same responseCode/HTTP-status gap (a misclassified 401 is a security-relevant robustness miss) and the lock-held-across-IO availability concern. |
| API shape | 9/10 | Exact match to spec's `Envelope`/`Transport`/`GrantType`/`Token`/`TokenManager` shapes, field names, and method signatures. Zero scope creep, zero Slice 4 leakage, zero per-service types. |
| Test quality | 8/10 | Table-driven, stdlib-only, tests fail meaningfully on formula/behavior regressions (not error-only checks). Docked for the unguarded shared-variable pattern in one test and the missing empty-responseCode-with-bad-status case. |
| Idiomatic Go | 8/10 | Clean `%w` wrapping throughout, no invented interfaces/factories/generics, small unexported helpers. Docked for holding the mutex across the full HTTP round-trip in `AccessTokenB2B` — a known lock-during-IO smell a senior reviewer would flag even though it satisfies the letter of the spec. |

**Weighted total = (2×9 + 8 + 9 + 8 + 8) / 6 = 51 / 6 = 8.5/10**

## Verdict: PASS (threshold: all dimensions ≥8, none <6 — met; weighted 8.5)

## Specific suggestions for next iteration (or a fast-follow fix)

1. In `AccessTokenB2B`, release the mutex before the HTTP call and
   re-check `now.Before(m.expiresAt)` after re-acquiring it, so a slow
   token endpoint doesn't serialize every caller for the full request
   duration.
2. In `doAccessTokenRequest`, fall back to a status-derived error when
   `resp.StatusCode` is non-2xx but `parsed.ResponseCode` is empty/absent,
   instead of only trusting the `responseCode` field.
3. Guard `gotTimestamp`/`gotSignature` in
   `TestTokenManager_AccessTokenB2B_SignsAsymmetricNeverSymmetric` with the
   same mutex pattern already used in `transport_test.go`.
