# Phase 33: Auth Refund

Source: `docs/research/2026-09-12-transfer-debit-portal-research.md` §5.3
(lines 303-315), Service Code 69. Third and final phase of the Auth
Payment sub-group (31: 63-64; 32: 65-68; 33: this phase) — **completes
Transfer Debit** (all 21 endpoints across all 4 sub-groups).

GET/POST resolved as POST, same reasoning as Phases 31-32 (research §6
item 1). Path taken verbatim from research §1 line 18: `auth/refund`.

## Naming

`AuthRefund` (69).

## §6 item 3 — opposite Conditional triggers, recorded as shown

`OriginalCaptureNo` is Conditional, "must be filled upon unsuccessful
transaction" — the opposite condition-sense from most Conditional
fields in the package (typically filled on success); this is §5.3's
own field-table note for the request side. `OriginalReferenceNo` in
the response is Conditional the ordinary way, "must be filled upon
successful transaction" — so the response's two Conditional identifier
fields read as having genuinely opposite success/failure triggers from
each other, not a copy-paste duplicate of the same note. Research §6
item 3 flags this specific response-side pairing as not yet
independently verified ("needs verification that isn't just a
documentation slip"), listed under "Open Contradictions to Resolve
Per-Phase (not resolved in this document)" — so this is recorded as
research's own open observation, not a settled fact. Either way there
is no code consequence beyond a
doc-comment note — both fields are Conditional → `omitempty`
regardless of which direction their trigger runs.

## RefundAmount container

Research marks `refundAmount O (members M)` on both the request and
response side — an explicit Optional container in both places, unlike
Phases 31/32 where request- and response-side containers sometimes
differed. Modeled as `*Money` (pointer, `omitempty`) in both
`AuthRefundRequest` and `AuthRefundResponse`, per the package's
established Optional-container rule.

## Auth Refund (69)

Req (10 fields): `OriginalPartnerReferenceNo string` M (no omitempty),
`OriginalReferenceNo O`, `PartnerRefundNo string` M (String(64), no
omitempty), `MerchantID O`, `SubMerchantID O`, `OriginalCaptureNo C`
(String(64), omitempty — "must be filled upon unsuccessful
transaction"), `RefundAmount *Money O`, `ExternalStoreID O`, `Reason O`
(String(256)), `AdditionalInfo O`.

Resp (10 fields incl. envelope): `OriginalCaptureNo C` (omitempty, same
unsuccessful-transaction condition as the request field), `OriginalReferenceNo
C` (omitempty — "must be filled upon successful transaction", the
opposite condition from `OriginalCaptureNo`), `OriginalPartnerReferenceNo
O`, `PartnerRefundNo O` (Optional in the response, unlike Direct Debit's
own Refund response where `PartnerRefundNo` is Mandatory, Phase 27 —
recorded per-occurrence, not harmonized), `RefundNo string` M
(String(64), no omitempty), `RefundAmount *Money O`, `RefundTime
string` M (String(25), no omitempty), `AdditionalInfo O`.

Function: POST, no method override, path `auth/refund`. Not idempotent
(a refund-initiating call) — non-idempotency doc-comment note
included, keyed on X-EXTERNAL-ID, matching every other initiating call
in the package.

## Test coverage

Same full suite as Phases 31-32 from the start: `FieldCounts` guards,
round-trip tests populating every field, mandatory-field zero-value
presence test, malformed-`AdditionalInfo` test, and the full
`httptest`-backed transport suite (`_ParsesResponse`,
`_RequestBodyRoundTrips`, `_NonTwoXXResponseCodeIsError`,
`_NonTwoXXStatusWithTwoXXBodyIsError`,
`_TwoXXStatusWithNoResponseCodeIsError`).

## `doc.go` update

This phase's bullet marks Auth Payment sub-group complete and, with
it, all of Transfer Debit (4 sub-groups, 21 endpoints) complete. The
trailing summary paragraph — stale since Phase 26 per a Phase 30
santa-loop note — is refreshed to state Transfer Debit is done.

## santa-loop round 1 findings (both reviewers converged)

Both reviewers independently flagged the same LOW issue: the
`OriginalCaptureNo`/`OriginalReferenceNo` doc comments (in
`auth_refund.go` and this doc's §6 item 3 section above) cited "§6
item 3" for the request-side note (actually §5.3's own field-table
text) and, more substantively, stated the response-side
opposite-trigger distinction as a settled fact ("records this as a
genuine, non-copy-paste distinction") when research §6 item 3 itself
lists it under "Open Contradictions to Resolve Per-Phase (not resolved
in this document)" and explicitly says it "needs verification that
isn't just a documentation slip." Fixed: repointed the request-side
citation to §5.3, and reworded the response-side comment in both
`auth_refund.go` and this doc to present the distinction as research's
own open observation pending verification, not a resolved fact. Zero
code consequence either way — both fields remain Conditional →
`omitempty` regardless of which direction the trigger actually runs,
since this package validates wire shape only, never business rules.

Both reviewers separately confirmed: all field markers, the
`auth/refund` path, the `RefundAmount *Money` modeling (research marks
`refundAmount O` explicitly on both request and response), the
`PartnerRefundNo`-differs-from-Direct-Debit-Refund comparison, and the
21-endpoint Transfer Debit completion count. Round 2 not needed — the
one converged finding was comment-only, already fixed above.
