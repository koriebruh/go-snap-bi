# Phase 29: CPM Query Payment, Cancel Payment, Payment Notification, Refund Payment

Source: `docs/research/2026-09-12-transfer-debit-portal-research.md` §5.2
(lines 195-233), Service Codes 61, 62, 79, 80 — the remaining 4 of
Transfer Debit's 6 CPM endpoints. **Completes the CPM sub-group.**
Follows Phase 28 (59-60).

## Query Payment (61)

`CPMQueryPaymentRequest`: `OriginalReferenceNo O`,
`OriginalPartnerReferenceNo O`, `OriginalExternalID O` (String(32)),
`MerchantID O`, `SubMerchantID O`, `ExternalStoreID O`,
`AdditionalInfo O`. No Mandatory field at all.

`CPMQueryPaymentResponse`: `OriginalReferenceNo C` (success only,
omitempty), `OriginalPartnerReferenceNo O`, `OriginalExternalID O`,
`Title O` (String(256)), `LatestTransactionStatus string` M (no
omitempty), `TransactionStatusDesc O`, `PaidTime string` M (String(25),
no omitempty), `AdditionalInfo O`. Two Mandatory fields beyond the
envelope.

Own independent type — no existing type in the package shares this
exact shape.

## Cancel Payment (62) — research's "structurally identical" claim does not hold; verified against the literal field table instead

Research line 209-212 states this endpoint is "structurally identical
to Transfer Kredit's `QRMPMCancelPaymentRequest`/`Response` (Phase
24)... all three `originalX` fields Optional, matching that same
narrower shape." Read `QRMPMCancelPaymentRequest`/`Response`
(`mpm_qr_cancel_payment.go`) directly to verify this claim before
reusing or shape-guarding against it, per the package's established
"verify a citation against the actual source, don't trust the
citation" practice:

- `QRMPMCancelPaymentRequest`'s Mandatory fields are `MerchantID` and
  `Reason` (both no omitempty); all three `originalX` fields are
  Optional there too — but CPM Cancel Payment (62)'s own field table
  (line 203) states `originalPartnerReferenceNo M`, not Optional, and
  does not mark `merchantId`/`reason` Mandatory at all. The two
  requests' Mandatory fields are disjoint, not matching.
- `QRMPMCancelPaymentResponse` has exactly 4 fields (envelope +
  `CancelTime`/`TransactionDate`) with no `originalX` fields at all.
  CPM Cancel Payment (62)'s response (line 206-208) explicitly adds
  `originalPartnerReferenceNo`/`originalReferenceNo`/`originalExternalId`
  beyond the envelope + `CancelTime`/`TransactionDate` — a materially
  larger shape.

Conclusion: this is **not** an identical shape. The research doc's own
summary sentence overstates the resemblance (likely referring loosely
to "no `serviceCode` field, a narrow originalX-adjacent cancel
pattern" rather than field-for-field identity); the per-field table
earlier in the same entry is authoritative and is what this
implementation follows. Modeled as its own independent type,
`CPMCancelPaymentRequest`/`Response`, with no reflection-drift-guard
against `QRMPMCancelPaymentRequest`/`Response` (they are not the same
shape, so a full-equality guard would be actively wrong).

`CPMCancelPaymentRequest`: `OriginalPartnerReferenceNo string` M (no
omitempty), `OriginalReferenceNo O`, `OriginalExternalID O`,
`MerchantID O`, `SubMerchantID O`, `ExternalStoreID O`, `Amount *Money
O`, `Reason O` (String(256)), `AdditionalInfo O`.

`CPMCancelPaymentResponse`: `OriginalPartnerReferenceNo O`,
`OriginalReferenceNo C` (success only, omitempty), `OriginalExternalID
O`, `CancelTime C` ("filled if successful", omitempty — matching the
package's Conditional-defaults-to-omitempty convention, same treatment
as `QRMPMCancelPaymentResponse.CancelTime`), `TransactionDate O`,
`AdditionalInfo O`. No Mandatory field beyond the envelope.

## Payment Notification (79) — inbound, struct-only

Research explicitly confirms this one matches Transfer Kredit's
`QRMPMPaymentNotification` pattern (envelope-only response, inbound
settlement callback, no calling function) — unlike Cancel Payment
above, this citation is verified consistent with the shape given, not
contradicted. Struct-only per the package's established convention for
"Notify"-named/settlement-callback-shaped endpoints.

`CPMPaymentNotificationRequest`: `OriginalPartnerReferenceNo O`,
`OriginalReferenceNo O`, `MerchantID string` M (no omitempty),
`SubMerchantID O`, `ExternalStoreID O`, `Amount *Money O`,
`LatestTransactionStatus string` M (no omitempty),
`TransactionStatusDesc O`, `CustomerNumber O` (String(64)),
`AccountType O` (String(25)), `DestinationNumber O` (String(25)),
`DestinationAccountName O` (String(25)), `SessionID O` (String(25)),
`BankCode O` (String(11)), `AdditionalInfo O`.

`CPMPaymentNotificationResponse`: envelope-only (`ResponseCode`,
`ResponseMessage`), matching research's explicit statement.

## Refund Payment (80)

`CPMRefundPaymentRequest`: `MerchantID O`, `SubMerchantID O`,
`ExternalStoreID O`, `OriginalPartnerReferenceNo string` M (no
omitempty), `OriginalReferenceNo O`, `OriginalExternalID O`,
`PartnerRefundNo string` M (String(64), no omitempty), `RefundAmount
*Money O`, `Reason O` (String(256)), `AdditionalInfo O`.

`CPMRefundPaymentResponse`: `OriginalPartnerReferenceNo O`,
`OriginalReferenceNo O` (research gives no C/M marker for this one
field in this specific endpoint row, unlike every other occurrence of
this field in the package — recorded as shown, modeled Optional per
the package's default-to-omitempty-when-unmarked handling),
`OriginalExternalID O`, `RefundNo string` M (String(64), no omitempty),
`PartnerRefundNo O` (String(64) — Optional here, unlike Direct Debit
Payment Refund's Mandatory `PartnerRefundNo` in its response, Phase
27 — independently derived, not copied), `RefundAmount *Money O`,
`RefundTime string` M (String(25), no omitempty), `AdditionalInfo O`.

Own independent type — `QRMPMRefundPaymentResponse` (Phase 24) has no
`originalX` fields at all, so this is a materially different shape,
same reasoning as Cancel Payment above.

## Function behavior (Query, Cancel, Refund)

All three follow the package's standard pattern: marshal `req`, set
`hb.Body`, call `t.Do`, `checkResponseStatus`, unmarshal into
`Response`, error if `ResponseCode == ""`. POST, no method override.
Query is read-only (no non-idempotency note, matching
`TransactionStatusInquiryBank`/`CPMQueryPayment` precedent). Cancel and
Refund are not idempotent (matching `QRMPMCancelPayment`/
`QRMPMRefundPayment`) — non-idempotency doc-comment note included,
keyed on X-EXTERNAL-ID.

Paths from research §1: `qr/qr-cpm-query` (61), `qr/qr-cpm-cancel`
(62), `qr/qr-cpm-notify` (79, no function), `qr/qr-cpm-refund` (80).

## FieldCounts guards

All four new types (eight structs total including responses) get
`FieldCounts` guard tests per the established convention.
