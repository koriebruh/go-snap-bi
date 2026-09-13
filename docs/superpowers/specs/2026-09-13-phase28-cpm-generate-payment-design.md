# Phase 28: Generate QR CPM, CPM Payment

Source: `docs/research/2026-09-12-transfer-debit-portal-research.md` §5.2
(lines 171-193), Service Codes 59-60 — the first two of Transfer
Debit's 6 CPM endpoints. First CPM sub-group phase, following the
completed Direct Debit sub-group (Phases 26-27).

Per advisor's original phase-sizing recommendation: Phase 28 = 59-60
(60 brings a new nested `scannerInfo` object and an untyped `items`
field); Phase 29 = 61, 62, 79, 80 (three of those four have direct
Transfer Kredit precedent).

## Naming

`CPMGenerateQR` / `CPMPayment`, prefixed `CPM` — the `qr/` path prefix
is shared across MPM/QR, CPM, and (later) Auth Payment sub-groups, so
path alone doesn't disambiguate; `CPM` is not used anywhere else in the
package.

## Generate QR CPM (59)

`CPMGenerateQRRequest`: `PartnerReferenceNo O`, `UserAccessToken O`
(String(64)), `MerchantID O`, `SubMerchantID O`, `PartnerTrxDate
string` M (String(25), no omitempty), `AdditionalInfo O`.

`CPMGenerateQRResponse`: `ReferenceNo O`, `PartnerReferenceNo O`,
`QRContent O` (String(512)), `QRURL O` (String(255)), `ExpiryTime
string` M (String(25), no omitempty), `AdditionalInfo O`.

Note: unlike Generate QR MPM (Transfer Kredit, Service 47), research
states `qrContent`/`qrUrl` here carry no one-of-three conditional rule
— both are plain Optional, and there is no `qrImage` field in this
sub-group at all. This is not a byte-for-byte copy of
`GenerateQRMPMResponse`; it's independently derived from this
endpoint's own field table.

Function: POST, no method override, path `qr/qr-cpm-generate`. Not
idempotent (QR-generating call, matching `GenerateQRMPM`'s precedent)
— non-idempotency doc-comment note included.

## CPM Payment (60)

New nested type, `CPMPaymentScannerInfo` (all-Optional per research:
`DeviceID O` (String(64)), `DeviceVersion O` (String(128)),
`DeviceModel O` (String(128)), `DeviceIP O` (String(64))) — the
container `scannerInfo` is itself Optional, so `*CPMPaymentScannerInfo`
per the package's Optional-nested-object convention.

`items` is documented as an unstructured "Object" with no item-level
field table given anywhere in research — modeled as `json.RawMessage`,
matching the package's existing treatment of genuinely untyped/opaque
fields (same rationale as `AdditionalInfo`, not a new convention).

`amount`/`feeAmount`: Optional containers with Mandatory members (same
pattern research flags elsewhere, e.g. Direct Debit Payment) —
resolved per the established handling: `*Money`, ignore internal-member
markers.

`CPMPaymentRequest`: `PartnerReferenceNo string` M (no omitempty),
`QRContent string` M (String(512), no omitempty), `Amount *Money O`,
`FeeAmount *Money O`, `MerchantID string` M (no omitempty),
`SubMerchantID O`, `Title O` (String(256)), `ExpiryTime O`
(String(25)), `Items json.RawMessage O`, `ExternalStoreID O`,
`MerchantName O` (String(64)), `MerchantLocation O` (String(64)),
`AcquirerName O` (String(64)), `TerminalID O` (String(32)),
`ScannerInfo *CPMPaymentScannerInfo O`, `AdditionalInfo O`.

`CPMPaymentResponse`: `ReferenceNo C` (success only, omitempty),
`PartnerReferenceNo O`, `TransactionDate O` (String(25)),
`AdditionalInfo O`. No Mandatory field beyond the envelope.

Function: POST, no method override, path `qr/qr-cpm-payment`. Not
idempotent (payment-initiating call) — non-idempotency doc-comment
note included, keyed on X-EXTERNAL-ID per existing wording.

## FieldCounts guards

All new types (`CPMGenerateQRRequest`/`Response`,
`CPMPaymentRequest`/`Response`, `CPMPaymentScannerInfo`) get
`FieldCounts` guard tests per the established convention.
