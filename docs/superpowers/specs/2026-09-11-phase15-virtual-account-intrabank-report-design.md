# Phase 15: Virtual Account Intrabank + Get Report

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.3
(lines 126-156). This phase covers the last 4 of the Virtual Account
sub-group's 12 endpoints: Service Codes 32 (VA Inquiry Payment
Intrabank), 33 (VA Payment Intrabank), 34 (VA Notify Payment
Intrabank), 35 (VA Get Report). This completes the Virtual Account
sub-group (Phases 13-15).

## Endpoint 34's direction (read first — this changes the file shape)

§5.3 line 154 says endpoint 34 is a "Callback the PJP sends outward."
This is the opposite direction from the RTGS/SKNBI/Interbank-Bulk
notification endpoints (Phase 12), which are inbound callbacks the PJP
*receives* and which are therefore modeled as plain struct pairs with
no calling function (the PJP implements a handler; this package has no
role in initiating them). Endpoint 34 is the PJP *initiating* an
outbound HTTP call, which is architecturally identical to every other
endpoint in this package — so it gets a full `VANotifyPaymentIntrabank`
calling function, not the struct-only treatment.

## `virtualAccountdata` — lowercase d (read first — highest-risk item)

§5.3 line 132 confirms, as "not a table typo," that all four endpoints
in this phase use the response envelope key `virtualAccountdata`
(lowercase d), unlike every VA type from Phases 13-14, which use
`virtualAccountData` (capital D). A copy-paste of an existing
`*Response` struct tag silently produces a field that never
populates — `encoding/json` unmarshals unknown keys into nothing, no
error. This package's `ParsesResponse` tests are hand-built to mirror
the struct (as already flagged for `AdditionalInfo` in Phase 14's
santa-loop round 1), so they cannot catch this by themselves; a
dedicated test asserting the literal struct tag string is added (see
"Drift guards" below) in addition to each endpoint's fixture using the
real lowercase-d key.

## Known package-level inconsistency (not fixed here)

Endpoints 32, 33, 34 all show `customerNo` as a bare JSON number in
their own worked examples (§5.3 line 135), so per the established
ambiguous-type rule `CustomerNo` is `json.RawMessage` on all three new
request/response types in this phase — same as Phase 14. This widens
an inconsistency already accepted in Phase 14's santa-loop round 1:
Phase 13's VA-record-management endpoints (27-31) type the same
`customerNo` field as plain `string`, since none of *their* worked
examples show a bare number. The package now has 8 VA endpoints typing
`customerNo` as `string` (27-31) and 6 typing it `json.RawMessage`
(24-26, 32-34), all for the same conceptual field, split by which
specific endpoint's own worked example was checked. This is stated
here as a known, accepted, package-level inconsistency — not something
this phase resolves. A future phase could unify by making all VA
`customerNo` fields `json.RawMessage` package-wide, but that is an
unscoped cross-phase change and out of scope here.

## Ambiguous-type decisions (this phase)

| Field | Endpoint(s) | Guides label | Worked example (§5.3) | Go type |
|---|---|---|---|---|
| `customerNo` | 32, 33, 34 | String(20) | bare number, per line 135 | `json.RawMessage` |
| `referenceNo` | 33 (request only) | String | bare number `123456789012345` in 33's own request (line 137); quoted in 33's own response | `json.RawMessage` (both request and response, for type consistency within the endpoint — the request wire shape is confirmed ambiguous, and a `json.RawMessage` field accepts a quoted-string response value without issue) |
| `partnerServiceId` | 35 only | flips to `Number` for 35 specifically (line 155), `String` everywhere else in the package | not shown as bare number in a worked example, but the Guides label itself now permits a non-string representation | `json.RawMessage` — first endpoint in the package where `partnerServiceId` is not `string` |
| `partnerServiceId` (typo note) | 34 | documented `"StringNumber"` (literal typo in the source table, line 140) | quoted string in 34's own worked example | `string` — the worked example resolves the typo; no ambiguity in practice |

## Endpoint 35: GET vs POST (research §6 item 1, deferred)

§5.3 line 155 documents endpoint 35 both ways: "GET per doc, POST+body
per code snippet." This package models it as POST with a JSON body,
matching the architecture used by every other endpoint (`hb.Body` set
from the marshaled request, no path parameters anywhere in this
group). The GET reading is recorded here and left unresolved, per the
same handling already applied to this exact contradiction in Phases
13-14 — not guessed at.

## Endpoint 35: array-shaped response

§5.3 line 155: "`virtualAccountdata` is an Array of Objects (only VA
endpoint whose top-level data is an array), each item shaped like the
Payment/Inquiry-Status response object." So
`GetReportResponse.VirtualAccountData` is `[]GetReportData`, not a
pointer to a single object — the only VA response in the package typed
this way. `GetReportData` is its own named type (per the "distinct
types per service code even for identical shapes" convention), field-
identical to `VAInquiryStatusData`, guarded by a full-equality
reflection test (same pattern as Phase 13's three-way VA-data guard,
since the shapes are claimed fully identical, not "plus one field"
like `VAInquiryStatusData` vs `VAPaymentData`).

## Known limitation carried over: bare-number `responseCode` on Get Report

§5.3 line 141 documents endpoint 35's own worked response as
`"responseCode":2003500,` (bare number) — the second of only two such
examples in the whole research doc (the other is endpoint 30, already
documented as a package-wide known limitation in Phase 13's design
doc). Per the lesson from Phase 13's santa-loop round 2/3 (a "known
limitation" write-up must be independently verified per new type, not
assumed from the existing citation), this phase adds its own pinning
test for `GetReportResponse.ResponseCode` specifically — it is a type
this phase introduces, so Phase 13's verification of `InquiryVAResponse`
does not automatically cover it.

## Types

### VAInquiryPaymentIntrabank (32)

§5.3 line 152: "adds `partnerReferenceNo String(128) O`,
`sourceAccountNo/Type O` ("D"/"S"). Resp adds `productName String(30)
O`, `billAmountLabel/Value`." This row does not restate `billDetails`,
`freeTexts`, `totalAmount`, or `feeAmount` the way endpoint 24's row
does, so — per the discipline of recording only confirmed fields —
this phase does not add them to 32's types; only the fields the row
actually names are modeled, beyond the identity triple.

`VAInquiryPaymentIntrabankRequest`: identity triple (`PartnerServiceID
string` M, `CustomerNo json.RawMessage` M, `VirtualAccountNo string` M)
+ `PartnerReferenceNo string` O + `SourceAccountNo string` O +
`SourceAccountType string` O (enum "D"=Debit/"S"=Savings, kept as a
plain string per the package's enum-as-string convention) +
`AdditionalInfo json.RawMessage` O (§3: present on every
request/response in the group).

`VAInquiryPaymentIntrabankData` (`virtualAccountdata`, lowercase d):
identity triple + `PartnerReferenceNo string` + `SourceAccountNo
string` + `SourceAccountType string` (echoed, matching the
request-mirrors-into-response convention used since Phase 13) +
`ProductName string` + `BillAmountLabel string` + `BillAmountValue
string` + `AdditionalInfo json.RawMessage`.

`VAInquiryPaymentIntrabankResponse`: `ResponseCode string`,
`ResponseMessage string`, `VirtualAccountData
*VAInquiryPaymentIntrabankData` `json:"virtualAccountdata,omitempty"`.

### VAPaymentIntrabank (33)

§5.3 line 153: "adds `sourceAccountNo/Type`, `inquiryRequestId O`,
`partnerReferenceNo M`, `paidAmount M`, `cumulativePaymentAmount O`,
`paidBills`, `paymentStatus String(20) O` (free-text status, not the
2-digit enum)." Plus `referenceNo` per line 137 (see ambiguous-type
table above).

`VAPaymentIntrabankRequest`: identity triple + `SourceAccountNo
string` O + `SourceAccountType string` O + `InquiryRequestID string` O
+ `PartnerReferenceNo string` M (no omitempty) + `PaidAmount Money` M
(Mandatory nested object → plain struct, per the established rule) +
`CumulativePaymentAmount *Money` O + `PaidBills string` O (hex
bitmask, same convention as Phase 14) + `PaymentStatus string` O
(free-text, distinct from the package's `transactionStatus` 2-digit
enum — explicitly called out by the research doc) + `ReferenceNo
json.RawMessage` O + `AdditionalInfo json.RawMessage` O.

`VAPaymentIntrabankData` (`virtualAccountdata`): identity triple +
every `VAPaymentIntrabankRequest` field, mirrored (same convention as
`VAPaymentData` mirroring `VAPaymentRequest` in Phase 14) —
`SourceAccountNo`, `SourceAccountType`, `InquiryRequestID`,
`PartnerReferenceNo`, `PaidAmount *Money` (Optional here, unlike the
Mandatory plain-struct request field), `CumulativePaymentAmount`,
`PaidBills`, `PaymentStatus`, `ReferenceNo`, `AdditionalInfo`.

`VAPaymentIntrabankResponse`: `ResponseCode string`, `ResponseMessage
string`, `VirtualAccountData *VAPaymentIntrabankData`
`json:"virtualAccountdata,omitempty"`.

### VANotifyPaymentIntrabank (34)

§5.3 line 154: `inquiryRequestId/paymentRequestId/partnerReferenceNo
O`, `trxDateTime O`, `paymentStatus O`, `paymentFlagReason
{LocalizedText}`. "Response mirrors most request fields plus
envelope." `PartnerServiceID`'s documented-typo note (line 140) is
resolved to `string` per its own worked example (see ambiguous-type
table above).

`VANotifyPaymentIntrabankRequest`: identity triple + `InquiryRequestID
string` O + `PaymentRequestID string` O + `PartnerReferenceNo string`
O + `TrxDateTime string` O + `PaymentStatus string` O +
`PaymentFlagReason *LocalizedText` O (Optional nested object →
pointer) + `AdditionalInfo json.RawMessage` O.

`VANotifyPaymentIntrabankData` (`virtualAccountdata`): identity triple
+ every `VANotifyPaymentIntrabankRequest` field beyond the triple,
mirrored (research says "mirrors most request fields" — this phase
mirrors all of them, since none are called out as excluded, matching
how "mirrors X's shape" was read literally in Phase 14).

`VANotifyPaymentIntrabankResponse`: `ResponseCode string`,
`ResponseMessage string`, `VirtualAccountData
*VANotifyPaymentIntrabankData` `json:"virtualAccountdata,omitempty"`.

`VANotifyPaymentIntrabank(ctx, t, hb, req)`: a normal calling function,
same shape as every VA-transaction function in Phase 14 — POST, no
method override, marshal → `hb.Body` → `t.Do` →
`checkResponseStatus` → unmarshal → empty-`responseCode` guard.

### VAGetReport (35)

§5.3 line 155. No identity triple in the request — only
`PartnerServiceID`, plus date/time range filters.

`GetReportRequest`: `PartnerServiceID json.RawMessage` M (documented
Number for this endpoint only — see ambiguous-type table above) +
`StartDate string` O (`yyyy-MM-dd`) + `StartTime string` O (`HH:mm`,
server defaults to `00:00` if omitted — documented default, not
enforced by this package) + `EndDate string` O + `EndTime string` O
(server defaults to `23:59`) + `AdditionalInfo json.RawMessage` O.

`GetReportData`: field-identical to `VAInquiryStatusData` (per "each
item shaped like the Payment/Inquiry-Status response object") — same
fields, same Go types, same JSON tags. A full-equality reflection
test guards the two against drift (see below).

`GetReportResponse`: `ResponseCode string`, `ResponseMessage string`,
`VirtualAccountData []GetReportData`
`json:"virtualAccountdata,omitempty"` — the only array-typed VA
response in the package.

`VAGetReport(ctx, t, hb, req)`: same calling pattern as every other
function in this phase.

## Drift guards

1. **Lowercase-d tag guard** (new): a reflection-based test asserting
   the literal struct tag on the `VirtualAccountData` field of each of
   the four new `*Response` types is exactly
   `json:"virtualAccountdata,omitempty"` (not `virtualAccountData`).
   This is the test that would have caught a copy-paste of the
   capital-D tag from any Phase 13/14 type.
2. **`GetReportData` ≡ `VAInquiryStatusData`** (new): a full-equality
   reflection test (field name, Go type, tag, and order), same pattern
   as Phase 13's three-way VA-data guard, since the two types are
   claimed fully identical rather than "plus one field."

## Known limitation: `GetReportResponse.ResponseCode` bare-number wire shape

Same package-wide gap as Phase 13's `InquiryVAResponse` (§5.3 line
141): `responseCode` is decoded twice per call — once by the shared
transport layer into an internal `string` field, and again by each
endpoint's own `Response.ResponseCode string` field. A server sending
a bare-number `responseCode` for Get Report fails at the transport
layer's decode first, and would also fail at `GetReportResponse`'s own
decode if the transport layer alone were fixed, for the same reason
verified empirically for `InquiryVAResponse` in Phase 13 (a `string`
field cannot receive a bare JSON number). This is confirmed here by
the same reasoning, not re-derived from scratch, and pinned with a
test analogous to Phase 13's `TestInquiryVA_BareNumberResponseCodeIsAKnownLimitation`.
A complete fix remains a package-wide architectural change (retyping
`ResponseCode` on every `Response` type plus the transport layer),
out of scope for this phase.
