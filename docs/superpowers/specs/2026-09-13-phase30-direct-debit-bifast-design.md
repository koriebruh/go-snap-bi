# Phase 30: Direct Debit BI-FAST (Registrasi e-Mandate, Trigger Transfer, Notify)

Source: `docs/research/2026-09-12-transfer-debit-portal-research.md` §5.4
(lines 317-346), Service Codes 70-72 — all 3 Direct Debit BI-FAST
endpoints. **Completes Direct Debit BI-FAST.** Per advisor's original
phase-sizing plan, this sub-group is tackled before Auth Payment
(63-69), whose GET/POST transport method is unsettled — no reason to
front-load the unsettled sub-group when this one has no such
contradiction.

Only Auth Payment (63-69) remains after this phase to complete all of
Transfer Debit.

## Naming

`DirectDebitBIFASTEMandateRegistration` (70), `DirectDebitBIFASTPayment`
(71, "Trigger Direct Debit Transfer / Payment" per research's own
endpoint title), `DirectDebitBIFASTNotification` (72) — `BIFAST`
disambiguates from the existing `DirectDebit*` (non-BI-FAST) types,
per advisor's original naming note for this sub-group.

## `sourceAccountNo` length discrepancy (research §6 item 4) — no code consequence

Research flags `sourceAccountNo` as `String(19)` in Registrasi
e-Mandate (70) but `String(34)` in Trigger Transfer (71) and Notify
(72), for what is presumably the same logical field. This package
enforces no field-length validation anywhere (a package-wide fact, not
specific to this field), so the discrepancy has zero code consequence
— modeled as a plain `string` in both places regardless of the
length comment. Recorded here per research's own instruction not to
silently resolve it as a copy-paste error.

## Registrasi e-Mandate (70)

`DirectDebitBIFASTEMandateRegistrationRequest`: `PartnerReferenceNo O`,
`BankCode string` M (String(11), no omitempty), `SourceAccountNo
string` M (String(19), no omitempty), `SourceAccountName string` M
(String(100), no omitempty), `MaxAmount *Money O` (container Optional,
members Mandatory — resolved per the package's established handling),
`BillerID string` M (String(30), no omitempty), `BillerName string` M
(String(50), no omitempty), `CustomerID string` M (String(45), no
omitempty), `ExpiredDatetime string` M (String(25), no omitempty),
`AdditionalInfo O`.

`DirectDebitBIFASTEMandateRegistrationResponse`: `ReferenceNo C`
(success only, omitempty), `PartnerReferenceNo O`, `EMandateReffID
string` M (String(30), no omitempty), `AdditionalInfo O`.

Function: POST, no method override, path `debit/fast-emandate`. Not
idempotent (a registration-initiating call) — non-idempotency
doc-comment note included, keyed on X-EXTERNAL-ID.

## Trigger Direct Debit Transfer / Payment (71)

`DirectDebitBIFASTPaymentRequest`: `PartnerReferenceNo string` M (no
omitempty), `Currency O` (String(3)), `CustomerReference string` M
(String(30), no omitempty), `FeeType O` (String(25)), `Remark O`
(String(50)), `BeneficiaryAccountNo string` M (String(19), no
omitempty), `BeneficiaryAccountName string` M (String(100), no
omitempty), `TransactionDate string` M (String(25), no omitempty),
`BankCode string` M (String(11), no omitempty), `SourceAccountNo
string` M (String(34) per this endpoint's own row — see the length
discrepancy note above, no omitempty), `SourceAccountName string` M
(String(100), no omitempty), `Amount *Money O` (container Optional,
members Mandatory), `EMandateReffID string` M (String(30), no
omitempty), `AdditionalInfo O`.

`DirectDebitBIFASTPaymentResponse`: `ReferenceNo C` (success only,
omitempty), `PartnerReferenceNo O`, `AdditionalInfo O`. No Mandatory
field beyond the envelope.

Function: POST, no method override, path `debit/fast-payment`. Not
idempotent (a transfer-initiating call) — non-idempotency doc-comment
note included, keyed on X-EXTERNAL-ID.

## Notify (72) — inbound, struct-only

Struct-only per the package's established convention for
"Notify"-named/settlement-callback-shaped endpoints; research gives no
outward/PJP-initiated direction language for this row.

Research names this endpoint's status field `transactionStatus`, not
`latestTransactionStatus` as used everywhere else in both Transfer
Kredit and the rest of Transfer Debit (research §6 item 5, "unexplained
naming inconsistency, recorded as shown, not silently normalized").
This package follows that instruction literally: the Go field is named
`TransactionStatus` (not `LatestTransactionStatus`), matching research's
explicit instruction to preserve the naming as documented rather than
"fix" it to match every other occurrence.

`DirectDebitBIFASTNotificationRequest`: `OriginalReferenceNo string` M
(no omitempty), `OriginalPartnerReferenceNo O`, `OriginalExternalID O`
(String(19)), `TransactionStatus string` M (String(2), the 8-value
enum, no omitempty), `TransactionStatusDesc O` (String(50)),
`EMandateReffID string` M (String(30), no omitempty), `SourceAccountNo
string` M (String(34), no omitempty), `SourceAccountName string` M
(String(100), no omitempty), `Amount *Money O` (container Optional,
members Mandatory), `TraceNo O` (String(16)), `AdditionalInfo O`.

`DirectDebitBIFASTNotificationResponse`: envelope-only
(`ResponseCode`, `ResponseMessage`), matching research's explicit
statement.

## FieldCounts guards

All three new types (six structs including responses) get
`FieldCounts` guard tests per the established convention.
