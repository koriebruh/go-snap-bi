# go-snap-bi: Phase 10 — Account Inquiry (Transfer Kredit)

Status: implemented. Scope: two POST endpoints, batched together — both
follow the Phase 2-9 marshal/hb.Body/checkResponseStatus/decode pattern
exactly, no shared-code changes needed. Opens the Transfer Kredit group.
Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.1.

## Ambiguous-type rule applied (unchanged from Phase 7/9)

Can the documented type have a non-string JSON representation? No field
in this sub-group is documented as Numeric/Decimal, and the worked
example for Internal Account Inquiry shows every field quoted. All
fields are `string`.

## Endpoint 1: Internal Account Inquiry (Service Code 15)

Path `.../{version}/account-inquiry-internal`. POST.

### Request body

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerReferenceNo | String(64) | O | `string` |
| beneficiaryAccountNo | String(34) | M | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body

| Field | Type | M/O | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |
| referenceNo | String(64) | O | `string` |
| partnerReferenceNo | String | O | `string` |
| beneficiaryAccountName | String(100) | M | `string` |
| beneficiaryAccountNo | String | M | `string` |
| beneficiaryAccountStatus | String(16) | O | `string` |
| beneficiaryAccountType | String(1) | O | `string` |
| currency | String(3) | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

`beneficiaryAccountType`'s worked example value is `"D"` (research §5.1
records a "D"/"S" enum).

## Endpoint 2: External Account Inquiry (Service Code 16)

Path `.../{version}/account-inquiry-external`. POST. Same shape as
Internal Account Inquiry with these differences:

### Request body

Adds `beneficiaryBankCode String(11) M`.

### Response body

Adds `beneficiaryBankName String(50) O`; drops `beneficiaryAccountType`
and `beneficiaryAccountStatus` (not present in this endpoint's Guides
tab response table).

## Design

Two new files, each following the Phase 2-9 shape exactly:

- `account_inquiry_internal.go`: `AccountInquiryInternalRequest`,
  `AccountInquiryInternalResponse`, `AccountInquiryInternal(ctx, t, hb, req)`.
- `account_inquiry_external.go`: `AccountInquiryExternalRequest`,
  `AccountInquiryExternalResponse`, `AccountInquiryExternal(ctx, t, hb, req)`.

Neither function carries a non-idempotency doc comment, matching
BalanceInquiry, TransactionHistoryList, AccountBindingInquiry, and
CardRegistrationInquiry — the note appears only on
account_creation.go, card_registration.go, account_unbinding.go,
card_registration_unbinding.go, and verify_otp.go, and on none of the
package's inquiry endpoints.

## Testing (mechanical precedent checks)

- `BeneficiaryAccountNo` is the sole mandatory request field without
  `omitempty` on `AccountInquiryInternalRequest` — gets a dedicated
  `AlwaysSerialized` test per the package's established convention.
- `BeneficiaryAccountNo` and `BeneficiaryBankCode` are the two mandatory
  fields on `AccountInquiryExternalRequest` — covered by one
  `TestAccountInquiryExternal_MandatoryFieldsAlwaysSerialized` test,
  mirroring `TestCardRegistration_MandatoryFieldsAlwaysSerialized` (the
  package's precedent for a struct with two such fields).
- Every response struct field appears in each endpoint's
  `ParsesResponse` fixture.
- Standard 5-test core pattern (full-struct response DeepEqual, request
  wire round-trip, non-2xx-responseCode, non-2xx-status-with-2xx-body,
  2xx-status-with-no-responseCode) per endpoint, plus the
  AlwaysSerialized test(s) above: 6 tests for Internal, 6 for External.
