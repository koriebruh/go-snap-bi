# go-snap-bi: Phase 11 — Trigger Transfer: Intrabank + Interbank Transfer

Status: implemented. Scope: two POST endpoints, batched together — same
Phase 2-10 pattern, no shared-code changes to Transport/checkResponseStatus.
Opens the Trigger Transfer sub-group of Transfer Kredit.
Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.2.

## Shared types

The `amount`-shaped `{value, currency}` object reuses the existing
`Money` type from `balance_inquiry.go` (Phase 2) rather than
introducing a new type — `transaction_history.go` already reuses it
(Phase 3's design doc: "Reuses Money from Phase 2"). Research §3's
`Amount` object matches `Money` field-for-field. This phase is the
package's first request-side use of `Money` (every prior use —
`AccountInfo`, `TransactionDetail`, `SourceOfFund` — is response-side),
so `Money`'s doc comment in `balance_inquiry.go` has been widened from
"per-service response" to "per-service request or response" to match.

`TransferOriginatorInfo` is new this phase — no existing type in the
package matches its shape — defined once in `transfer_shared_types.go`
and reused across the Trigger Transfer sub-group.

| Type.Field | Type | M/O | Go type |
|---|---|---|---|
| Money.Value | String(16,2) | M | `string` |
| Money.Currency | String(3, ISO4217) | M | `string` |
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
| amount | Object | M | `Money` |
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

Research §5.2's response line for this endpoint marks only
`referenceNo` (C) and `partnerReferenceNo` (O) explicitly; every field
after that (`amount`, `beneficiaryAccountNo`, `currency`,
`customerReference`, `sourceAccountNo`, `transactionDate`,
`originatorInfos`, `additionalInfo`) is listed with no M/O marker in the
source. Recorded here as unmarked, not asserted as Optional — the code
treats them as Optional (`omitempty`) as the safe default absent a
stated marker.

| Field | Type | M/O (source marking) | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |
| referenceNo | String(64) | C | `string` |
| partnerReferenceNo | String | O | `string` |
| amount | Object | unmarked | `*Money` |
| beneficiaryAccountNo | String | unmarked | `string` |
| currency | String | unmarked | `string` |
| customerReference | String | unmarked | `string` |
| sourceAccountNo | String | unmarked | `string` |
| transactionDate | String | unmarked | `string` |
| originatorInfos | Array of Object | unmarked | `[]TransferOriginatorInfo` |
| additionalInfo | Object | unmarked | `json.RawMessage` |

`ReferenceNo` is Conditional per the Guides tab; per the package's
established convention (Phase 9's design doc: "referenceNo C fields
elsewhere" carry `omitempty` the same as Optional), Conditional
response fields carry `omitempty` the same as Optional.

`Amount` is `Money` (plain struct, always sent) on the Mandatory
request field and `*Money` (pointer, `omitempty`) on the response
field, which the code treats as Optional per the unmarked-field
convention above — `encoding/json`'s `omitempty` has no effect on a
non-pointer struct value, so an omittable nested-object field needs a
pointer, matching the existing
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

- `transfer_shared_types.go`: `TransferOriginatorInfo` — no functions,
  just the shared struct.
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

- `PartnerReferenceNo`, `Amount`, `BeneficiaryAccountNo`,
  `SourceAccountNo`, `TransactionDate` are the five mandatory
  `IntrabankTransferRequest` fields without `omitempty` — covered by one
  `TestIntrabankTransfer_MandatoryFieldsAlwaysSerialized` test (keyed
  loop for the string fields plus a full-shape assertion on `Amount`,
  matching `TestCardRegistration_MandatoryFieldsAlwaysSerialized`).
- `InterbankTransferRequest` adds `BeneficiaryAccountName` and
  `BeneficiaryBankCode` to that same mandatory set (seven fields total)
  — its own `TestInterbankTransfer_MandatoryFieldsAlwaysSerialized`
  covers all seven.
- Every response struct field appears in each endpoint's
  `ParsesResponse` fixture, including the nested `Money` and
  `[]TransferOriginatorInfo` fields.
- Standard 5-test core pattern per endpoint, plus the
  AlwaysSerialized test: 6 tests per endpoint, 12 total.
