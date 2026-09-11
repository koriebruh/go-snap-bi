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

Both are pure inquiry (read) calls — no resource is minted and the
group-level response-code table's duplicate-detection notes apply only
to mutating calls, so neither gets a non-idempotency doc comment.

## Testing (mechanical precedent checks)

- `BeneficiaryAccountNo` is the sole mandatory request field without
  `omitempty` on `AccountInquiryInternalRequest`; `BeneficiaryAccountNo`
  and `BeneficiaryBankCode` are the two on `AccountInquiryExternalRequest`
  — each gets an `AlwaysSerialized` test per the package's established
  convention.
- Every response struct field appears in each endpoint's
  `ParsesResponse` fixture.
- Standard 5-test core pattern (full-struct response DeepEqual, request
  wire round-trip, non-2xx-responseCode, non-2xx-status-with-2xx-body,
  2xx-status-with-no-responseCode) per endpoint, plus the
  AlwaysSerialized test(s) above: 6 tests for Internal, 7 for External.
