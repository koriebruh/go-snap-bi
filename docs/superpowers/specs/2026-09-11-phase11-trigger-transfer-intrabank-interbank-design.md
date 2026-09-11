# go-snap-bi: Phase 11 — Trigger Transfer: Intrabank + Interbank Transfer

Status: implemented. Scope: two POST endpoints, batched together — same
Phase 2-10 pattern, no shared-code changes to Transport/checkResponseStatus.
Opens the Trigger Transfer sub-group of Transfer Kredit.
Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.2.

## Shared types introduced this phase

`TransferAmount` and `TransferOriginatorInfo` are defined once in
`transfer_shared_types.go` and reused across the Trigger Transfer
sub-group. Per research §3, both are documented as all-String fields
with no worked-example wire shape showing a non-string representation.

| Type.Field | Type | M/O | Go type |
|---|---|---|---|
| TransferAmount.Value | String(16,2) | M | `string` |
| TransferAmount.Currency | String(3, ISO4217) | M | `string` |
| TransferOriginatorInfo.OriginatorCustomerNo | String(34) | M | `string` |
| TransferOriginatorInfo.OriginatorCustomerName | String(100) | M | `string` |
| TransferOriginatorInfo.OriginatorBankCode | String(11) | M | `string` |

## Ambiguous-type rule applied

Can the documented type have a non-string JSON representation? No field
in this phase is documented as Numeric/Decimal. All fields are `string`.

## Endpoint 1: Intrabank Transfer (Service Code 17)

Path `.../{version}/transfer-intrabank`. POST.

### Request body

| Field | Type | M/O/C | Go type |
|---|---|---|---|
| partnerReferenceNo | String | M | `string` |
| amount | Object | M | `TransferAmount` |
| beneficiaryAccountNo | String(34) | M | `string` |
| beneficiaryEmail | String | O | `string` |
| currency | String | O | `string` |
| customerReference | String(30) | O | `string` |
| feeType | String(25) | O | `string` |
| remark | String | O | `string` |
| sourceAccountNo | String(19) | M | `string` |
| transactionDate | String(25) | M | `string` |
| originatorInfos | Array of Object | C | `[]TransferOriginatorInfo` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body

| Field | Type | M/O | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |
| referenceNo | String(64) | C | `string` |
| partnerReferenceNo | String | O | `string` |
| amount | Object | O | `*TransferAmount` |
| beneficiaryAccountNo | String | O | `string` |
| currency | String | O | `string` |
| customerReference | String | O | `string` |
| sourceAccountNo | String | O | `string` |
| transactionDate | String | O | `string` |
| originatorInfos | Array of Object | O | `[]TransferOriginatorInfo` |
| additionalInfo | Object | O | `json.RawMessage` |

`ReferenceNo` is Conditional per the Guides tab; per the package's
established convention (e.g. Phase 9's `CardRegistrationUnbindingResponse`
Conditional fields), Conditional response fields carry `omitempty` the
same as Optional.

`Amount` is `TransferAmount` (plain struct, always sent) on the
Mandatory request field and `*TransferAmount` (pointer, `omitempty`) on
the Optional response field — `encoding/json`'s `omitempty` has no
effect on a non-pointer struct value, so an Optional nested-object field
needs a pointer to actually be omittable, matching the existing
`AccountBindingResponse.AccessTokenInfo *BindingAccessTokenInfo` pattern.

## Endpoint 2: Interbank Transfer (Service Code 18)

Path `.../{version}/transfer-interbank`. POST. Same shape as Intrabank
Transfer with these differences:

### Request body

Adds `beneficiaryAccountName String(100) M`, `beneficiaryAddress String
O`, `beneficiaryBankCode String(11) M`, `beneficiaryBankName String O`.

### Response body

Adds `traceNo String(16) O`.

## Design

Three new files:

- `transfer_shared_types.go`: `TransferAmount`, `TransferOriginatorInfo`
  — no functions, just the two shared structs.
- `transfer_intrabank.go`: `IntrabankTransferRequest`,
  `IntrabankTransferResponse`, `IntrabankTransfer(ctx, t, hb, req)`.
- `transfer_interbank.go`: `InterbankTransferRequest`,
  `InterbankTransferResponse`, `InterbankTransfer(ctx, t, hb, req)`.

Both are mutating (fund-moving) POST calls. Per the group-level
response-code table (research §4), `409/xx/01 Duplicate
partnerReferenceNo` is the dedup key and `409/xx/00 Conflict` is the
daily X-EXTERNAL-ID dedup dimension — the same shape of note this
package already carries on `AccountBinding`/`CardRegistration` et al.
Both functions get the package's standard non-idempotency doc comment.

## Testing (mechanical precedent checks)

- `PartnerReferenceNo`, `BeneficiaryAccountNo`, `SourceAccountNo`,
  `TransactionDate` are the four mandatory `IntrabankTransferRequest`
  fields without `omitempty` — covered by one
  `TestIntrabankTransfer_MandatoryFieldsAlwaysSerialized` test (keyed
  loop, matching `TestCardRegistration_MandatoryFieldsAlwaysSerialized`).
- `InterbankTransferRequest` adds `BeneficiaryAccountName` and
  `BeneficiaryBankCode` to that same mandatory set — its own
  `TestInterbankTransfer_MandatoryFieldsAlwaysSerialized` covers all six.
- Every response struct field appears in each endpoint's
  `ParsesResponse` fixture, including the nested `TransferAmount` and
  `[]TransferOriginatorInfo` fields.
- Standard 5-test core pattern per endpoint, plus the
  AlwaysSerialized test: 6 tests per endpoint, 12 total.
