# Phase 31: Auth Payment, Payment Query

Source: `docs/research/2026-09-12-transfer-debit-portal-research.md` §5.3
(lines 235-266), Service Codes 63-64. First phase of the Auth Payment
sub-group — the last sub-group of Transfer Debit, split across three
phases (31: 63-64; 32: 65-68 Capture/Void lifecycle; 33: 69 Refund) per
advisor's phase-sizing plan.

## GET/POST method contradiction (research §6 item 1) — resolved: POST

Research records the Overview tab marking all 7 Auth Payment endpoints
GET, while every Code Snippet worked example across all 7 is POST-shaped
with a JSON request body. Resolved as POST, for two independent reasons:

1. Every worked example, request and response, is POST-shaped. No
   endpoint in the package has ever had a body-carrying GET.
2. Structurally: this package's signing path
   (`BuildStringToSignTransaction`, `header.go`) computes the
   string-to-sign from `HTTPMethod:EndpointUrl:AccessToken:HexSHA256(body):TimeStamp`
   (symmetric) — every existing calling function sets `hb.Body` to the
   marshaled request and relies on that body being signed and sent. A
   GET carrying a mandatory JSON body isn't representable by any
   existing calling-function shape in this package, and the package's
   sole method-override point, VA Get Report (Phase 15), exists
   precisely because that one endpoint deviates from POST — it does
   not carry a body when it does so. An unexplained "GET" marking with
   a full mandatory-body worked example, and no field table describing
   query parameters, is treated as a portal Overview-tab defect, not a
   real transport difference. No method override is used for any of
   the 7 Auth Payment endpoints.

## Naming

`AuthPayment` (63), `AuthPaymentQuery` (64). `Auth` prefix does not
collide with existing `Token*`/`Sign*` auth-token/signing code.

## Auth Payment (63)

Req fields per research: `PartnerReferenceNo string` M (no omitempty),
`MerchantID string` M (no omitempty), `SubMerchantID O`, `Amount
*Money` — note research records `amount` itself as not separately
marked Optional/Mandatory at the container level, unlike the
package-wide `Amount O`-container convention (research §6 item 7, "not
a new decision, flagged per-occurrence") — modeled as `*Money` (pointer,
`omitempty`) anyway, consistent with every other Amount-family field in
the package; `FeeType O` (String(25), OUR/BEN/SHA), `MCC O`
(String(32)), `ProductCode O` (String(64)), `Title string` M
(String(256), no omitempty), `Items O` — a list of purchased goods with
no full item-level schema given anywhere beyond the worked example's
`goodsId`/`price`/`category`/`unit`/`quantity` — modeled as
`json.RawMessage`, matching the package's established treatment of
genuinely untyped fields (`AdditionalInfo`; also `AccountBindingRequest.AdditionalData`,
`VerifyOTPResponse.QParams`, `CPMPaymentRequest.Items` — the last being
the exact same field name and shape precedent, Phase 28), `AdditionalInfo O`.

Resp: `ReferenceNo C` (success only, omitempty), `PartnerReferenceNo O`,
`Amount` (`value`/`currency` members Mandatory per research; the
container itself carries no M/O marker either way, same as the request
side) — modeled as plain, non-pointer `Money`, since this is the
success response for a hold that was just placed and an amount is
expected to always be present on it (see the santa-loop round 1 note
below — this is a narrower judgment call than Payment Query's
Optional-marked container, not "the container is Mandatory here" as an
earlier draft of this doc incorrectly claimed), `PaidTime string` M
(String(25), no omitempty), `AdditionalInfo O`.

Function: POST, no method override, path `auth/payment` (research §1
line 36, quoted verbatim — an earlier draft of this doc claimed the
path "is not spelled out verbatim in research" and invented
`debit/auth-payment`; that claim was false and is corrected here, see
santa-loop round 1 below). Not idempotent (an authorization-initiating
call) — non-idempotency doc-comment note included, keyed on
X-EXTERNAL-ID, matching every other initiating call in the package.

## Payment Query (64)

Req: `OriginalPartnerReferenceNo O`, `OriginalReferenceNo O`,
`MerchantID O`, `SubMerchantID O`, `ExternalStoreID O`,
`AdditionalInfo O` — no Mandatory field.

Resp: `OriginalPartnerReferenceNo O`, `OriginalReferenceNo O`, `Amount
O` (members M) — modeled as `*Money` (pointer, `omitempty`), per the
package's established rule that an explicitly Optional-marked
container uses `*Money` regardless of its members' own markers (research
§6 item 7; an earlier draft of this doc incorrectly modeled this as a
plain, Mandatory `Money`, claiming a container-level M marker research
does not give — see santa-loop round 1 below), `PaidTime string` M
(String(25), no omitempty), `LatestTransactionStatus string` M (no
omitempty), `TransactionStatusDesc O` (String(50)), `AdditionalInfo O`.

### Casing contradiction (research §6 item 2) — no code consequence

The worked response example key-cases `originalpartnerReferenceNo`
(lowercase p) against the field table's `originalPartnerReferenceNo`.
This field sits on the **response** side only for this contradiction
(the worked example in question is a response body) — Go's
`encoding/json` unmarshal is case-insensitive, so either wire casing
populates `OriginalPartnerReferenceNo` correctly on the read path,
which is the only path this field is used on here (it is
request-Optional and, per the response table, echoed back). The field
table's casing (`originalPartnerReferenceNo`) is used for the Go tag,
consistent with every other occurrence of this field name across the
whole package. Recorded per research's own instruction not to silently
resolve table-vs-example contradictions.

Function: POST, no method override, path `auth/query` (research §1
line 37, quoted verbatim — same correction as Auth Payment above).
Idempotent (a query), no non-idempotency note.

## FieldCounts guards

Both new types (four structs including responses) get `FieldCounts`
guard tests per the established convention, plus mandatory-field
zero-value presence tests and full wire-body map comparisons in
round-trip tests, per the conventions reinforced in Phases 26 and 30.

## go-review fixes (round 1)

Two issues found and fixed before santa-loop:

1. **HIGH — zero transport-level test coverage.** The initial test
   files only exercised struct marshal/unmarshal and never called
   `AuthPayment`/`AuthPaymentQuery` themselves, leaving `hb.Body`
   assignment, `checkResponseStatus` wiring, and the empty-`ResponseCode`
   guard entirely untested (0.0% coverage, the only two functions in the
   package at that level). Fixed by adding the standard `httptest`-backed
   suite every other endpoint file carries: `_ParsesResponse`,
   `_RequestBodyRoundTrips`, `_NonTwoXXResponseCodeIsError`,
   `_NonTwoXXStatusWithTwoXXBodyIsError`, `_TwoXXStatusWithNoResponseCodeIsError`,
   mirroring `transaction_status_inquiry_bank_test.go` exactly. This
   section itself is the fix for the doc gap that let it happen — Phases
   32/33 must include these five test shapes too, not just FieldCounts/
   round-trip/zero-value.
2. **MEDIUM — wrong response-code fixtures.** Test fixtures used
   `"2007300"`/`"2007400"` (Service Codes 73/74, both unrelated and
   already assigned elsewhere in research) instead of the correct
   `"2006300"`/`"2006400"` for Service Codes 63/64 (responseCode =
   HTTPStatus(3) + ServiceCode(2) + CaseCode(2), per `ParseResponseCode`).
   Fixed by correcting every fixture in both test files.

## santa-loop round 1 findings (both reviewers converged)

Two independent reviewers (`ecc:code-reviewer`, `ecc:go-reviewer`) each
verified every field against research §5.3/§1/§6 directly rather than
trusting this doc's prior draft, and both surfaced the same two
factual errors in this doc's own reasoning — a case of the design
doc's own citations being wrong, same class as prior phases' research-
citation slips, except here the doc's error was in its own inference,
not in quoting research:

1. **MEDIUM — invented endpoint paths.** This doc's first draft
   claimed `debit/auth-payment` / `debit/auth-payment-status` were
   used because "the exact path is not spelled out verbatim in
   research" and inferred them from a (nonexistent) `debit/`-prefix
   convention. Both false: research §1's path column gives `auth/payment`
   (63, line 36) and `auth/query` (64, line 37) verbatim, and the
   `debit/` prefix belongs to the Direct Debit and Direct Debit
   BI-FAST sub-groups specifically — Auth Payment's own sub-group
   prefix is `auth/`, CPM's is `qr/`. Fixed: both doc comments and all
   test-server endpoint paths now use the verbatim research paths.
2. **MEDIUM — `AuthPaymentQueryResponse.Amount` modeled Mandatory
   against an explicit Optional marker.** Research §5.3 marks Payment
   Query's response `amount O (members M)` — an explicit Optional
   container marker, the exact case research §6 item 7 already
   resolves as `*Money` regardless of member markers. This doc's first
   draft claimed "container M per research" for this field, which
   research does not state. Fixed: `AuthPaymentQueryResponse.Amount`
   is now `*Money` (pointer, `omitempty`), consistent with every other
   Optional-container response in the package
   (`TransactionStatusInquiryBankResponse.Amount`,
   `CPMQueryPaymentResponse.Amount`, etc.); the zero-value test was
   corrected to expect `amount` absent, not `{"value":"","currency":""}`.

Both reviewers separately confirmed `AuthPaymentResponse.Amount`
(Auth Payment 63's own response) is *not* the same case: research
gives that container no marker at all (neither O nor M), only its
members are marked M, identical to the request side's own unmarked
container — so the plain-`Money` choice there is a defensible judgment
call specific to a just-succeeded hold response, not a research
citation error. This doc's comments were corrected to stop claiming a
container-level M marker that research does not give, while keeping
the modeling choice itself (reviewers explicitly did not flag it as
wrong, only the justification as inaccurate).

Both reviewers independently confirmed the GET/POST resolution and the
casing-contradiction resolution as sound after re-verifying the
underlying reasoning themselves (not just re-reading this doc). Both
MEDIUM findings above are fixed; a fresh round 2 of both reviewers runs
on the corrected diff per the established santa-loop process.
