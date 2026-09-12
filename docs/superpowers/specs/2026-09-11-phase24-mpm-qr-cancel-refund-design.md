# Phase 24: MPM / QR — Cancel Payment, Refund Payment

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.9
(lines 204, 206), Service Codes 77-78. Paired per advisor guidance: both
are mutating, merchant-scoped operations carrying a `reason` field and
the standard non-idempotency note — same shape family.

This is the last pair in the MPM/QR sub-group; §5.10's Transaction
Status Inquiry (non-bank), Service Code 53, is deferred to Phase 25,
which completes Transfer Kredit entirely.

## Endpoint 77 (Cancel Payment)

Research line 204: "Req: `originalX O`, `merchantId M`, `subMerchantId/
externalStoreId O`, `reason string M`, `amount O`. Resp: `cancelTime C`,
`transactionDate O`." Unlike every other `originalX` base-pattern reuse
in this package (Phase 16/20/23), this row has **no `serviceCode`
field** and **all three `originalX` fields are Optional** (no mandatory
identifier among them) — a narrower base than
`TransactionStatusInquiryBankRequest`. This is modeled as its own type
with just the three `originalX` fields, not a reuse of the larger base
pattern, since the research row genuinely lists no `serviceCode` and no
`M` among the identifiers.

`MerchantID` is Mandatory here — the first Cancel-style endpoint in the
package where a merchant identifier, not an original-transaction
identifier, is the required field. `Reason` is Mandatory (no documented
length in this row, unlike Transfer To OTC Cancel Payment's
`Reason String(512)`, Phase 20).

## Endpoint 78 (Refund Payment)

Research line 206: "Req: `merchantId/subMerchantId/externalStoreId O`,
`originalPartnerReferenceNo M`, `originalReferenceNo O`,
`originalExternalId O`, `partnerRefundNo String(64) M`, `refundAmount
O`, `reason String(256) O`. Resp: `refundNo String(64) M`,
`partnerRefundNo O`, `refundAmount O`, `refundTime String(25) M`."

`PartnerRefundNo` (request, Mandatory) and `RefundNo`/`RefundTime`
(response, both Mandatory) are new fields not seen elsewhere in the
package — no base-pattern reuse applies to this endpoint at all; it is
modeled directly from this row alone.

## Types

### QRMPMCancelPayment (77)

`QRMPMCancelPaymentRequest`: `OriginalPartnerReferenceNo string` O +
`OriginalReferenceNo string` O + `OriginalExternalID string` O +
`MerchantID string` M (no omitempty) + `SubMerchantID string` O +
`ExternalStoreID string` O + `Reason string` M (no omitempty) +
`Amount *Money` O.

`QRMPMCancelPaymentResponse`: `ResponseCode string`, `ResponseMessage
string`, `CancelTime string` C (omitempty) + `TransactionDate string` O
(omitempty).

### QRMPMRefundPayment (78)

`QRMPMRefundPaymentRequest`: `MerchantID string` O + `SubMerchantID
string` O + `ExternalStoreID string` O + `OriginalPartnerReferenceNo
string` M (no omitempty) + `OriginalReferenceNo string` O +
`OriginalExternalID string` O + `PartnerRefundNo string` M (no
omitempty; String(64)) + `RefundAmount *Money` O + `Reason string` O
(String(256)).

`QRMPMRefundPaymentResponse`: `ResponseCode string`, `ResponseMessage
string`, `RefundNo string` M (no omitempty; String(64)) +
`PartnerRefundNo string` O (omitempty) + `RefundAmount *Money` O
(omitempty) + `RefundTime string` M (no omitempty; String(25)).

## Function behavior

Both follow the package's standard pattern: marshal `req`, set
`hb.Body`, call `t.Do`, `checkResponseStatus`, unmarshal into
`Response`, error if `ResponseCode == ""`. POST, no method override.
Both are mutating/state-changing calls and get the standard
non-idempotency doc note.

Paths from research §1's inventory table: `qr/qr-mpm-cancel` (77, line
54), `qr/qr-mpm-refund` (78, line 55).
