# Phase 23: MPM / QR — Query Payment, Payment Notification

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.9
(lines 200, 202), Service Codes 51-52. Paired together per advisor
guidance: both reuse the `originalX/serviceCode` base pattern already
established by `TransactionStatusInquiryBank` (Phase 16), so they share
the same drift-risk surface and belong in one review.

## Endpoint 51 (Query Payment) — full-shape reuse of the Transaction Status Inquiry Bank pattern

Research line 200: "originalX/serviceCode pattern + `merchantId/
subMerchantId/externalStoreId O`. Resp adds `paidTime String(25) C`,
`terminalId O`." This names the same base pattern Phase 20's Transfer
Status (45) named ("originalX/serviceCode/status pattern"), which Phase
20 established means: reuse the *entire* `TransactionStatusInquiryBank`
request/response field set (including its bank-transfer-shaped fields
— `BeneficiaryAccountNo`, `SourceAccountNo`, etc. — which the research
document treats as a generic reusable status-response schema across
sub-groups, not something scoped to bank transfers specifically), then
add the fields explicitly listed.

`QRMPMQueryPaymentRequest` is therefore `TransactionStatusInquiryBankRequest`'s
full field set (`OriginalPartnerReferenceNo` O, `OriginalReferenceNo` O,
`OriginalExternalID` O, `ServiceCode` M, `TransactionDate` O, `Amount
*Money` O, `AdditionalInfo` O) plus `MerchantID` O, `SubMerchantID` O,
`ExternalStoreID` O — all newly added fields Optional, so this is a
genuine superset, not an identical shape. Per Phase 20's established
practice for a superset-of-a-named-base relationship, this is
documented here in prose, not enforced with a reflection drift-guard
test (full-equality guards are reserved for types the research
documents as identical shapes; a superset gets no such guard, matching
how Phase 20 treated `TransferToOTCTransferStatusRequest`).

`QRMPMQueryPaymentResponse` is likewise `TransactionStatusInquiryBankResponse`'s
full field set plus `PaidTime` C (omitempty; String(25)) and
`TerminalID` O (omitempty) — again a superset, not identical, and so
not reflection-guarded against the base type.

## Endpoint 52 (Payment Notification) — inbound-only, per the package's "Notification"-named default

Research line 202: "Req: `originalReferenceNo M`, `originalPartnerReferenceNo
O`, `latestTransactionStatus M`, `customerNumber/accountType/
destinationNumber/destinationAccountName O`, `amount O`, `sessionId/
bankCode/externalStoreId O`. Resp: envelope-only."

This is exactly the shape the package's standing convention treats as
a settlement callback: a "Notification"-named endpoint whose request
carries `originalReferenceNo`/`latestTransactionStatus` (a status
report about a prior transaction) and whose response is envelope-only
(the caller's own acknowledgment, not new business data). Per the
convention established after Phase 18's santa-loop correction — default
to **inbound** (struct-only, no calling function) unless the research
explicitly states outward/PJP-initiated direction language (the one
confirmed exception being VA Notify Payment Intrabank, §5.3 line 154) —
and §5.9 line 202 carries no such language, this is modeled as
struct-only: `QRMPMPaymentNotificationRequest`/`Response`, no calling
function, doc comment pointing callers at
`ServerVerifier.VerifyTransactionRequest`, matching the
`NotifyBulkCashInRequest`/`Response` precedent (Phase 18) exactly.

## Types

### QRMPMQueryPayment (51)

`QRMPMQueryPaymentRequest`: `OriginalPartnerReferenceNo string` O +
`OriginalReferenceNo string` O + `OriginalExternalID string` O +
`ServiceCode string` M (no omitempty) + `TransactionDate string` O +
`Amount *Money` O + `MerchantID string` O + `SubMerchantID string` O +
`ExternalStoreID string` O + `AdditionalInfo json.RawMessage` O
(carried over from the base pattern, per the same reasoning Phase 20
used for `TransferToOTCTransferStatusRequest`: research's "adds X, Y"
wording describes additions, not a subtraction, and the base pattern's
own request always documents `additionalInfo`).

`QRMPMQueryPaymentResponse`: `ResponseCode string`, `ResponseMessage
string`, then the full `TransactionStatusInquiryBankResponse` field set
verbatim (same names, tags, and omitempty markers) plus `PaidTime
string` C (omitempty) + `TerminalID string` O (omitempty).

### QRMPMPaymentNotification (52) — inbound only, no calling function

`QRMPMPaymentNotificationRequest`: `OriginalReferenceNo string` M (no
omitempty) + `OriginalPartnerReferenceNo string` O + `LatestTransactionStatus
string` M (no omitempty) + `CustomerNumber string` O + `AccountType
string` O + `DestinationNumber string` O + `DestinationAccountName
string` O + `Amount *Money` O + `SessionID string` O + `BankCode
string` O + `ExternalStoreID string` O.

`QRMPMPaymentNotificationResponse`: `ResponseCode string`,
`ResponseMessage string` — envelope-only per research line 202, no
other fields (matches the minimal envelope-only response shape; no
existing precedent type is reused since none is exactly this shape).

## Function behavior

`QRMPMQueryPayment` follows the package's standard pattern: marshal
`req`, set `hb.Body`, call `t.Do`, `checkResponseStatus`, unmarshal into
`Response`, error if `ResponseCode == ""`. POST, no method override.
This is a read-only status query and carries no non-idempotency note
(matching `TransactionStatusInquiryBank`'s precedent).

`QRMPMPaymentNotification` has no calling function — callers receive
this on their own inbound HTTP handler for this path, verify it with
`ServerVerifier.VerifyTransactionRequest`, and `json.Unmarshal` the body
into `QRMPMPaymentNotificationRequest`.

Path for 51 from research §1's inventory table: `qr/qr-mpm-query`.
Endpoint 52's inbound path (`qr/qr-mpm-notify`, research §1 line 53) is
recorded for documentation completeness only — inbound endpoints have
no `EndpointURL` in this package, since the caller owns the receiving
route.
