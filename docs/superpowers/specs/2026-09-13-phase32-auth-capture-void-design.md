# Phase 32: Auth Capture, Capture Query, Void, Void Query

Source: `docs/research/2026-09-12-transfer-debit-portal-research.md` §5.3
(lines 267-301), Service Codes 65-68. Second of three phases for the
Auth Payment sub-group (31: 63-64 done; 32: this phase; 33: 69
Refund). Capture and Void share the `lastCapture`/`voidRemainingAmount`
String(8)-with-`"TRUE"` pattern and the `latestCaptureStatus`/
`latestVoidStatus` 3-value enum, so both mutating operations and both
`*Query` companions are done together per advisor's phase-sizing plan.

GET/POST resolved as POST for all four, same reasoning as Phase 31
(research §6 item 1 — every worked example POST-shaped, package's
signing path has no representation for a body-carrying GET). Paths
taken verbatim from research §1: `auth/capture` (65),
`auth/capture-query` (66), `auth/void` (67), `auth/void-query` (68) —
Phase 31's santa-loop caught an invented-path error, so this phase
quotes §1 directly rather than re-deriving.

## Naming

`AuthCapture` (65), `AuthCaptureQuery` (66), `AuthVoid` (67),
`AuthVoidQuery` (68), per advisor's original naming note.

## `lastCapture`/`voidRemainingAmount` (research §6 item 6) — plain string

Both documented `String(8)` but carry worked-example value `"TRUE"` — a
boolean-shaped value in a string field. Research itself states the
wire shape (quoted string) is unambiguous here, unlike Transfer
Kredit's genuinely ambiguous number-vs-string cases — so both are
modeled as plain `string`, not `json.RawMessage`, per advisor's
explicit guidance during phase planning.

## Amount-family container modeling

Per-field, following the container-marker rule established in Phase
31 (Optional container → `*Money`; Mandatory or explicitly-unmarked
container in a success-only response → plain `Money`, a per-occurrence
judgment call, not a blanket rule):

- `AuthCaptureRequest.CaptureAmount`: research marks `captureAmount O
  (members M)` — explicit Optional container → `*Money`.
- `AuthCaptureResponse.CaptureAmount`: research's own parenthetical
  reads "captureAmount M (members M, container itself unmarked —
  recorded as shown)" — the same unmarked-container-but-M-members
  ambiguity as Phase 31's `AuthPaymentResponse.Amount`, in a
  success-response field → plain `Money`, matching that precedent.
- `AuthCaptureQueryResponse.CaptureAmount`: research states plainly
  `captureAmount M (members M)` with no unmarked-container caveat — an
  explicit Mandatory container → plain `Money`.
- `AuthVoidRequest.VoidAmount`: research marks `voidAmount O (members
  M)` — explicit Optional container → `*Money`.
- `AuthVoidResponse.VoidAmount` and `AuthVoidQueryResponse.VoidAmount`:
  research states plainly `voidAmount M (members M)` for both, no
  unmarked-container caveat — explicit Mandatory container → plain
  `Money` for both.

## Auth Capture (65)

Req (9 fields): `OriginalReferenceNo string` M (no omitempty),
`OriginalPartnerReferenceNo string` M (no omitempty), `MerchantID
string` M (no omitempty), `SubMerchantID O`, `PartnerCaptureNo string`
M (String(64), no omitempty), `CaptureAmount *Money O`, `Title string`
M (String(256), no omitempty), `LastCapture O` (String(8), plain
string), `AdditionalInfo O`.

Resp (9 fields incl. envelope): `OriginalReferenceNo O`,
`OriginalPartnerReferenceNo O`, `PartnerCaptureNo O`, `CaptureNo C`
(String(64), omitempty), `CaptureAmount Money` (no omitempty, per
above), `CaptureTime C` (String(25), omitempty), `AdditionalInfo O`.

Function: POST, no method override, path `auth/capture`. Not
idempotent (a capture-initiating call) — non-idempotency doc-comment
note included, keyed on X-EXTERNAL-ID, matching every other initiating
call in the package.

## Auth Capture Query (66)

Req (7 fields): `OriginalReferenceNo string` M (no omitempty),
`OriginalPartnerReferenceNo O`, `MerchantID string` M (no omitempty),
`SubMerchantID O`, `CaptureNo O` (String(64)), `PartnerCaptureNo
string` M (String(64), no omitempty), `AdditionalInfo O`.

Resp (10 fields incl. envelope): `OriginalReferenceNo O`,
`OriginalPartnerReferenceNo O`, `CaptureNo O`, `CaptureAmount Money`
M (no omitempty, per above), `CaptureTime C` (omitempty),
`LatestCaptureStatus C` (String(32), omitempty — the 3-value enum),
`PartnerCaptureNo string` M (no omitempty — Mandatory here, unlike the
request's own `CaptureNo`), `AdditionalInfo O`.

Function: POST, no method override, path `auth/capture-query`.
Idempotent (a query), no non-idempotency note.

## Auth Void (67)

Req (9 fields): `OriginalReferenceNo string` M (no omitempty),
`OriginalPartnerReferenceNo string` M (no omitempty), `MerchantID
string` M (no omitempty), `SubMerchantID O`, `VoidAmount *Money O`,
`PartnerVoidNo string` M (String(64), no omitempty),
`VoidRemainingAmount O` (String(8), plain string), `Reason O`
(String(256)), `AdditionalInfo O`.

Resp (9 fields incl. envelope): `OriginalReferenceNo O`,
`OriginalPartnerReferenceNo O`, `VoidNo C` (String(64), omitempty),
`PartnerVoidNo string` M (no omitempty — Mandatory in the response,
unlike Capture's own `PartnerCaptureNo O` in its response), `VoidAmount
Money` M (no omitempty, per above), `VoidTime C` (String(25),
omitempty), `AdditionalInfo O`.

Function: POST, no method override, path `auth/void`. Not idempotent
(a void-initiating call) — non-idempotency doc-comment note included,
keyed on X-EXTERNAL-ID.

## Auth Void Query (68)

Req (7 fields): `OriginalReferenceNo string` M (no omitempty),
`OriginalPartnerReferenceNo O`, `MerchantID string` M (no omitempty),
`SubMerchantID O`, `VoidNo O` (String(64)), `PartnerVoidNo string` M
(String(64), no omitempty), `AdditionalInfo O`.

Resp (10 fields incl. envelope): `OriginalReferenceNo O`,
`OriginalPartnerReferenceNo O`, `VoidNo O`, `VoidAmount Money` M (no
omitempty, per above), `VoidTime C` (omitempty), `LatestVoidStatus C`
(String(32), omitempty — the 3-value enum), `PartnerVoidNo O`
(omitempty — Optional here, unlike Capture Query's own
`PartnerCaptureNo` Mandatory), `AdditionalInfo O`.

Function: POST, no method override, path `auth/void-query`. Idempotent
(a query), no non-idempotency note.

## Test coverage (applying Phase 31's round-1 lesson upfront)

Every one of the four endpoint files gets, from the start: `FieldCounts`
guards, round-trip tests populating every field including
`AdditionalInfo`, mandatory-field zero-value presence tests, a
malformed-`AdditionalInfo`-is-marshal-error test, AND the full
`httptest`-backed transport suite (`_ParsesResponse`,
`_RequestBodyRoundTrips`, `_NonTwoXXResponseCodeIsError`,
`_NonTwoXXStatusWithTwoXXBodyIsError`,
`_TwoXXStatusWithNoResponseCodeIsError`) — Phase 31's go-review found
the calling functions themselves untested when only the struct-level
tests were written; this phase does not repeat that gap.
