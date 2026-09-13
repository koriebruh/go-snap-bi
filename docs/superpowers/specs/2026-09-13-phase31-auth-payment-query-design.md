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
`Amount *Money` M (both `value`/`currency` members Mandatory per
research; modeled as plain, non-pointer `Money` since the container
itself is Mandatory here, unlike the request side — this is the first
Auth Payment field where the container itself carries an M marker, so
it gets the plain-struct treatment per the package's established
optional-container-vs-mandatory-container rule), `PaidTime string` M
(String(25), no omitempty), `AdditionalInfo O`.

Function: POST, no method override, path `debit/auth-payment` (per
services naming convention — the exact path is not spelled out
verbatim in research beyond the worked example's host, and is inferred
consistently with every other Transfer Debit endpoint's `debit/`-prefixed
paths). Not idempotent (an authorization-initiating call) —
non-idempotency doc-comment note included, keyed on X-EXTERNAL-ID,
matching every other initiating call in the package.

## Payment Query (64)

Req: `OriginalPartnerReferenceNo O`, `OriginalReferenceNo O`,
`MerchantID O`, `SubMerchantID O`, `ExternalStoreID O`,
`AdditionalInfo O` — no Mandatory field.

Resp: `OriginalPartnerReferenceNo O`, `OriginalReferenceNo O`, `Amount
*Money` M (container M per research, members M — plain `Money`, not
pointer, same reasoning as Auth Payment's response), `PaidTime string`
M (String(25), no omitempty), `LatestTransactionStatus string` M (no
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

Function: POST, no method override, path `debit/auth-payment-status`
(inferred, matching the `-status`/`-query`-suffix convention already
used by `DirectDebitPaymentStatus`, Phase 26). Idempotent (a query),
no non-idempotency note.

## FieldCounts guards

Both new types (four structs including responses) get `FieldCounts`
guard tests per the established convention, plus mandatory-field
zero-value presence tests and full wire-body map comparisons in
round-trip tests, per the conventions reinforced in Phases 26 and 30.
