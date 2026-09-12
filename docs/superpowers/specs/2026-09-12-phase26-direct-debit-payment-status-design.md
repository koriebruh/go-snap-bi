# Phase 26: Direct Debit Payment, Direct Debit Payment Status

Source: `docs/research/2026-09-12-transfer-debit-portal-research.md` §5.1
(lines 109-141), Service Codes 54-55 — the first two of Transfer
Debit's 5 Direct Debit endpoints. First Transfer Debit implementation
phase (research pass completed and merged prior, PR #26).

## Naming

`DirectDebitPayment` / `DirectDebitPaymentStatus`, prefixed by
sub-group per advisor's guidance — the `debit/` path prefix is shared
across Direct Debit and Direct Debit BI-FAST (later sub-group), so path
alone would not disambiguate a bare `Payment`/`PaymentStatus` name.

## New item types (three new array item types, per advisor's stated ceiling for one phase)

**`DirectDebitPaymentURLParam`** (54's `urlParams[]`): `URL string M
(String(512))`, `Type string M (String(32))` — PAY_RETURN/PAY_NOTIFY
values, documented as a plain string enum, not modeled as a Go type
since research gives no exhaustive closed list beyond these two
examples. `IsDeeplink string M (String(1))` — Y/N. This is the
documented-String-with-quoted-wire-value case advisor flagged, not the
ambiguous-numeric-type case: plain `string`, not `json.RawMessage`.

**`DirectDebitPayOptionDetail`** (54's `payOptionDetails[]`): `PayMethod
M`, `PayOption M`, `TransAmount *Money O`, `FeeAmount *Money O`,
`CardToken O`, `MerchantToken O`, `AdditionalInfo json.RawMessage O`.
Seven fields, matching the field table exactly. `PayMethod`/
`PayOption` have no documented String() length in the field table, so
plain `string` with no length comment.

**`DirectDebitRefundHistoryItem`** (55 response's `refundHistory[]`):
`RefundNo string C` (success only), `PartnerRefundNo string M`,
`RefundAmount *Money O`, `RefundStatus string M` (String(2), 00/03/06
subset of the 8-value enum — not modeled as a distinct type, plain
string, matching the package's existing convention of not creating enum
types for `latestTransactionStatus` elsewhere), `RefundDate string C`
(date-mandatory/time-optional per the portal's own note — modeled as
Optional/omitempty per the package's Conditional-defaults-to-omitempty
convention used throughout), `Reason string O`.

## `amount` / `payOptionDetails[].transAmount`/`feeAmount`: Optional container, Mandatory members

Research explicitly flags this pattern for 54's top-level `amount`
(container O, `value`/`currency` members M) and by extension the
`payOptionDetails[]` item's own `transAmount`/`feeAmount`. Resolved per
the package's established handling (Phase 21 on): Optional container →
`*Money`, ignore internal-member markers, matching existing precedent
everywhere else in this package. No new decision here.

## Direct Debit Payment (54)

`DirectDebitPaymentRequest`: `PartnerReferenceNo string` M (String(64),
no omitempty), `BankCardToken O`, `ChargeToken O`, `OTP O`, `OTPTrxCode
O`, `MerchantID O`, `TerminalID O`, `JourneyID O`, `SubMerchantID O`,
`Amount *Money O`, `URLParams []DirectDebitPaymentURLParam O`,
`ExternalStoreID O`, `ValidUpTo O`, `PointOfInitiation O`, `FeeType O`,
`DisabledPayMethods O`, `PayOptionDetails []DirectDebitPayOptionDetail
O`, `AdditionalInfo O`.

`DirectDebitPaymentResponse`: `ReferenceNo string C` (success only,
omitempty), `PartnerReferenceNo O`, `ApprovalCode O`, `AppRedirectURL O`
(String(2048)), `WebRedirectURL O` (String(2048)), `AdditionalInfo O`.
No mandatory response fields beyond the envelope (`ResponseCode`/
`ResponseMessage`, both no-omitempty per package convention).

Function: POST, no method override, path `debit/payment-host-to-host`.
Not idempotent (a payment-initiating call, matching precedent set by
every other payment/transfer-initiating endpoint in this package, e.g.
`SubmitBulkCashIn`, `TransferToBankPayment`) — non-idempotency
doc-comment note included, keyed on X-EXTERNAL-ID per existing wording.

## Direct Debit Payment Status (55)

Per §4, uses the same "originalX + serviceCode" core shape as
`TransactionStatusInquiryBankRequest` but with this sub-group's own
distinct additions (`merchantId`/`subMerchantId`/`externalStoreId`
instead of `AdditionalInfo`-adjacent positioning). Left as its own
independent type this phase — not reused/extended from Transfer
Kredit's base type, since the two differ in field composition and the
package has never shared a request type across Transfer Kredit/Transfer
Debit sub-groups. A future phase could introduce a shared base if more
Transfer Debit endpoints turn out to need the identical core, but one
occurrence does not justify an abstraction now (YAGNI).

`DirectDebitPaymentStatusRequest`: `OriginalPartnerReferenceNo O`,
`OriginalReferenceNo O`, `OriginalExternalID O`, `ServiceCode string` M
(no omitempty), `TransactionDate O`, `Amount *Money O`, `MerchantID O`,
`SubMerchantID O`, `ExternalStoreID O`.

`DirectDebitPaymentStatusResponse`: envelope + the same
`OriginalPartnerReferenceNo`/`OriginalReferenceNo`/`OriginalExternalID`/
`ServiceCode`/`TransactionDate`/`Amount` core (all Optional, matching
`TransactionStatusInquiryBankResponse`'s existing omitempty pattern for
these) plus `ApprovalCode O`, `LatestTransactionStatus string` M (no
omitempty, matching every other occurrence of this field package-wide),
`TransactionStatusDesc O`, `OriginalResponseCode O`,
`OriginalResponseMessage O`, `SessionID O`, `RequestID O`,
`RefundHistory []DirectDebitRefundHistoryItem O`, `TransAmount *Money
O`, `FeeAmount *Money O`, `PaidTime string C` (omitempty, Conditional).

Note: this response's addition set (`originalResponseCode`/
`originalResponseMessage`/`sessionId`/`requestId`) overlaps textually
with Phase 25's `TransactionStatusInquiryNonBankRequest` additions —
but there they landed on the *request* (inferred from an absent
`Resp:` marker), while here research explicitly labels them "Resp
adds" (line 132), i.e. response-side. Different sub-groups, different
rows, and this row's placement is stated outright rather than
inferred, so no tension to resolve — recorded here only because a
reviewer diffing the two might otherwise flag it as an inconsistency
in this package's handling.

Function: POST, no method override, path `debit/status`. Read-only
status query, no non-idempotency note (matching
`TransactionStatusInquiryBank`/`TransactionStatusInquiryNonBank`
precedent).

## FieldCounts guards

Both types get `TestDirectDebitPaymentTypes_FieldCounts` and
`TestDirectDebitPaymentStatusTypes_FieldCounts` per the Phase 24/25
convention, given the request/response field counts here are unusually
high and easy to silently drift.
