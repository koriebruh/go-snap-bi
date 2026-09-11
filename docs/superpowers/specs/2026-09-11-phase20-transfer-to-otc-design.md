# Phase 20: Transfer To OTC

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.8
(lines 182-188), Service Codes 44-46, all 3 endpoints in this
sub-group.

## Open contradiction not resolved here (research §6 item 5)

§5.8 line 188 documents endpoint 46 (Cancel Payment) with two
conflicting paths: the portal's Overview tab says `emoney/otc-cancel`,
but the Code Snippet tab's own request line reads `POST
.../v1.0/otc/cashout/cancel` — research explicitly flags this as
"unresolved, needs sandbox/Postman verification before hardcoding
either path." This phase's doc comment states `emoney/otc-cancel` (the
Overview reading, consistent with the sub-group's other two endpoints'
`emoney/otc-*` naming — `emoney/otc-cashout` for 44, `emoney/otc-status`
for 45) but records the Code Snippet's conflicting reading explicitly
in the same comment, per the established practice for unresolved path
contradictions (VA Get Report's GET/POST, Phase 15). This is
documentation only — `HeaderBuilder.EndpointURL` is always
caller-supplied, so no runtime behavior depends on which reading is
correct.

## Types

### TransferToOTCCreatePayment (44)

`TransferToOTCCreatePaymentRequest`: `PartnerReferenceNo string` M (no
omitempty) + `CustomerNumber string` M (no omitempty) + `OTP string` M
(no omitempty; String(8) per the Guides tab, an OTP code — no length
enforcement in Go, matching the package's general practice of not
validating field contents) + `Amount Money` M (Mandatory nested object
→ plain struct, per the established rule) + `FeeType string` O.

`TransferToOTCCreatePaymentResponse`: `ResponseCode string`,
`ResponseMessage string`, `ReferenceNo string` C (omitempty) +
`TransactionDate string` O.

### TransferToOTCTransferStatus (45)

§5.8 line 186: "originalX/serviceCode/status pattern, adds
`customerNumber M`, `amount M` on request." The base pattern is the
same originalX/serviceCode/status shape already used by
`TransactionStatusInquiryBank` (Phase 16) and `CustomerTopUpInquiryStatus`
(Phase 17). This phase's own wording scopes the two additions to "on
request" only, so the response is field-identical to
`TransactionStatusInquiryBankResponse` (full-equality reflection guard,
matching Phase 17's precedent), while the request is NOT a pure
superset of `TransactionStatusInquiryBankRequest`: `Amount` there is
Optional (`*Money`), but here it is explicitly promoted to Mandatory
(`amount M`), which per the established rule changes it from a pointer
to a plain `Money` value — a genuine type-shape difference, not just an
added field. `TransferToOTCTransferStatusRequest` is therefore its own
type, not a drift-guarded twin of the base request type.

`TransferToOTCTransferStatusRequest`: `OriginalPartnerReferenceNo
string` O + `OriginalReferenceNo string` O + `OriginalExternalID
string` O + `ServiceCode string` M (no omitempty, from the base
pattern) + `TransactionDate string` O + `CustomerNumber string` M (no
omitempty, added) + `Amount Money` M (no omitempty, plain struct,
added/promoted).

`TransferToOTCTransferStatusResponse`: field-identical to
`TransactionStatusInquiryBankResponse` under its own name, per the
"distinct types per service code even for identical shapes"
convention — guarded by a full-equality reflection test.

### TransferToOTCCancelPayment (46)

`TransferToOTCCancelPaymentRequest`: `OriginalReferenceNo string` C
(omitempty; String(64)) + `OriginalPartnerReferenceNo string` M (no
omitempty) + `OriginalExternalID string` O + `CustomerNumber string` M
(no omitempty) + `Reason string` M (no omitempty; String(512)).

`TransferToOTCCancelPaymentResponse`: `ResponseCode string`,
`ResponseMessage string`, `OriginalReferenceNo string` M (no
omitempty — research explicitly notes this flips from Conditional in
the request to Mandatory in the response) + `CancelTime string` C
(omitempty; "must be filled if cancelled transaction success" — a
data-dependent condition the type system can't express, so Optional
here, same handling as every other Conditional field in the package)
+ `TransactionDate string` O.

## Function behavior

All three follow the exact pattern already established across the
package: marshal `req`, set `hb.Body`, call `t.Do`,
`checkResponseStatus`, unmarshal into `Response`, error if
`ResponseCode == ""`. No method override — POST, matching every
endpoint in this sub-group. `TransferToOTCCreatePayment` and
`TransferToOTCCancelPayment` get the standard non-idempotency doc note
(mutating, state-changing calls); `TransferToOTCTransferStatus` does
not (a read-only status check).
