# Evaluation — Iteration 001 (Slice 1: Signing core)

Commit evaluated: `ca2026b3f6045ab1b482b2fb22a27a07a503705b`
Files: `signing.go`, `signing_test.go`
Mode: code-only (Go library, no UI)

## Commands run (actual output, not self-reported)

```
$ go build ./...
(exit 0, no output)

$ go vet ./...
(exit 0, no output)

$ go test ./... -race -v -count=1
=== RUN   TestSignVerifySymmetric ... --- PASS
=== RUN   TestSignAsymmetricProducesLowercaseHex --- PASS
=== RUN   TestSignVerifyAsymmetric ... --- PASS (5 subtests)
=== RUN   TestVerifyAsymmetricRejectsNonRSAKey --- PASS
=== RUN   TestBuildStringToSignAccessToken --- PASS
=== RUN   TestBuildStringToSignTransaction --- PASS
=== RUN   TestBuildStringToSignTransactionEmptyBody --- PASS
=== RUN   TestParseRSAPrivateKeyPEM ... --- PASS (4 subtests)
PASS
ok  github.com/koriebruh/go-snap-bi  1.056s

$ gofmt -l .
(no output — clean)

$ cat go.mod
module github.com/koriebruh/go-snap-bi
go 1.26.5
(zero require lines — no third-party dependencies anywhere in the module)

$ grep -nE '"github|"golang.org' signing.go signing_test.go
(no matches — stdlib-only imports, confirmed)

$ go doc -all .
(8 exported symbols listed: DefaultTimestampLayout, BuildStringToSignAccessToken,
 BuildStringToSignTransaction, ParseRSAPrivateKeyPEM, SignAsymmetric, SignSymmetric,
 VerifyAsymmetric, VerifySymmetric — exactly the 8 items spec.md asks for, nothing more)

$ grep -nE 'log\.|fmt\.Print|panic\(' signing.go
(no matches — no secret material logged or printed)
```

Build clean, vet clean, race-clean, gofmt-clean. No automatic-0 trigger.

## Manual verification against spec.md

- **Symmetric formula** (`signing.go:87`): `method + ":" + endpointURL + ":" + accessToken + ":" + bodyHashHex + ":" + timestamp` — matches spec.md:35 exactly, AccessToken segment present.
- **Asymmetric formula** (`signing.go:89`): `method + ":" + endpointURL + ":" + bodyHashHex + ":" + timestamp` — matches spec.md:37 exactly, AccessToken segment correctly omitted.
- **`VerifySymmetric`** (`signing.go:32-35`): uses `hmac.Equal([]byte(expected), []byte(signature))` — constant-time, not `==`. Passes the security bullet as literally worded.
- **`SignAsymmetric` signature**: `func SignAsymmetric(signer crypto.Signer, stringToSign string) (string, error)` (`signing.go:41`) — matches the named design decision in spec.md:27 and eval-rubric.md:31-33 character-for-character. No `*rsa.PrivateKey` concrete type, no raw PEM param.
- **Scope discipline**: grepped the file for `Profile`, `HeaderBuilder`, `Transport`, `Envelope`, `TokenManager`, `KeyStore`, `ServerVerifier`, `ParseResponseCode` — none present. Slice 1 boundary respected.
- **Test dependencies**: `signing_test.go` imports only `crypto/ed25519`, `crypto/rand`, `crypto/rsa`, `crypto/sha256`, `crypto/x509`, `encoding/hex`, `encoding/pem`, `strings`, `sync`, `testing` — all stdlib. No testify/gomock/etc.
- **Empty-body case**: `BuildStringToSignTransaction` never special-cases `nil`/`[]byte{}`; `sha256.Sum256(body)` on a nil slice legitimately produces the SHA-256-of-empty-string digest. Test hardcodes the real digest (`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`) independently via `printf '' | sha256sum` rather than deriving it from the implementation, so the test is non-tautological. Verified it's the correct, well-known empty-SHA256 value.
- **RSA round-trip**: `testRSAKeys` generates two real 2048-bit `rsa.GenerateKey` keypairs via `sync.OnceValue`; sign/verify is exercised against real keys, not stubs.
- **`DefaultTimestampLayout`**: `"2006-01-02T15:04:05.000-07:00"` matches spec.md:47's literal example character-for-character. Not covered by any test (see Minor Issues) but not in the spec's "Required tests" list either, so not scored as a skipped required case.

## Findings

### Should-fix

1. **Untyped PEM parse errors, contradicting the rubric's Security and Idiomatic bullets.** `signing.go:98` (`errors.New("snap: parse rsa private key: no PEM block found")`) and `signing.go:109` (`errors.New("snap: parse rsa private key: not an RSA key")`) are both bare `*errors.errorString` values with no exported sentinel and no custom type. Eval-rubric.md's Security section states verbatim: "`ParseRSAPrivateKeyPEM` rejects non-RSA key types and malformed PEM with a **typed error**" — this is not met; a caller cannot `errors.Is`/`errors.As` to distinguish "no PEM block" from "not RSA" from any other failure. The Idiomatic section independently names this exact case: "Errors wrapped with `%w` where a caller might reasonably want `errors.Is`/`errors.As` (e.g. PEM parse failures)." Two of the four PEM failure paths in this function are unmatchable. *(Note: spec.md:44 itself only asks for "a clear error," so this is a rubric-vs-spec tension; I am scoring against the rubric as instructed.)* Fix: export `var ErrNoPEMBlock = errors.New(...)` and `var ErrNotRSAKey = errors.New(...)`, return them via `%w`.
2. **Test doesn't distinguish the two PEM failure modes.** `TestParseRSAPrivateKeyPEM` (signing_test.go:217-238) only asserts `(err != nil) != tt.wantErr` for the "malformed PEM" and "non-RSA key type" cases — it would still pass if both errors collapsed to the same message/type. This mirrors finding #1 and should be tightened once sentinels exist (assert `errors.Is(err, ErrNoPEMBlock)` etc.).

### Minor (nice to fix, not scored against)

1. `DefaultTimestampLayout` has zero test coverage (a `time.Parse`/`Format` round-trip assertion would catch a future typo like `.000` → `.999` or `-07:00` → `Z07:00`, both silent, high-impact bugs in a real SNAP client). Not a required test per spec.md, so not penalized, but worth a one-liner next iteration.
2. `TestBuildStringToSignAccessToken` (signing_test.go:118) is a single assertion, not table-driven, unlike every other test in the file. Cosmetic inconsistency only.
3. `VerifySymmetric`/`VerifyAsymmetric` compare/decode hex case-sensitively as written by `SignSymmetric`/`SignAsymmetric` (both lowercase) — an uppercase-hex signature from a lenient counterparty would fail. Inbound signature verification against third-party formatting is `ServerVerifier` territory (Slice 4), so out of scope here; flagging only for awareness.
4. The generator committed `gan-harness/spec.md` and `gan-harness/eval-rubric.md` in the same commit as the implementation (`ca2026b`, diff stat shows both harness files added). Process note only — no score impact, but harness files should generally already exist before generation starts, not be introduced by the generator's own commit.

### Not findings (checked and cleared)

- `SignAsymmetric` does not type-assert `signer` to `*rsa.PrivateKey` — this is correct per spec.md:29's explicit requirement to accept any `crypto.Signer`; adding a concrete-type check would itself be the deviation.
- `hmac.Equal` on two hex strings of otherwise-equal expected length is not a timing-relevant leak; expected signature length is public information.
- Clean `-race` run is expected and vacuous for this slice (no goroutines/concurrency in scope) — not treated as a quality signal in either direction.

## Scores

| Criterion | Score | Rationale |
|---|---|---|
| Correctness | 9/10 | Build/vet/test/race all pass; both stringToSign formulas verified byte-for-byte against spec; empty-body path correct and non-tautologically tested; real RSA round-trip. No functional bugs found. Not a 10 only because `DefaultTimestampLayout`, a spec-mandated exported const, ships with no verification anywhere (test or otherwise) that it round-trips a real timestamp. |
| Security | 7/10 | `VerifySymmetric` correctly uses `hmac.Equal` (constant-time) — full credit there. No secret material logged, printed, or wrapped into error strings (grepped and confirmed). However `ParseRSAPrivateKeyPEM`'s two reject paths use untyped `errors.New`, directly failing the rubric's explicit "typed error" requirement for this function — this is a named rubric bullet, not an inferred one. |
| API shape fidelity | 9/10 | `SignAsymmetric`'s `crypto.Signer` parameter matches the named design decision exactly. All 6 function names/parameter orders/return types match spec.md verbatim. `go doc -all` confirms exactly 8 exported symbols, zero scope creep, no Slice 2/3/4 surface. Not a 10 because the untyped-error API shape (returning bare `error` with no exported sentinel type where the rubric expects one) is itself a shape gap, distinct from but adjacent to the Security finding. |
| Test quality | 8/10 | Table-driven throughout except one function; stdlib `testing` only (confirmed via import grep); tests are non-tautological (independently hardcoded SHA-256 and formula expectations, not derived from the implementation); every required test case from spec.md's "Required tests" list is present (symmetric round-trip incl. tamper, asymmetric round-trip incl. tamper/wrong-key, formula delta test, empty-body test, all 4 PEM cases). Docked for not asserting distinct error identity in the PEM malformed-vs-non-RSA cases (mirrors the typed-error gap) and for the one non-table test. |
| Idiomatic Go / simplicity | 8/10 | No unrequested abstractions, no interfaces invented for one implementation, no factories, no generics/reflection/unsafe. Most errors correctly wrapped with `%w` (`SignAsymmetric`, `VerifyAsymmetric`'s decode/verify paths, `ParseRSAPrivateKeyPEM`'s PKCS#8 branch). Docked for the two `errors.New` calls in `ParseRSAPrivateKeyPEM` that the rubric explicitly calls out as needing `%w`-wrapped, `errors.Is`-able sentinels. |

**Weighted total** = (2×9 + 7 + 9 + 8 + 8) / 6 = (18 + 7 + 9 + 8 + 8) / 6 = **50 / 6 = 8.33/10**

## Verdict: FAIL

Passing bar is per-dimension: "all dimensions >= 8, no dimension below 6." Security scores **7/10**, below the 8 floor, due to a rubric-named, verifiable gap (untyped PEM errors). The weighted average (8.33) clears the bar, but the rubric's pass condition is the per-dimension floor, not the average — one dimension under 8 fails the run regardless of overall score.

## Fix required to pass

Export two sentinel errors in `signing.go` (e.g. `ErrNoPEMBlock`, `ErrNotRSAKey`), wrap both `ParseRSAPrivateKeyPEM` reject paths with `%w` against them, and add `errors.Is` assertions to the corresponding `TestParseRSAPrivateKeyPEM` subtests. This is a small, localized diff — everything else in the slice (formulas, constant-time compare, `crypto.Signer` shape, scope boundary, test coverage) is already correct and does not need to change.
