# Phase 27: Direct Debit Payment Notification, Cancel, Refund

Source: `docs/research/2026-09-12-transfer-debit-portal-research.md` §5.1
(lines 143-169), Service Codes 56-58 — the remaining 3 of Transfer
Debit's 5 Direct Debit endpoints. **Completes the Direct Debit
sub-group.** Follows Phase 26 (54-55).

All three are flat (no new array-item types), per advisor's original
phase-sizing recommendation grouping 56-58 together for that reason.

## Direct Debit Payment Notification (56) — inbound, struct-only

Per the package's established convention for "Notify"-named endpoints
(`QRMPMPaymentNotification`, Phase 23; `NotifyBulkCashIn`, Phase 18):
struct-only, no calling function. Research §5.1 line 143 states no
outward/PJP-initiated direction language (contrast VA Notify Payment
Intrabank's explicit "Callback the PJP sends outward"), so this
defaults to inbound — a caller wires their own HTTP handler, verifies
with `ServerVerifier.VerifyTransactionRequest`, and unmarshals into
`DirectDebitPaymentNotificationRequest`.

`DirectDebitPaymentNotificationRequest`: `OriginalPartnerReferenceNo O`,
`OriginalReferenceNo string` M (no omitempty), `OriginalExternalID O`
(String(32)), `MerchantID O`, `SubMerchantID O`, `Amount *Money O`,
`LatestTransactionStatus string` M (no omitempty), `TransactionStatusDesc
O`, `CreatedTime O` (String(25)), `FinishedTime O` (String(25)),
`ExternalStoreID O`, `AdditionalInfo O`.

`DirectDebitPaymentNotificationResponse`: envelope + `ApprovalCode O`
(String(20)) only. Research's worked example shows a non-standard
field order (`responseCode, approvalCode, responseMessage`) — recorded
as a curiosity in the doc comment since JSON field order carries no
semantic meaning; the Go struct keeps the conventional
envelope-first-then-additions order used everywhere else in the
package.

## Direct Debit Payment Cancel (57)

Own independent type (not shared with `QRMPMCancelPaymentRequest`,
matching the "distinct types per service code" convention already
applied to that Phase 24 sibling). `OriginalPartnerReferenceNo` is the
only Mandatory request field.

`DirectDebitPaymentCancelRequest`: `OriginalPartnerReferenceNo string` M
(no omitempty), `OriginalReferenceNo O`, `ApprovalCode O` (String(20)),
`OriginalExternalID O`, `MerchantID O`, `SubMerchantID O`, `Reason O`
(String(256)), `ExternalStoreID O`, `Amount *Money O`, `AdditionalInfo
O`.

`DirectDebitPaymentCancelResponse`: `OriginalPartnerReferenceNo O`,
`OriginalReferenceNo C` (success only, omitempty), `OriginalExternalID
O`, `CancelTime C` ("required if successful", omitempty — matching
`QRMPMCancelPaymentResponse.CancelTime`'s identical Conditional
treatment, Phase 24), `TransactionDate O`, `AdditionalInfo O`. No
Mandatory field beyond the envelope.

## Direct Debit Payment Refund (58)

Own independent type (not shared with `QRMPMRefundPaymentRequest`,
same convention as Cancel above). `OriginalPartnerReferenceNo` and
`PartnerRefundNo` are Mandatory request fields.

`DirectDebitPaymentRefundRequest`: `MerchantID O`, `SubMerchantID O`,
`OriginalPartnerReferenceNo string` M (no omitempty), `OriginalReferenceNo
O`, `OriginalExternalID O`, `PartnerRefundNo string` M (no omitempty),
`RefundAmount *Money O`, `ExternalStoreID O`, `Reason O` (String(256)),
`AdditionalInfo O`.

`DirectDebitPaymentRefundResponse`: `OriginalPartnerReferenceNo O`,
`OriginalReferenceNo C` (success only, omitempty), `OriginalExternalID
O`, `PartnerTrxID O` (String(32)), `RefundNo string` M (no omitempty),
`PartnerRefundNo string` M (no omitempty), `RefundAmount *Money O`,
`RefundTime string` M (no omitempty), `AdditionalInfo O`. Three
Mandatory fields beyond the envelope, matching
`QRMPMRefundPaymentResponse`'s own precedent of RefundNo/RefundTime
being Mandatory (that sibling type has no PartnerRefundNo mandatory
marker in its response, so this is not a byte-for-byte copy — it is
independently derived from this endpoint's own field table).

## Function behavior (Cancel, Refund)

Both follow the package's standard pattern: marshal `req`, set
`hb.Body`, call `t.Do`, `checkResponseStatus`, unmarshal into
`Response`, error if `ResponseCode == ""`. POST, no method override.
Not idempotent (matching every other cancel/refund endpoint in this
package, e.g. `QRMPMCancelPayment`, `QRMPMRefundPayment`) —
non-idempotency doc-comment note included, keyed on X-EXTERNAL-ID.

Paths from research §1: `debit/notify` (56, no function), `debit/cancel`
(57), `debit/refund` (58).

## FieldCounts guards

All three types (plus their responses) get `FieldCounts` guard tests
per the established convention, given the pattern of a field silently
added without updating literal-map wire-assertion tests. Given round 1
of Phase 26's santa-loop, each new type here also gets a zero-value
mandatory-field presence test up front (not deferred to a review
finding this time), mirroring `TestMPMMerchantInfo_FieldsHaveNoOmitempty`.
