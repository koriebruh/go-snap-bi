# Evaluation — Iteration 002 (Slice 1: Signing core, re-eval)

Commit evaluated: `bacc861d9fd4429e44ebdf6e66d42d54b98f9a91` (on top of `ca2026b`, evaluated in feedback-001)
Files changed by this fix: `signing.go`, `signing_test.go`
Mode: code-only (Go library, no UI)

This is a re-verification, not a rubber stamp on the stated fix. Every dimension
from feedback-001 was re-checked from scratch, not assumed to still hold.

## Commands run (actual output, not self-reported)

```
$ go build ./...
(exit 0, no output)

$ go vet ./...
(exit 0, no output)

$ gofmt -l .
(no output — clean)

$ go test ./... -race -v -count=1
=== RUN   TestSignVerifySymmetric ... --- PASS (4 subtests)
=== RUN   TestSignAsymmetricProducesLowercaseHex --- PASS
=== RUN   TestSignVerifyAsymmetric ... --- PASS (5 subtests)
=== RUN   TestVerifyAsymmetricRejectsNonRSAKey --- PASS
=== RUN   TestBuildStringToSignAccessToken --- PASS
=== RUN   TestBuildStringToSignTransaction --- PASS
=== RUN   TestBuildStringToSignTransactionEmptyBody --- PASS
=== RUN   TestParseRSAPrivateKeyPEM ... --- PASS (4 subtests)
PASS
ok  github.com/koriebruh/go-snap-bi  1.094s
```

Build clean, vet clean, race-clean, gofmt-clean. No automatic-0 trigger.

## The fix, verified

`git show bacc861` diff (reproduced and inspected in full):

- `signing.go`: adds `var ( ErrNoPEMBlock = errors.New(...); ErrNotRSAKey = errors.New(...) )`,
  exported at package scope, doc-commented as "Sentinel errors returned by
  ParseRSAPrivateKeyPEM ... wrapped with `%w` so callers can distinguish
  failure modes via `errors.Is`."
- Both `ParseRSAPrivateKeyPEM` reject paths changed from bare `errors.New(...)`
  to `fmt.Errorf("snap: parse rsa private key: %w", ErrNoPEMBlock)` and
  `fmt.Errorf("snap: parse rsa private key: %w", ErrNotRSAKey)` respectively.
- `signing_test.go`: `TestParseRSAPrivateKeyPEM`'s table changed `wantErr bool`
  → `wantErr error`, and the malformed-PEM / non-RSA-key cases now carry
  `ErrNoPEMBlock` / `ErrNotRSAKey` and assert via `errors.Is(err, tt.wantErr)`
  instead of `(err != nil) != tt.wantErr`.
- Diff is minimal and exactly targeted at the flagged gap: 11 lines changed in
  `signing.go`, 22 in `signing_test.go`, two files touched, nothing else moved.

**Mutation test (not just "would catch it mentally" — actually run).** Copied
`signing.go`/`signing_test.go`/`go.mod` to an isolated scratch dir, swapped the
two sentinel arguments at the wrap sites (`ErrNoPEMBlock` ↔ `ErrNotRSAKey`), and
reran `go test -run TestParseRSAPrivateKeyPEM -v`. Result:

```
--- FAIL: TestParseRSAPrivateKeyPEM/malformed_PEM
    error = snap: parse rsa private key: snap: not an RSA key, want errors.Is(err, snap: no PEM block found)
--- FAIL: TestParseRSAPrivateKeyPEM/non-RSA_key_type
    error = snap: parse rsa private key: snap: no PEM block found, want errors.Is(err, snap: not an RSA key)
```

Both subtests fail under the mutation. The `errors.Is` assertions are
genuinely discriminating, not incidentally passing because both errors happen
to be non-nil. This closes out feedback-001's should-fix #2 as well as #1.
(Scratch copy was made and mutated only in an isolated temp directory. Separately ran `git status --short` on the repo after the mutation test: output showed only the new untracked `gan-harness/feedback/` directory from this eval's own output — `signing.go`/`signing_test.go` show no diff, confirming the repository working tree itself was never touched by the mutation.)

## Full re-verification of every other dimension (not assumed to still hold)

- **Symmetric formula** (`signing.go:94`): `method + ":" + endpointURL + ":" + accessToken + ":" + bodyHashHex + ":" + timestamp` — matches spec.md exactly, unchanged from iteration 1.
- **Asymmetric formula** (`signing.go:96`): `method + ":" + endpointURL + ":" + bodyHashHex + ":" + timestamp` — AccessToken segment correctly omitted, unchanged.
- **`VerifySymmetric`**: still `hmac.Equal([]byte(expected), []byte(signature))` — constant-time, untouched by this diff.
- **`SignAsymmetric` signature**: still `func SignAsymmetric(signer crypto.Signer, stringToSign string) (string, error)` — untouched, matches the named `crypto.Signer` design decision.
- **Scope discipline**: re-grepped `signing.go`/`signing_test.go` for `Profile|HeaderBuilder|Transport|Envelope|TokenManager|KeyStore|ServerVerifier|ParseResponseCode` — zero matches. `doc.go` (pre-existing since `ca2026b`, untouched by this commit) contains a package-level doc comment mentioning later-phase concepts, but implements none of them — informational only, not a functional-surface finding, and explicitly out of the diff under review.
- **Exported surface**: `go doc -all .` now lists `DefaultTimestampLayout`, `ErrNoPEMBlock`, `ErrNotRSAKey`, `BuildStringToSignAccessToken`, `BuildStringToSignTransaction`, `ParseRSAPrivateKeyPEM`, `SignAsymmetric`, `SignSymmetric`, `VerifyAsymmetric`, `VerifySymmetric` — 10 exported symbols (8 spec-mandated + the 2 new sentinels). The rubric's API-shape section penalizes "extra public API surface... not asked for," and taken in isolation these two vars are technically that. But the rubric's own Security section explicitly demands a "typed error" for exactly this function, and Idiomatic explicitly asks for `%w`-wrapped sentinels "where a caller might reasonably want `errors.Is`" — citing this exact case. Where the two bullets collide, Security/Idiomatic wins: two sentinel vars scoped tightly to one function's two failure modes is the minimum surface that satisfies the explicit requirement, not scope creep toward Slice 2+.
- **Secret leakage**: re-grepped `signing.go` for `log.|fmt.Print|panic(` — zero matches, unchanged from iteration 1.
- **Test dependencies**: `signing_test.go` imports remain stdlib-only (`crypto/ed25519`, `crypto/rand`, `crypto/rsa`, `crypto/sha256`, `crypto/x509`, `encoding/hex`, `encoding/pem`, `errors` (newly added, stdlib), `strings`, `sync`, `testing`). No third-party test helpers.
- **Empty-body case**: `BuildStringToSignTransaction` logic untouched by this commit; still hashes `nil`/`[]byte{}` correctly without special-casing, tests hardcode the independently-derived SHA-256-of-empty-string digest.
- **RSA round-trip**: `testRSAKeys` generation and asymmetric round-trip tests untouched, still exercised against real 2048-bit keys.
- **`DefaultTimestampLayout`**: unchanged, still `"2006-01-02T15:04:05.000-07:00"`. Manually re-verified by hand against the SNAP spec's `yyyy-MM-ddTHH:mm:ss.SSSTZD` pattern: `yyyy`→`2006`, `MM`→`01`, `dd`→`02`, `HH`→`15`, `mm`→`04`, `ss`→`05`, `SSS`→`.000` (milliseconds), `TZD`→`-07:00` — every field maps correctly to Go's reference-time layout. Confirmed correct by inspection; still has zero test coverage (no `time.Parse`/`Format` round-trip assertion exists anywhere in the file). This is the same gap noted in feedback-001, carried forward unchanged — it is not a new finding introduced by this commit, and a future generator should not spend an iteration "fixing" a value that is already correct; the ask is a coverage gap, not a bug.

### Not findings (checked and cleared, re-confirmed)

- `SignAsymmetric` still does not type-assert to `*rsa.PrivateKey` — correct, per spec.md's explicit `crypto.Signer` requirement.
- `hmac.Equal` on hex strings is not a timing-relevant leak — expected signature length is public.
- `-race` clean is vacuous for this slice (no concurrency in scope) — not a quality signal either direction.

## Findings remaining

### Minor (nice to fix, not scored against — none newly introduced by this commit)

1. `DefaultTimestampLayout` still has zero test coverage. Carried over verbatim from feedback-001; not a regression, not new.
2. `TestBuildStringToSignAccessToken` is still a single assertion, not table-driven, unlike every other test in the file. Cosmetic, unchanged from iteration 1.
3. Case-sensitive hex comparison in `VerifySymmetric`/`VerifyAsymmetric` — still out of scope (Slice 4 / `ServerVerifier` territory), unchanged.

No new issues were introduced by this commit. The diff is surgical: it touches exactly the two reject paths named in feedback-001 and the one test that needed tightening, and nothing else in the file changed behavior.

## Scores

| Criterion | Score | Rationale |
|---|---|---|
| Correctness | 9/10 | Build/vet/test/race all pass; both stringToSign formulas re-verified byte-for-byte against spec; empty-body path correct; real RSA round-trip. Unchanged from iteration 1 — this commit didn't touch signing logic. Still not a 10 solely because `DefaultTimestampLayout` ships with zero test coverage (manually verified correct by hand, see above, but unasserted in code). This is a carried-over gap, not a new finding. |
| Security | 9/10 | `VerifySymmetric` still uses `hmac.Equal` (constant-time). No secret material logged/printed/wrapped (re-grepped, confirmed). **`ParseRSAPrivateKeyPEM` now rejects both failure modes with exported, `%w`-wrapped sentinel errors (`ErrNoPEMBlock`, `ErrNotRSAKey`), directly closing the iteration-1 gap.** Mutation-tested: swapping the sentinels breaks both PEM subtests, confirming the fix is real, not cosmetic. No residual defect was found on inspection — the 9 (vs. 10) is reserved headroom for a library this size, not a deduction for any named gap; treat it as "no work needed here," not an open item. |
| API shape fidelity | 10/10 | All 8 spec-mandated function/const signatures unchanged and verified against spec.md verbatim (names, parameter order, return types). `crypto.Signer` design decision intact. The 2 new exported sentinel vars are additional surface beyond the literal 8-item list, but they exist specifically to satisfy the rubric's own Security/Idiomatic requirement for typed PEM errors — the rubric's bullets collide here and the narrower, function-scoped fix is the correct resolution, not scope creep toward Slice 2+ (`Profile`, `Transport`, `KeyStore`, etc. — none present, re-confirmed by grep). |
| Test quality | 9/10 | Table-driven throughout except one function (unchanged cosmetic gap). Stdlib-only imports re-confirmed. `TestParseRSAPrivateKeyPEM` now asserts `errors.Is(err, tt.wantErr)` against the correct sentinel per case instead of `err != nil` — mutation-tested and confirmed genuinely discriminating (see above), closing iteration-1's should-fix #2. Every required spec.md test case still present and passing. Docked only for the still-non-table `TestBuildStringToSignAccessToken`. |
| Idiomatic Go / simplicity | 10/10 | No unrequested abstractions, no interfaces/factories invented, no generics/reflection/unsafe. The two remaining `errors.New` calls that lacked `%w` wrapping in iteration 1 are now both wrapped with `%w` against exported sentinels — the exact idiomatic-Go gap named in feedback-001 is fully closed with no collateral change. |

**Weighted total** = (2×9 + 9 + 10 + 9 + 10) / 6 = (18 + 9 + 10 + 9 + 10) / 6 = **56 / 6 = 9.33/10**

## Verdict: PASS

All five dimensions are at or above the 8 floor (Correctness 9, Security 9,
API shape 10, Test quality 9, Idiomatic 10), and none is below the 6 floor.
The single dimension that failed the bar in iteration 1 (Security, 7/10) is
now verified — not merely asserted — to be fixed, via a real mutation test on
the sentinel-error wrapping rather than a re-read of the diff. No regression
was found in any of the other four dimensions; every formula, signature, and
scope boundary re-checked from feedback-001 still holds unchanged.

## What improved since last iteration

- `ParseRSAPrivateKeyPEM`'s two reject paths now return exported,
  `%w`-wrapped sentinel errors (`ErrNoPEMBlock`, `ErrNotRSAKey`), closing the
  Security and Idiomatic gaps named in feedback-001.
- `TestParseRSAPrivateKeyPEM` now asserts `errors.Is` against the specific
  expected sentinel per failure mode, not just "an error occurred" — verified
  via mutation testing to be a real, discriminating assertion.

## What regressed since last iteration

None found. The diff is minimal (11 + 22 lines across 2 files) and scoped
exactly to the flagged issue; every other function, formula, test, and scope
boundary was re-verified independently and matches iteration 1's state.

## Specific suggestions for next iteration (optional polish, not required to pass)

1. Add one `time.Parse`/`Format` round-trip test for `DefaultTimestampLayout`
   (e.g. parse a fixed timestamp string, reformat, assert equality) — cheap,
   catches a future silent typo like `.000`→`.999` or `-07:00`→`Z07:00`. Not
   required by spec.md's "Required tests" list, so not scored against, and
   not a reason to touch signing.go itself — the const is already correct.
2. Make `TestBuildStringToSignAccessToken` table-driven for consistency with
   every other test in the file (cosmetic only).

Neither suggestion blocks Slice 1 sign-off; this iteration passes as-is.
