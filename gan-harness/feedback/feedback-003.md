# Evaluation — Iteration 003 (Slice 2: Header assembly + response codes)

Commit evaluated: `13e15eacbcd0b6a3c2846cadbf1a1585cc4a319f` ("feat: implement
Slice 2 header assembly and response codes"), on top of `546eb68` (docs) and
`2748c83` (Slice 1 fix commit). Files in scope: `header.go`, `header_test.go`,
`profile.go`, `profile_test.go`, `responsecode.go`, `responsecode_test.go`,
all at repo root.
Mode: code-only (Go library, no UI).

## Commands run (actual output)

```
$ go build ./...
(exit 0, no output)

$ go vet ./...
(exit 0, no output)

$ gofmt -l .
(no output — clean)

$ go test ./... -race -v -count=1
PASS — all tests and subtests pass, including every Slice 1 test
(unchanged) plus all new Slice 2 tests:
  TestHeaderBuilder_Build_B2B
  TestHeaderBuilder_Build_Asymmetric
  TestHeaderBuilder_Build_B2B2C
  TestHeaderBuilder_Build_B2B2C_OptionalFieldsOmittedWhenEmpty
  TestHeaderBuilder_Build_CustomProfileChangesTimestamp
  TestDefaultProfile_BuildPath (+3 subtests)
  TestDefaultProfile_TimestampLayout
  TestParseResponseCode (+12 subtests)
  TestResponseCodeError (+9 subtests)
ok  github.com/koriebruh/go-snap-bi  1.242s
```

No build failure, no vet finding, no gofmt diff, no race — no automatic-0
trigger.

## Scope and API-shape verification

- `Profile` interface matches spec.md verbatim: `TimestampLayout() string`,
  `BuildPath(serviceGroup, productType string) string`.
- `DefaultProfile` is `struct { Domain, Version string }`, not a zero-field
  singleton — a caller genuinely must construct
  `DefaultProfile{Domain: "...", Version: "v1.0"}`. `BuildPath` defaults
  `Version` to `"v1.0"` only when empty, matching spec's stated default.
  `fixedLayoutProfile` (test-only) embeds `DefaultProfile` and overrides only
  `TimestampLayout`, exactly the "PJP-specific profile" shape spec.md asks
  for.
- `ParseResponseCode(code string) (httpStatus int, serviceCode, caseCode
  string, err error)` — exact signature match.
- Sentinel set exactly matches spec.md's list: `ErrBadRequest`,
  `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrInternalServerError`,
  `ErrServiceUnavailable`, `ErrTimeout`, plus an unexported
  `errUnmappedResponseCode` for the documented "no sentinel" fallback case.
  `ResponseCodeError(code string) error` matches signature and always
  returns non-nil.
- Grepped all six Slice 2 files for `Transport|Envelope|TokenManager|
  KeyStore|ServerVerifier` — zero matches. No Slice 3/4 scope creep.
- `git show --stat` on the Slice 2 commit (`13e15ea`) touches only the six
  files in scope (578 insertions, 0 deletions, 0 other files). `git log -p`
  on `signing.go`/`signing_test.go` confirms their last touch was the prior
  `2748c83` fix commit — Slice 2's commit does not modify Slice 1 files at
  all.
- `go.mod` has no `require` block; grepped all `.go` files for
  `"github.com` / `"golang.org/x` imports — zero matches. Stdlib only,
  including in tests (`testing`, `crypto/rand`, `crypto/rsa`, `errors`,
  `strings`).

## Header assembly — behavior verification

Read `header.go` in full and traced `HeaderBuilder.Build()`:

- Always-present headers (`Content-Type`, `X-TIMESTAMP` via
  `profile.TimestampLayout()`, `X-CLIENT-KEY`, `X-SIGNATURE`, `X-PARTNER-ID`,
  `X-EXTERNAL-ID`, `CHANNEL-ID`, `Authorization: Bearer <token>`) are all set
  unconditionally, matching spec.
- `X-SIGNATURE` is computed by threading `b.Symmetric` through
  `BuildStringToSignTransaction` (Slice 1) and then calling
  `SignSymmetric`/`SignAsymmetric` accordingly — not hardcoded to one path.
  Verified this is a real, non-decorative parameterization: both
  `TestHeaderBuilder_Build_B2B` and `TestHeaderBuilder_Build_Asymmetric`
  independently recompute the expected `stringToSign` from the actual
  `X-Timestamp` placed in the header and verify the signature against it
  (and, for the symmetric case, assert it does *not* verify against the
  wrong-formula string) — this is a genuine round-trip check, not a
  presence-only check.
- `ORIGIN` is set only when `b.Origin != ""`, independent of B2B2C, matching
  spec's placement of that bullet outside the B2B2C-only list.
- B2B2C-only headers (`Authorization-Customer`, `X-IP-ADDRESS`,
  `X-DEVICE-ID`, `X-LATITUDE`, `X-LONGITUDE`) are each gated on both
  `b.B2B2C` and a per-field non-empty check, matching "include only if
  supplied."
- `DefaultProfile{}` is substituted when `b.Profile == nil` — reasonable
  default, not silently broken by a missing Profile.

**Custom-Profile hook genuinely takes effect** (spec explicitly requires
proof of this, not just acceptance-and-ignore): `fixedLayoutProfile`
overrides `TimestampLayout()` to `"2006"` (a 4-character year-only layout).
`TestHeaderBuilder_Build_CustomProfileChangesTimestamp` builds headers with
`DefaultProfile{}` and with `fixedLayoutProfile{}` and asserts the resulting
`X-TIMESTAMP` values differ in length (full layout vs 4 characters) and that the custom one is
exactly 4 characters. This is a real behavioral fork through the interface
seam, confirmed by reading `Build()`'s `time.Now().Format(profile.
TimestampLayout())` call — not stubbed.

## Response code — behavior verification

- `ParseResponseCode` validates length (exactly 7) and digit-only content
  before slicing, so it cannot panic on short/malformed/empty input — traced
  the slicing (`code[3:5]`, `code[5:7]`) and confirmed it is unreachable
  unless `len(code) == 7` has already been checked.
- `ResponseCodeError` parses first; on parse failure it returns the parse
  error directly (not a sentinel) — `TestResponseCodeError`'s "malformed
  code" subtest explicitly asserts this does **not** match `ErrBadRequest`,
  which is the correct behavior (a malformed code is not a 400, it's
  unparseable).
- Sentinel wrapping uses `fmt.Errorf("%w: response code %s", sentinel,
  code)` — `errors.Is` works, and the raw code is recoverable from
  `err.Error()` (tested via `strings.Contains`).

## Mutation testing (executed, not just reasoned about)

All mutations run against isolated scratch copies (`/tmp/.../scratchpad/
mutest_run{2,3,4,5}`), never against the tracked working tree. Confirmed
with `git status --short` in the repo directory that no repo file was
touched.

**Mutation A — swap `serviceCode`/`caseCode` return order in
`ParseResponseCode`:** 7 of 12 `TestParseResponseCode` subtests fail
(all classes except 200, which happens to have `"00"` in both fields).
Test is genuinely discriminating. ✅

**Mutation B — swap the `ErrBadRequest`/`ErrUnauthorized` sentinel mapping
in `ResponseCodeError`'s switch:** `400_bad_request` and `401_unauthorized`
subtests fail immediately. Test is genuinely discriminating. ✅

**Mutation C — remove the `if b.Origin != ""` guard, always
`h.Set("ORIGIN", b.Origin)`:** **all header tests still pass.** Root cause,
confirmed with a standalone probe: `http.Header{}.Set("ORIGIN", "")`
produces `http.Header{"Origin":[]string{""}}`, and `h.Get("Origin")` on that
returns `""` — identical to what `Get` returns for a key that was never set
at all. Every "assert header X is absent when not supplied" check in
`header_test.go` is written as `h.Get(k) != ""`, which cannot distinguish
"header never set" from "header set to an empty string." The check is
non-discriminating for this exact bug class.

**Mutation D — remove all five per-field `if ... != ""` guards inside the
`b.B2B2C` block** (always set `Authorization-Customer`, `X-IP-ADDRESS`,
`X-DEVICE-ID`, `X-LATITUDE`, `X-LONGITUDE` unconditionally):
`TestHeaderBuilder_Build_B2B2C_OptionalFieldsOmittedWhenEmpty` (the test
where `B2B2C: true` and these per-field guards actually matter) fails, but
**only on `Authorization-Customer`** (`"Bearer "` is non-empty even when the
underlying field is `""`, so that one header accidentally becomes
distinguishable). The other four subfields (`X-IP-ADDRESS`, `X-DEVICE-ID`,
`X-LATITUDE`, `X-LONGITUDE`) produce empty-but-present headers under the
mutation and this test still passes. Note: `TestHeaderBuilder_Build_B2B`'s
own "B2B2C-only headers absent" loop (`header_test.go:63-67`) is not
affected by this mutation at all — that test sets `B2B2C: false`, so the
entire `if b.B2B2C` block never executes regardless of the per-field guards,
and those headers are genuinely absent there. That loop is defense-in-depth,
not part of the proven hole. Combined with Mutation C, the confirmed hole is
4 of 5 B2B2C-only optional headers (all but `Authorization-Customer`) in
`OptionalFieldsOmittedWhenEmpty`, plus `ORIGIN` in both tests it appears in —
5 of 6 total optional headers, measured by mutating the `ORIGIN` guard and
the B2B2C per-field guards in separate, independent scratch copies (Mutation
C and Mutation D were not run together), not inferred from one combined
run.

This is not a hypothetical or a style nitpick: it is an empirically
demonstrated case of the rubric's own example scenario ("mutate one
character... confirm the test would catch it") failing to hold, for exactly
the clause the spec's Required Tests section calls out by name ("absent
when not [supplied]").

**Important: the shipped implementation itself is correct.** Every guard in
`header.go` (`if b.Origin != ""`, and the four per-field checks inside
`if b.B2B2C`) is present and behaves correctly. This is purely a test
file gap — `header_test.go` needs a stronger assertion, not `header.go`.
The next iteration should touch only `header_test.go`.

### Exact fix

Replace `h.Get(k) != ""` (`header_test.go:58` for `Origin`, `header_test.go:
64` for the B2B2C-only loop, and `header_test.go:178` in the
`OptionalFieldsOmittedWhenEmpty` test) with a presence check on the
underlying map, e.g.:

```go
if _, ok := h["X-Latitude"]; ok {
    t.Errorf("header %q present, want absent when not supplied", "X-Latitude")
}
```

or equivalently `len(h.Values("X-Latitude")) != 0`. **Caveat worth stating
explicitly:** `http.Header` keys are stored canonicalized
(`textproto.CanonicalMIMEHeaderKey`), so direct map indexing must use the
canonical form (`"X-Latitude"`, `"X-Ip-Address"`, `"Authorization-Customer"`,
`"Origin"`), not the wire-format spelling (`"X-LATITUDE"`). Using the wrong
casing for a map-key check silently never matches and reintroduces the same
false-pass in a new form — this is the trap the next iteration is most
likely to hit while fixing this.

## Scores

| Criterion | Score | Rationale |
|---|---|---|
| Correctness | 9/10 | Build/vet/gofmt/race-test all clean. Every formula, signature, and struct shape verified against spec.md by direct reading (`Profile`, `DefaultProfile`, `HeaderBuilder`, `ParseResponseCode`, sentinels, `ResponseCodeError`). Mutation-tested `ParseResponseCode`'s field split and `ResponseCodeError`'s sentinel mapping — both genuinely correct, not just untested-and-lucky. Custom-`Profile` hook confirmed to actually change output via a real behavioral fork, not accept-and-ignore. The implementation itself has no confirmed defect; the one gap found (see Test quality) is in the test file, not in `header.go`. Reserved headroom, not a specific deduction. |
| Security | 9/10 | No secret material (`ClientSecret`, `Signer`) logged, printed, or wrapped into error strings anywhere in the six files — grepped for `log.`/`fmt.Print`/`panic(`, zero matches. `ParseResponseCode` cannot panic on malformed/short/empty input (validated before slicing, confirmed by tracing and by the empty-string/too-short/non-digit test cases). Slice 1's already-hardened signing primitives (constant-time `VerifySymmetric`, RSA key-size floor) are reused unmodified. Minor observation only, not a deduction: `HeaderBuilder.ClientSecret` is a plaintext exported struct field with no redaction, so `fmt.Printf("%+v", b)` would leak it — this is the spec-mandated shape (a plain struct field for the secret), not a code defect, so it's noted for awareness only. |
| API shape fidelity | 9/10 | `Profile`/`DefaultProfile`/`ParseResponseCode`/sentinel names all match spec.md verbatim (names, parameter order, return types). `DefaultProfile` is a real struct with `Domain`/`Version` fields as explicitly required (not a singleton). No extra exported surface beyond what's asked — `errUnmappedResponseCode` is deliberately unexported. Zero Slice 3/4 leakage (grepped, confirmed). `HeaderBuilder`'s 17-field flat struct is spec-compliant (spec lists exactly this set of inputs) rather than an invented abstraction. |
| Test quality | 7/10 | Table-driven, stdlib-`testing`-only for `profile_test.go` and `responsecode_test.go`, and both mutation-tested clean (Mutation A and B above caught real, non-trivial bugs immediately). **However, mutation testing (Mutations C and D, executed against isolated scratch copies, not just reasoned about) empirically proved that 5 of the 6 "header omitted when not supplied" assertions in `header_test.go` (`ORIGIN`, `X-IP-ADDRESS`, `X-DEVICE-ID`, `X-LATITUDE`, `X-LONGITUDE`) do not actually verify omission — they use `h.Get(k) != ""`, which is indistinguishable from `h.Get(k)` on a header that was `Set` to an empty string. Only `Authorization-Customer` accidentally survives the mutation because of its `"Bearer "` prefix.** This directly undermines the spec's Required Tests clause "absent when not [supplied]," which is the specific behavior this dimension is meant to certify, and is exactly the class of gap the rubric's own scoring guidance calls out ("tests fail meaningfully if the implementation is wrong... not tests that only check no error returned"). This is a real, evidence-backed deduction, not a stylistic one — implementation is correct, only the test's discriminating power is not. |
| Idiomatic Go / simplicity | 9/10 | No unrequested abstractions — `Profile`/`DefaultProfile` are exactly what spec asked for, no factories, no premature generics/reflection/unsafe. `ResponseCodeError` wraps with `%w` correctly where a caller would want `errors.Is`. `ParseResponseCode`'s manual digit-to-int arithmetic (`int(code[0]-'0')*100 + ...`) is not a correctness issue — it runs only after the digit-only loop has validated all 7 characters, so it cannot misbehave — but `strconv.Atoi(code[:3])` would be the more idiomatic, self-evidently-correct choice; noted as a minor style observation, not scored down further. |

**Weighted total** = (2×9 + 9 + 9 + 7 + 9) / 6 = (18 + 9 + 9 + 7 + 9) / 6 =
**52 / 6 = 8.67/10**

## Verdict: FAIL

The weighted average (8.67/10) clears the harness's general 7.0 threshold,
but this project's own `gan-harness/eval-rubric.md` pass bar is stricter:
**all dimensions must be ≥8, none below 6.** Test quality scores 7/10,
failing that per-dimension floor by one point. Every other dimension is at
or above 8. This is a narrow, well-scoped FAIL: the gap is confirmed to be
approximately 6 lines of assertion changes in one test file
(`header_test.go`), with zero required changes to any production code
(`header.go`, `profile.go`, `responsecode.go` all pass mutation testing on
every checked path except this one test-assertion class).

## Critical Issues (must fix)

1. **`header_test.go`'s "absent when not supplied" assertions are
   non-discriminating for 5 of 6 optional headers** (`ORIGIN`,
   `X-IP-ADDRESS`, `X-DEVICE-ID`, `X-LATITUDE`, `X-LONGITUDE`) →
   Replace `h.Get(k) != ""` with a map-presence check (`_, ok := h[k]; ok`
   using the **canonical** header key form, e.g. `h["X-Latitude"]` not
   `h["X-LATITUDE"]`) or `len(h.Values(k)) != 0`. The **load-bearing** fix
   sites, where the mutation actually proved the assertion is currently
   blind, are `header_test.go:58` (`Origin` check in
   `TestHeaderBuilder_Build_B2B`, live regardless of `B2B2C`) and
   `header_test.go:178` (`TestHeaderBuilder_Build_B2B2C_
   OptionalFieldsOmittedWhenEmpty`, where `B2B2C: true` and the per-field
   guards are actually exercised). `header_test.go:64` (the B2B2C-only-
   headers-absent loop inside `TestHeaderBuilder_Build_B2B`, where
   `B2B2C: false`) is not proven broken — that whole header block is
   skipped there regardless of the per-field guards — but tightening it the
   same way is cheap defense-in-depth and prevents a false sense of
   coverage if that test is ever changed to `B2B2C: true`. A fix that
   touches only line 64 would **not** close this finding; lines 58 and 178
   are the ones that matter. No change needed in `header.go` — its guards
   are already correct.

## Major Issues (should fix)

None beyond the critical issue above — everything else that could plausibly
be a major issue (formula correctness, sentinel mapping, scope discipline,
crypto reuse) was mutation-tested or read-verified clean.

## Minor Issues (nice to fix)

1. `ParseResponseCode`'s manual `int(code[0]-'0')*100 + ...` arithmetic could
   be `strconv.Atoi(code[:3])` for marginally better readability — not a
   correctness issue since it only runs after digit validation, purely
   cosmetic.
2. `HeaderBuilder.ClientSecret` is a plaintext exported field with no
   `String()`/`GoString()` redaction, so `%+v`/`%#v` formatting of a
   `HeaderBuilder` value would print the secret. Spec-mandated shape (a
   struct field carrying the secret), not a defect — flagging for awareness
   only, not scored against.

## What Improved Since Last Iteration

- New Slice 2 surface (`Profile`, `DefaultProfile`, `HeaderBuilder`,
  `ParseResponseCode`, response-code sentinels, `ResponseCodeError`) is
  implemented with zero build/vet/gofmt/race issues on the first eval pass —
  no re-fix cycle needed to reach a clean build, unlike Slice 1 which took
  two fix commits (feedback-001 → feedback-002) to close Security and
  Idiomatic gaps.
- Slice 1 files (`signing.go`, `signing_test.go`) were left untouched by
  this slice's commit, confirmed via `git log -p`, preserving the
  already-passed Slice 1 review.
- `profile_test.go` and `responsecode_test.go` are properly table-driven and
  both survived targeted mutation testing (sentinel-mapping swap,
  field-order swap) without any false pass.
- The custom-`Profile` hook is proven to genuinely take effect (not
  accepted-and-ignored) via a real behavioral fork in the test, matching
  the spec's explicit ask to "prove... a custom Profile... changes the
  resulting X-TIMESTAMP header."

## What Regressed Since Last Iteration

None. This is a new-surface slice, not a fix to prior work, so there is no
prior-iteration behavior to regress from within Slice 2's own files, and
Slice 1's files/behavior are unchanged (verified via git history).

## Specific Suggestions for Next Iteration

1. Apply the exact fix under "Critical Issues" above — swap `h.Get(k) != ""`
   for a canonical-key map-presence check in the three locations named. This
   alone should be sufficient to clear the Test Quality floor; re-run the
   same four mutations (A–D) described above against the fixed test file to
   confirm before resubmitting.
2. Optional, not required to pass: table-ize `header_test.go`'s five test
   functions into one `[]struct{...}` + `t.Run` loop for consistency with
   `profile_test.go`/`responsecode_test.go`'s established style — the
   current per-scenario functions are readable and not a scored deduction,
   but a single table would make the eventual re-run of mutation checks
   easier to reason about.
3. Do not touch `header.go`, `profile.go`, or `responsecode.go` production
   logic for this fix — every mutation path in the actual implementation
   passed; only the test file's assertion strength needs to change.
