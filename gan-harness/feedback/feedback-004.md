# Evaluation — Iteration 004 (Slice 2 re-verification after Test Quality fix)

Commit evaluated: `ad0036d` ("fix: distinguish absent header from empty-value
header in tests"), on top of `13e15ea` (Slice 2 feature commit) and
`546eb68`/`2748c83` (Slice 1 + docs). Files in scope: `header.go`,
`header_test.go`, `profile.go`, `profile_test.go`, `responsecode.go`,
`responsecode_test.go`, all at repo root. Mode: code-only (Go library, no
UI). Branch left untouched throughout — no git command that changes
branch/HEAD was run; only `status`, `log`, `show --stat`, and `diff` (all
read-only).

## Blast-radius check (fix scope discipline)

`git show --stat ad0036d`: touches **only** `header_test.go`
(+15/-6 lines). `git diff 13e15ea ad0036d -- header.go profile.go
responsecode.go signing.go signing_test.go profile_test.go
responsecode_test.go`: **empty** — every one of those files is
byte-identical to the state feedback-003 already read and verified. This is
the discriminating check the task asked for ("a fix can regress something
else"): the fix commit could not have regressed Correctness, Security, API
shape, or Idiomatic Go, because it did not touch any file those dimensions
depend on. Their feedback-003 scores are inherited by file-identity, not
re-asserted from memory. `responsecode.go`/`responsecode_test.go` being
identical also means Mutations A and B (sentinel-mapping swap, field-order
swap) from feedback-003 carry forward without re-running — the code paths
they probed are unchanged.

## Commands run (actual output)

```
$ go build ./...
(exit 0, no output)

$ go vet ./...
(exit 0, no output)

$ gofmt -l .
(no output — clean)

$ go test ./... -race -v -count=1
PASS — all tests and subtests pass, including every Slice 1 test and the
new/changed Slice 2 tests. No failures, no race, no skip.
ok  github.com/koriebruh/go-snap-bi  1.179s
```

No build failure, no vet finding, no gofmt diff, no race — no automatic-0
trigger.

## The fix itself

`header_test.go` now has:

```go
func headerAbsent(h http.Header, k string) bool {
	_, ok := h[http.CanonicalHeaderKey(k)]
	return !ok
}
```

used at all three "should be absent" sites: line 67 (`Origin`, in
`TestHeaderBuilder_Build_B2B`), lines 72–76 (the B2B2C-only-headers loop,
same test), and lines 186–190
(`TestHeaderBuilder_Build_B2B2C_OptionalFieldsOmittedWhenEmpty`).

Notably, the fix **exceeds** feedback-003's suggested fix in one respect:
feedback-003's exact-fix snippet indexed the map with a literal canonical
string (`h["X-Latitude"]`) and explicitly warned that using the wrong
casing "silently never matches and reintroduces the same false-pass in a
new form — this is the trap the next iteration is most likely to hit." The
shipped fix instead calls `http.CanonicalHeaderKey(k)` on the caller-supplied
key at call time, so every call site can pass the header's natural
wire-format spelling (`"X-Ip-Address"`, `"X-Latitude"`, etc. — as it already
does) without the caller having to get the canonicalization right by hand.
This closes the exact trap feedback-003 predicted, structurally rather than
by convention.

## Mutation re-verification (executed against isolated scratch copies)

Both mutations run in fresh copies of the repo under
`/tmp/.../scratchpad/mutest_v2_C` and `mutest_v2_D`, confirmed via
`git status --short` in the tracked repo directory to have touched nothing
tracked (only the new, not-yet-written `gan-harness/feedback/` directory is
untracked).

**Mutation C — remove the `if b.Origin != ""` guard in `mutest_v2_C`,
always `h.Set("ORIGIN", b.Origin)`:**

```
--- FAIL: TestHeaderBuilder_Build_B2B
    header_test.go:68: ORIGIN header present = "", want absent (not supplied)
--- FAIL: TestHeaderBuilder_Build_B2B2C_OptionalFieldsOmittedWhenEmpty
    header_test.go:188: header "Origin" present = "", want absent when not supplied
```

Previously (feedback-003): 0 of 2 assertion sites caught this. Now: **2 of
2**. ✅ Confirmed fixed.

**Mutation D — remove all five per-field guards inside the `b.B2B2C` block
in `mutest_v2_D`, always set unconditionally:**

```
--- FAIL: TestHeaderBuilder_Build_B2B2C_OptionalFieldsOmittedWhenEmpty
    header_test.go:188: header "Authorization-Customer" present = "Bearer ", want absent when not supplied
    header_test.go:188: header "X-Ip-Address" present = "", want absent when not supplied
    header_test.go:188: header "X-Device-Id" present = "", want absent when not supplied
    header_test.go:188: header "X-Latitude" present = "", want absent when not supplied
    header_test.go:188: header "X-Longitude" present = "", want absent when not supplied
```

Previously (feedback-003): 1 of 5 fields caught (`Authorization-Customer`
only, by accident of its `"Bearer "` prefix making the empty case
non-empty). Now: **5 of 5**. ✅ Confirmed fixed.

**Net result: 6 of 6 optional headers (`ORIGIN` + the 5 B2B2C-only fields)
are now genuinely discriminating for the "absent when not supplied" claim,
versus 1 of 6 before.** The critical issue from feedback-003 is closed, and
the fix is proven closed by re-running the identical mutation methodology,
not by re-reading the code and assuming.

`TestHeaderBuilder_Build_CustomProfileChangesTimestamp` and
`TestHeaderBuilder_Build_Asymmetric` were unaffected by either mutation, as
expected (they don't touch the mutated guards).

## One residual gap found (not present in feedback-003's scope, found on
this pass)

Reading `header.go:55-63`:

```go
if b.Symmetric {
	signature = SignSymmetric(b.ClientSecret, stringToSign)
} else {
	var err error
	signature, err = SignAsymmetric(b.Signer, stringToSign)
	if err != nil {
		return nil, fmt.Errorf("snap: build headers: %w", err)
	}
}
```

The `SignAsymmetric` error path (`return nil, ...`) is exercised by **zero**
tests. `TestHeaderBuilder_Build_Asymmetric` is the only `Symmetric: false`
test in `header_test.go`, and it always supplies a valid, freshly generated
2048-bit RSA key — `SignAsymmetric` never fails on that input. A mutation
that replaced the `if err != nil { return nil, ... }` block with a no-op
(swallowing the error and returning `nil` for the error return only) would
pass the entire suite undetected, since no test ever drives `Signer` into
an error-returning state (e.g. `nil` signer, or a signer whose `Public()`/
`Sign()` returns an error). This is a real, un-exercised branch, not a
style nitpick — it is exactly the class of gap the rubric penalizes
(branch with no test proving it fires correctly).

Fix: one additional test, e.g. `Symmetric: false, Signer: nil` (or a small
stub `crypto.Signer` whose `Sign` returns an error), asserting
`Build()` returns a non-nil error whose message/wrapped chain traces back to
the underlying `SignAsymmetric` failure.

## Re-confirmation of feedback-003's other findings (all files byte-identical)

- **API shape**: `Profile`, `DefaultProfile`, `HeaderBuilder`,
  `ParseResponseCode`, sentinel set, `ResponseCodeError` — all unchanged
  from feedback-003's read, all still match spec.md verbatim (verified by
  the diff being empty, not by re-reading and re-asserting from scratch).
- **Correctness**: `ParseResponseCode`'s length/digit validation before
  slicing, `ResponseCodeError`'s parse-first-then-map behavior, the
  custom-`Profile` hook's genuine behavioral fork — all unchanged, all still
  hold.
- **Security**: no secret material logged/printed anywhere in the six
  files (still zero `log.`/`fmt.Print`/`panic(` matches on the unchanged
  production files); `ParseResponseCode` still cannot panic on malformed
  input.
- **Idiomatic Go**: no unrequested abstractions introduced by the fix
  commit; `headerAbsent` is a small, single-purpose unexported test helper,
  not scope creep.

## Scores

| Criterion | Score | Rationale |
|---|---|---|
| Correctness | 9/10 | Build/vet/gofmt/race-test all clean. `header.go`, `profile.go`, `responsecode.go` byte-identical to the feedback-003 state that was fully mutation- and read-verified correct. Score inherited by confirmed file-identity, not re-derived from a fresh read-through. |
| Security | 9/10 | Byte-identical production files to feedback-003's verified-clean state (no secret logging, no panic surface). Unchanged. |
| API shape fidelity | 9/10 | Byte-identical production files; all signatures/names/struct shapes still match spec.md verbatim as previously confirmed. |
| Test quality | 9/10 | The specific critical issue from feedback-003 is confirmed closed by re-running the identical Mutation C and Mutation D methodology: 6 of 6 previously-blind "absent when not supplied" assertions are now genuinely discriminating (up from 1 of 6). The fix's use of `http.CanonicalHeaderKey` at the helper level is more robust than the literal-map-index fix feedback-003 suggested, closing the casing trap that feedback-003 explicitly flagged as the likely next failure mode. Not a 10: `header.go`'s `SignAsymmetric` error-return branch (lines 60-62) is exercised by zero tests — every `Symmetric: false` test path supplies a valid key, so a mutation silencing that error path would pass the suite undetected. This is a newly-identified, real, un-exercised branch (see above), one test short of full coverage on the error path. |
| Idiomatic Go / simplicity | 9/10 | Byte-identical production files to the previously-verified-clean state; the test-only fix adds one small, appropriately-scoped helper (`headerAbsent`) with no unrequested abstraction. Unchanged. |

**Weighted total** = (2×9 + 9 + 9 + 9 + 9) / 6 = (18 + 9 + 9 + 9 + 9) / 6 =
**54 / 6 = 9.00/10**

## Verdict: PASS

All dimensions ≥8 (all are 9), none below 6 — clears this project's stricter
per-dimension pass bar (all dimensions ≥8, none <6) as well as the general
7.0 threshold.

## Critical Issues (must fix)

None. The prior critical issue (non-discriminating "absent" assertions in
`header_test.go`) is confirmed closed by re-executed mutation testing.

## Major Issues (should fix)

None.

## Minor Issues (nice to fix)

1. **`SignAsymmetric`'s error path in `HeaderBuilder.Build()` is untested**
   (`header.go:58-62`) → Add a test with `Symmetric: false` and a
   `Signer` that returns an error on `Sign` (e.g. `Signer: nil`, which will
   panic-or-error depending on `SignAsymmetric`'s nil-handling — check
   Slice 1's behavior first — or a small stub `crypto.Signer`), asserting
   `Build()` returns a non-nil error that wraps it. This is the one
   concretely un-exercised branch found on this pass.
2. `ParseResponseCode`'s manual `int(code[0]-'0')*100 + ...` arithmetic
   could be `strconv.Atoi(code[:3])` for marginally better readability —
   carried forward from feedback-003, still open, still purely cosmetic,
   not scored against.
3. `HeaderBuilder.ClientSecret` is a plaintext exported field with no
   `String()`/`GoString()` redaction, so `%+v`/`%#v` formatting would leak
   it — carried forward from feedback-003, still a spec-mandated shape not
   a defect, flagged for awareness only.

## What Improved Since Last Iteration

- The critical Test Quality gap from feedback-003 is closed: `header_test.go`
  now uses `headerAbsent` (map-presence check via
  `http.CanonicalHeaderKey`) at all three "should be absent" sites, and
  re-running the exact same Mutation C and Mutation D methodology confirms
  6 of 6 previously-blind assertions now fail correctly when the underlying
  guard is removed (up from 1 of 6).
- The fix is more robust than the literal fix feedback-003 proposed:
  canonicalizing inside the helper (rather than requiring each call site to
  pass an already-canonical key) eliminates the exact casing trap
  feedback-003 called out as the most likely way the fix could go wrong.
- The fix commit (`ad0036d`) has a clean, narrow blast radius — confirmed
  via `git diff 13e15ea ad0036d` to touch only `header_test.go`, meaning
  zero risk of regression to `header.go`, `profile.go`, or `responsecode.go`
  production logic or their tests.

## What Regressed Since Last Iteration

None. The fix is scoped to exactly the one file the prior feedback named,
confirmed by diff, and introduces no new failures in the full test suite.

## Specific Suggestions for Next Iteration

1. Add one test covering `SignAsymmetric`'s error path inside
   `HeaderBuilder.Build()` (see Minor Issue #1) to close the last
   un-exercised branch in this slice's code.
2. Optional, not required: apply the same cosmetic `strconv.Atoi` cleanup
   and consider documenting the `ClientSecret` exposure risk (both carried
   forward from feedback-003, neither blocking).
