# Phase 14: Virtual Account Transaction (VA Inquiry, VA Payment, VA Inquiry Status)

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.3
(lines 126-156), the "Virtual Account (12 endpoints)" sub-group. This
phase covers 3 of the 12: Service Codes 24 (VA Inquiry), 25 (VA
Payment), 26 (VA Inquiry Status) — the biller-side inquiry/pay/status
triad, distinct from the VA-record-management endpoints (27-31) done in
Phase 13.

## Naming note (read first)

The portal's own endpoint titles are "VA Inquiry" (24) and "Inquiry VA"
(30, done in Phase 13 as `InquiryVA`). These are two different,
non-interchangeable endpoints with a confusingly similar name in the
source material itself, not a naming mistake introduced here. This
phase names its new type `VAInquiryRequest`/`VAInquiryResponse` and
function `VAInquiry` — word order mirrors the portal's own title for
endpoint 24, deliberately kept distinct from Phase 13's `InquiryVA`
(endpoint 30) by that same word order.

## Endpoints

All three are HTTP POST, no path parameters (research §2/§5.3 gives no
method override for 24/25/26, unlike the documented PUT/DELETE/GET
cases at 28/29/31/35).

- **24 VA Inquiry** — `.../{version}/transfer-va/inquiry` — `VAInquiry(ctx, t, hb, req)`
- **25 VA Payment** — `.../{version}/transfer-va/payment` — `VAPayment(ctx, t, hb, req)`
- **26 VA Inquiry Status** — `.../{version}/transfer-va/inquiry-status` — `VAInquiryStatus(ctx, t, hb, req)`

(Path suffixes follow the portal's slug convention already used for
27-31 — `create-va`, `update-va`, `update-status`, `inquiry-va`,
`delete-va` — applied to these three endpoint titles.)

## Ambiguous-type decisions (this phase)

Per the package's ambiguous-type rule ("can the documented type have a
non-string JSON representation? If yes → `json.RawMessage`") and the
`BillDetail.BillReferenceNo` precedent (Phase 13 santa-loop round 2: a
documented Numeric field shown quoted in one place still gets
`json.RawMessage`, because one worked example is thin evidence about
what every issuer sends):

| Field | Guides label | Worked example (§5.3) | Go type |
|---|---|---|---|
| `customerNo` | String(20) | quoted in 24/25's own examples; bare number (20 digits) in 26's | `json.RawMessage` |
| `channelCode` | Number | bare number `6011`, consistently | `json.RawMessage` |
| `paymentType` | String(1) | bare number `1` in every worked example that includes it | `json.RawMessage` |
| `paidBills` | String(6) | hex bitmask, no non-string example noted | `string` |
| `referenceNo` | String | quoted in every worked example belonging to endpoints 24/25/26 (the one bare-number occurrence noted in §5.3 line 137 is endpoint 33's own request, out of scope this phase) | `string` |

`customerNo` is decided once for this phase and applied identically to
all three new request/response types (`VAInquiryRequest`/`Data`,
`VAPaymentRequest`/`Data`, `VAInquiryStatusRequest`/`Data`), since all
three share the identity triple and 26's own worked example is the
bare-number case.

A caller setting `CustomerNo`, `ChannelCode`, or `PaymentType` must
supply a complete JSON value (e.g. `json.RawMessage("12345")` or
`json.RawMessage(`"12345"`)`), not a bare Go string — same rule as
`BillReferenceNo` (Phase 13).

A mandatory `json.RawMessage` field (`CustomerNo`, `PartnerServiceID`,
`VirtualAccountNo` are the identity triple, all Mandatory per §5.3 line
128) carries no `omitempty` tag, matching the package's "mandatory
fields always serialize" convention — but note `json.RawMessage`'s zero
value marshals as JSON `null`, not `""`. This differs from a mandatory
`string` field's `""` and is called out explicitly in each new file's
mandatory-fields test.

## Open contradiction not resolved here (research §6 item 1)

VA Inquiry Status (26)'s own field table says `inquiryRequestId` is
Conditional with the portal note "if not sent, returns array based on
virtualAccountNo," but the response is documented as a single Object,
not an array, everywhere else in §5.3. This phase models the response
as the documented single Object (matching the response-shape
description "mirrors VA Payment's response shape"), per the field
table being the stronger source. The array-reading portal sentence is
recorded here and left unresolved, per the established
research-contradiction handling (VA Get Report's GET/POST split,
Bulk Cashin's `bulkid` casing — deferred, not guessed at).

## Types

### VAInquiry (24)

`VAInquiryRequest`: identity triple (`PartnerServiceID string` M,
`CustomerNo json.RawMessage` M, `VirtualAccountNo string` M) +
`TrxDateInit string` O (Date, string per package convention —
`ExpiredDate`/`LastUpdateDate` precedent) + `ChannelCode json.RawMessage`
O + `Language string` O + `HashedSourceAccountNo string` C (omitempty,
Conditional treated as Optional for serialization purposes, matching
Phase 12's `hashedSourceAccountNo`/`sourceBankCode` handling) +
`SourceBankCode string` C + `PassApp string` O + `InquiryRequestID
string` M (no omitempty) + `AdditionalInfo json.RawMessage` O (§3:
present on every request/response in the group).

`VAInquiryData` (the `virtualAccountData` object — §5.3 line 130-131
confirms 24 uses capital-D `virtualAccountData`): identity triple +
`InquiryStatus string` + `InquiryReason *LocalizedText` (Optional
nested object → pointer, per the `omitempty`-is-a-no-op-on-structs
rule) + `VirtualAccountName/Email/Phone string` + `InquiryRequestID
string` + `TotalAmount *Money` + `SubCompany string` + `BillDetails
[]BillDetail` + `FreeTexts []LocalizedText` + `VirtualAccountTrxType
string` + `FeeAmount *Money`. All fields besides the identity triple
are unmarked for M/O in §5.3's response summary; treated as Optional
(omitempty), matching how Phase 11-13 handled response fields with no
explicit cardinality letter.

`VAInquiryResponse`: `ResponseCode string`, `ResponseMessage string`,
`VirtualAccountData *VAInquiryData` (`omitempty`).

### VAPayment (25)

`VAPaymentRequest`: identity triple + `TrxID string` C (omitempty;
"mandatory if from Create VA" per §5.3 — a data-dependent condition
this package cannot express in the type system, so it's typed as
Optional/omitempty and the condition is documented in the field
comment, matching how other Conditional fields are handled) +
`PaymentRequestID string` M (no omitempty) + `ChannelCode
json.RawMessage` O + `HashedSourceAccountNo string` C + `SourceBankCode
string` C + `PaidAmount Money` M (Mandatory nested object → plain
struct, not pointer, per the established rule) + `CumulativePaymentAmount
*Money` O + `PaidBills string` O (hex bitmask, e.g. `"3F"`) +
`TotalAmount *Money` O + `TrxDateTime string` O + `ReferenceNo string`
O + `JournalNum string` O + `PaymentType json.RawMessage` O +
`FlagAdvise string` O + `SubCompany string` O + `BillDetails
[]BillDetail` O + `FreeTexts []LocalizedText` O + `AdditionalInfo
json.RawMessage` O.

`VAPaymentData`: §5.3 line 145 says "Resp adds paymentFlagReason{...},
paymentFlagStatus, per-bill status/reason" — read together with the
"beyond identity triple" framing used throughout §5.3, this response
mirrors the full request field set (Update VA's response is described
the same way relative to its own request) plus the two named
additions. Per-bill `status`/`reason` need no new field — `BillDetail`
already carries `Status string` and `Reason *LocalizedText` (Phase 13).
So `VAPaymentData` = identity triple + every `VAPaymentRequest` field
(same names/types) + `PaymentFlagReason *LocalizedText` +
`PaymentFlagStatus string`.

`VAPaymentResponse`: `ResponseCode string`, `ResponseMessage string`,
`VirtualAccountData *VAPaymentData` (`omitempty`).

### VAInquiryStatus (26)

`VAInquiryStatusRequest`: identity triple + `InquiryRequestID string` C
(omitempty) + `PaymentRequestID string` O (omitempty).

`VAInquiryStatusData`: §5.3 line 146 — "mirrors VA Payment's response
shape plus `transactionDate Date O`." Per the "distinct types per
service code even for identical/near-identical shapes" convention
(Phase 12 RTGS/SKNBI, Phase 13 VA data types), this is its own named
type, not a reuse of `VAPaymentData`: every `VAPaymentData` field plus
`TransactionDate string` (Date → string, same convention as
`ExpiredDate`).

`VAInquiryStatusResponse`: `ResponseCode string`, `ResponseMessage
string`, `VirtualAccountData *VAInquiryStatusData` (`omitempty`).

## Drift guard

`VAInquiryStatusData` is not byte-identical to `VAPaymentData` (it has
one extra field), so the Phase 13 pattern of a single
`reflect.DeepEqual` over the whole struct doesn't apply directly.
Instead, `transfer_shared_types_test.go` gets a new test that iterates
`VAPaymentData`'s fields via reflection and asserts each one exists on
`VAInquiryStatusData` with an identical Go type and struct tag, plus
asserts `VAInquiryStatusData.TransactionDate` exists as `string` with
tag `json:"transactionDate,omitempty"`. This catches the same
class of drift (a hand-edit to one type's field/tag not mirrored on
the other) without requiring full identity.

## Function behavior

All three functions follow the exact pattern already established for
`InquiryVA`/`CreateVA` (Phase 13): marshal `req`, set `hb.Body`, call
`t.Do`, `checkResponseStatus`, unmarshal into the `Response` type,
error if `ResponseCode == ""`. None of the three sets `hb.Method` — the
package's default (`POST`, whatever the caller configured via
`testHeaderBuilder`/production `HeaderBuilder`) applies, since no
method override is documented for 24/25/26 (unlike the PUT/DELETE
cases in Phase 13).

## Known limitation carried over (not re-litigated)

`responseCode` bare-number wire shape (Phase 13's known-limitation
finding) applies package-wide, already documented in Phase 13's design
doc against `InquiryVAResponse`. §5.3's bare-number `responseCode`
examples are specifically Inquiry VA (30) and Get Report (35) — neither
is in this phase's scope, so no new pinning test is added here; the
existing package-wide gap is unchanged by this phase's own worked
examples.
