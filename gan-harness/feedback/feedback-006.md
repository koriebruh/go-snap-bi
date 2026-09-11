# Evaluation — Slice 4 (KeyStore, ServerVerifier — final Phase 1 slice) — commit 7773403

## Verification performed

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `gofmt -l .` — no output (all files formatted).
- `go test ./... -race -v -count=1` — all tests pass, including every
  required table case from `spec.md`'s Slice 4 "Required tests" section.
- `go test ./... -race -count=10` — pass, no flakiness.
- `git show --stat 7773403` — confirms the commit touched only two new
  files (`verify.go`, `verify_test.go`, 510 insertions, 0 deletions). No
  regression into Slice 1/2/3 files.
- `go doc -all .` exported-surface dump — package-wide exported identifiers
  are exactly the accumulated Slice 1–4 surface (`KeyStore`, `SignatureMode`,
  `IncomingRequest`, `ServerVerifier` + its two methods, plus everything from
  prior slices). No per-service types, no extra exported helpers.
- `go.mod` has no `require` block; `verify_test.go` imports only `crypto`,
  `errors`, `fmt`, `testing`, `time` — stdlib only, no testify/gomock.

## Point-by-point findings from the assigned checklist

1. **`go build`/`go vet`/`gofmt`/`go test -race` all clean**, including
   `-count=10` for flakiness — confirmed, no flakes, no races.

2. **`VerifyAccessTokenRequest` is unconditionally asymmetric regardless of
   `Mode`** — confirmed structurally, not just by test. The method (`verify.go:96-109`)
   has no branch on `v.Mode` at all; it always calls `KeyStore.PublicKey` +
   `VerifyAsymmetric`. `TestServerVerifier_VerifyAccessTokenRequest_IgnoresMode`
   (`verify_test.go:356-368`) exercises both `SignatureModeSymmetric` and
   `SignatureModeAsymmetric` on the same `ServerVerifier` against the same
   asymmetrically-signed request and asserts both succeed — this is a real
   discriminating test (a regression that made the method branch on `Mode`
   would fail the symmetric-mode subtest, since a symmetric verify against
   an RSA-signed stringToSign would not match).

3. **`VerifyTransactionRequest` branches correctly on `Mode`.**
   `verify.go:115-142`: symmetric path uses `KeyStore.ClientSecret` +
   `VerifySymmetric`, asymmetric path uses `KeyStore.PublicKey` +
   `VerifyAsymmetric`, and `stringToSign` is built once with
   `symmetric := v.Mode == SignatureModeSymmetric` feeding both the formula
   selection and the verification branch — a single source of truth, so the
   two can't disagree. `VerifySymmetric`'s `bool` `false` is correctly
   converted into a real error via the `ErrSignatureMismatch` sentinel
   (`verify.go:128-130`), exactly as the spec suggests ("e.g. a new
   `ErrSignatureMismatch` sentinel"), not silently returned/ignored.

4. **Timestamp freshness is genuinely wired, not decorative.**
   `checkFreshness` (`verify.go:75-91`) skips parsing entirely when
   `TimestampWindow <= 0` (confirmed by
   `TestServerVerifier_TimestampFreshness/.../window_zero_accepts_wildly_old_timestamp`,
   which uses a 24h-old timestamp with `window: 0` and asserts success), and
   fails symmetrically in both directions (`within window past/future` pass,
   `outside window past/future` fail) for both `VerifyAccessTokenRequest`
   and `VerifyTransactionRequest`. Crucially, it parses via
   `v.profile().TimestampLayout()`, not a hardcoded
   `DefaultTimestampLayout` — proven by
   `TestServerVerifier_ProfileHookAffectsTimestampParsing`
   (`verify_test.go:319-354`), which signs a request with a custom
   `fixedLayoutProfile` timestamp format (`"2026"`), shows it verifies with
   that `Profile` set, and then shows the *same* request fails once
   `v.Profile` is reset to `nil` (falls back to `DefaultProfile{}`) — a
   genuinely load-bearing test, not a no-op assertion.

5. **`KeyStore` lookup failures are a distinct, non-nil error, never
   conflated with signature mismatch.**
   `TestServerVerifier_KeyStoreLookupFailureIsNotSwallowed`
   (`verify_test.go:202-246`) covers all three call sites (access-token,
   transaction-symmetric, transaction-asymmetric) with an unknown
   `clientKey`, asserts `errors.Is(err, errUnknownClientKey)` (the fake
   store's own sentinel), and explicitly asserts
   `!errors.Is(err, ErrSignatureMismatch)` for the access-token case — this
   is exactly the kind of test that would catch an implementation that
   collapsed "key not found" and "signature didn't verify" into the same
   generic error.

6. **Genuine round-trip against Slice 1's real signing functions.** Every
   fixture in `verify_test.go` (`validAccessTokenRequest`,
   `validTransactionRequest`, `accessTokenAt`/`transactionAt` in the
   freshness test) calls `BuildStringToSignAccessToken`/
   `BuildStringToSignTransaction` + `SignSymmetric`/`SignAsymmetric` from
   `signing.go` to construct fixtures — no hand-rolled fake signature
   scheme. RSA keys come from `testRSAKeys` (`signing_test.go:35`, a
   `sync.OnceValue`-memoized pair of real 2048-bit `rsa.GenerateKey` keys
   shared with Slice 1's own tests), so this is real crypto, not a stub.
   `tamperHex` (`signing_test.go:26`) flips the last hex character with an
   explicit guard against a same-value no-op, so tamper assertions can't
   accidentally pass by mutating a signature into itself.

7. **No scope creep.** `git show --stat 7773403` confirms only `verify.go`/
   `verify_test.go` were added — no edits to `signing.go`, `header.go`,
   `responsecode.go`, `token.go`, or `transport.go`. `go doc -all .`
   confirms the exported surface is exactly `KeyStore`, `SignatureMode` (+2
   constants), `IncomingRequest`, `ServerVerifier` (+2 methods) added on top
   of the prior three slices — no per-service types, no invented config
   structs, no extra exported helpers.

8. **No third-party test dependencies** — `verify_test.go` imports only
   `crypto`, `errors`, `fmt`, `testing`, `time`.

## One real finding beyond the assigned checklist: unguarded nil `KeyStore` panics

`ServerVerifier.Profile` and `.Now` both have nil-guards (`profile()`/`now()`,
`verify.go:57-69`) consistent with the spec's "optional; nil means
`DefaultProfile{}`" / "nil means `time.Now`" language. `KeyStore` has no such
guard, and both `VerifyAccessTokenRequest` and `VerifyTransactionRequest` call
`v.KeyStore.PublicKey(...)`/`v.KeyStore.ClientSecret(...)` directly. Confirmed
empirically:

```
v := &ServerVerifier{}
v.VerifyAccessTokenRequest(IncomingRequest{ClientKey: "x"})
// panic: runtime error: invalid memory address or nil pointer dereference
```

The spec doesn't mark `KeyStore` optional the way it marks `Profile`/`Now`
optional, so a nil `KeyStore` is arguably caller misconfiguration rather than
a documented zero-value case, and the failure mode is fail-closed (a panic
stops verification rather than silently accepting a request), which is the
safer of the two bad outcomes. Still, this repo already took a "panic safety"
finding on Slice 3 (commit `76e69d3`) for exactly this class of issue, and a
signature-verification entry point panicking on a missing dependency — rather
than returning a clear configuration error — is inconsistent with the
defensive pattern already applied twice on the same struct. No test in
`verify_test.go` exercises a nil `KeyStore`. This is a Major issue, not
Critical: it doesn't produce an incorrect verify result, and it fails loudly
rather than fail-open.

**Fix:** add `if v.KeyStore == nil { return errors.New("snap: verify: KeyStore is nil") }` (or a typed sentinel) as the first line of both methods, and add a
one-case test asserting a clear error instead of a panic.

## Minor issues

1. **No positive `errors.Is(err, ErrSignatureMismatch)` assertion.** The
   suite proves `ErrSignatureMismatch` is never returned for a *lookup*
   failure (`verify_test.go:216-217`), but no test asserts it *is* returned
   for the `"tampered signature"` case in either round-trip table — the
   sentinel's reachability on the actual mismatch path is verified only by
   reading the code (`verify.go:128-130`, `139`), not by the suite itself.
   Fix: add `if !errors.Is(err, ErrSignatureMismatch) { t.Fatalf(...) }`
   to the `"tampered signature"` case in
   `TestServerVerifier_VerifyTransactionRequest_RoundTrip`.
2. **`VerifyTransactionRequest` proves the signature covers `AccessToken`
   (symmetric mode); it does not itself validate that the bearer token is
   authentic/unexpired.** That's correctly out of scope for a signature
   verifier, but a one-line doc comment on `IncomingRequest.AccessToken` or
   the method itself would prevent an integrator from assuming signature
   verification alone authenticates the token.
3. **An out-of-range `SignatureMode` (e.g. `SignatureMode(7)`) silently
   takes the asymmetric branch** (`symmetric := v.Mode ==
   SignatureModeSymmetric` is `false` for anything other than the symmetric
   constant). Fail-closed in the sense that it still requires a valid
   signature, so genuinely low priority — worth a one-line note, not a fix.
4. **`now()`/`profile()` are duplicated verbatim between `ServerVerifier`
   (`verify.go:57-69`) and `TokenManager` (`token.go:70-83`).** This is the
   correct call under the project's own constraints (no shared
   interface/mixin was asked for, and inventing one for two three-line
   methods would itself be the kind of unrequested abstraction the rubric
   penalizes) — noted for completeness, not a deduction.

## What improved since last iteration

Slice 4 correctly reuses Slice 1's `BuildStringToSignAccessToken`/
`BuildStringToSignTransaction`/`SignSymmetric`/`SignAsymmetric`/
`VerifyAsymmetric` and Slice 2's `Profile`/`DefaultTimestampLayout`/
`truncateForError` end-to-end rather than reinventing any of them —
`truncateForError` in particular (originally written for
`responsecode.go`) is reused as-is for the untrusted timestamp string in
`checkFreshness`'s error message, which is exactly the kind of cross-slice
reuse the harness is testing for. The freshness test's method-table pattern
(`verify_test.go:287-316`, iterating both `VerifyAccessTokenRequest` and
`VerifyTransactionRequest` through the same table of window/offset cases) is
tighter than Slice 3's equivalent coverage and avoids duplicating the table
twice.

## Scores

| Dimension | Score | Notes |
|---|---|---|
| Correctness | 9/10 | Build/vet/gofmt clean; race-free across repeated runs; every required test case present and genuinely discriminating; formulas and branching match spec exactly. Docked 1 for the unguarded nil-`KeyStore` panic — a basic misconfiguration crashes instead of erroring. |
| Security | 9/10 | Real RSA/HMAC round-trips via Slice 1 primitives (no fake crypto); lookup failures never conflated with signature-mismatch failures; no secret material leaked into errors; untrusted timestamp truncated before interpolation. Docked 1 for the same nil-`KeyStore` panic — a signature-verification entry point should fail closed with a clear error, not a runtime panic, on missing configuration. |
| API shape | 10/10 | `KeyStore`, `SignatureMode`, `IncomingRequest`, `ServerVerifier` and both methods match the spec's struct/interface/signature shapes and field order exactly. Zero scope creep confirmed via `git show --stat` and `go doc -all .` — no per-service types, no extra exported surface. |
| Test quality | 9/10 | Table-driven, stdlib-only, every required test case present plus genuinely useful extras (`AsymmetricIgnoresAccessToken`, `ProfileHookAffectsTimestampParsing`). Docked for the missing nil-`KeyStore` test and the missing positive `errors.Is(ErrSignatureMismatch)` assertion on the tampered-signature path. |
| Idiomatic Go | 9/10 | Clean `%w`/double-`%w` wrapping, no invented interfaces/factories/generics, correctly declines to DRY up `now()`/`profile()` across structs rather than inventing an unrequested shared abstraction. Docked 1 for the missing nil-guard on a required dependency, inconsistent with the guards already present on `Profile`/`Now` in the same struct. |

**Weighted total = (2×9 + 9 + 10 + 9 + 9) / 6 = 55 / 6 = 9.17/10**

## Verdict: PASS (threshold: all dimensions ≥8, none <6 — met; weighted 9.17)

## Specific suggestions for a fast-follow fix (Phase 1 is otherwise complete)

1. Add `if v.KeyStore == nil { return errors.New("snap: verify: KeyStore is nil") }`
   as the first statement of both `VerifyAccessTokenRequest` and
   `VerifyTransactionRequest`, plus a one-case test per method asserting a
   clear error rather than `recover()`-catching a panic.
2. Add an explicit `errors.Is(err, ErrSignatureMismatch)` assertion to the
   `"tampered signature"` case in
   `TestServerVerifier_VerifyTransactionRequest_RoundTrip` (both symmetric
   and asymmetric subtests) so the sentinel's reachability is verified by
   the suite, not just by reading `verify.go`.
3. Optional doc-comment note on `IncomingRequest.AccessToken` clarifying
   that signature verification proves the token wasn't tampered with in
   transit, not that the token itself is valid/unexpired.
