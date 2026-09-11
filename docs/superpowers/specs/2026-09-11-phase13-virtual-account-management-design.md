# go-snap-bi: Phase 13 — Virtual Account: Management (Create/Update/UpdateStatus/Inquiry/Delete)

Status: implemented. Scope: five endpoints (Service Codes 27, 28, 29,
30, 31), the "management" half of the Virtual Account sub-group — the
half research §6's open contradictions do not touch. `customerNo` is
quoted-string in all five worked examples (research §3.4's ambiguity
list), and none of these five uses `channelCode`/`paymentType`.
Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.3.

## Response envelope shape

Research §5.3 records a response-envelope top-level field name split:
`virtualAccountData` (capital D) for endpoints 24-31, `virtualAccountdata`
(lowercase d) for 32-35. All five endpoints in this phase are in the
capital-D group, so every response here wraps its business fields under
`virtualAccountData`.

Per this package's established convention for near-identical shapes
across distinct service codes (Phase 12's `RTGSTransferResponse`/
`SKNBITransferResponse`), each endpoint's nested data object gets its
own named Go type rather than being shared across service codes, even
where research states two endpoints have the same shape.

## Shared types introduced this phase

`LocalizedText` (`{english, indonesia string}`, research §3) and
`BillDetail` (research §3's `billDetails[]` shape) are added to
`transfer_shared_types.go`. Neither's own fields carry an M/O marker in
the research doc; both are unmarked, so the code treats them as
Optional (`omitempty`) as the safe default absent a stated marker, the
same convention already applied to unmarked trailing fields in Phase 11.

| Type.Field | Type | Go type |
|---|---|---|
| LocalizedText.English | String | `string` |
| LocalizedText.Indonesia | String | `string` |
| BillDetail.BillCode | (unmarked) | `string` |
| BillDetail.BillNo | (unmarked) | `string` |
| BillDetail.BillName | (unmarked) | `string` |
| BillDetail.BillShortName | (unmarked) | `string` |
| BillDetail.BillDescription | Object (LocalizedText) | `*LocalizedText` |
| BillDetail.BillSubCompany | (unmarked) | `string` |
| BillDetail.BillAmount | Object (Money) | `*Money` |
| BillDetail.AdditionalInfo | Object | `json.RawMessage` |
| BillDetail.BillAmountLabel | (unmarked) | `string` |
| BillDetail.BillAmountValue | (unmarked) | `string` |
| BillDetail.BillReferenceNo | (unmarked) | `string` |
| BillDetail.Status | (unmarked) | `string` |
| BillDetail.Reason | Object (LocalizedText) | `*LocalizedText` |

`BillDescription`, `BillAmount`, and `Reason` are the three nested-object
fields — pointers with `omitempty`, since `omitempty` has no effect on a
non-pointer struct value (a verified `encoding/json` fact, not a new
claim).

## Endpoint 1: VA - Create VA (Service Code 27)

Path `.../{version}/transfer-va/create-va`. POST.

### Request body

Research §5.3: the identity triple (`partnerServiceId`, `customerNo`,
`virtualAccountNo`) is Optional here, unlike every other VA endpoint
where it is Mandatory.

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerServiceId | String(8) | O | `string` |
| customerNo | String(20) | O | `string` |
| virtualAccountNo | String(28) | O | `string` |
| virtualAccountName | String | M | `string` |
| trxId | String(64) | M | `string` |
| totalAmount | Object | O | `*Money` |
| billDetails | Array of Object | O | `[]BillDetail` |
| freeTexts | Array of Object | O | `[]LocalizedText` |
| virtualAccountTrxType | String(1) | O | `string` |
| feeAmount | Object | O | `*Money` |
| expiredDate | String(25) | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body (`CreateVAData`, wrapped as `virtualAccountData`)

Same field set as the request (identity triple + business fields), no
`lastUpdateDate`/`paymentDate` (those are added only by Update VA's
response per research §5.3).

## Endpoint 2: VA - Update VA (Service Code 28, PUT)

Path `.../{version}/transfer-va/update-va`. PUT — the package's first
PUT binding. `hb.Method` is set to `http.MethodPut` by the function
itself, the same way `CardRegistrationInquiry` sets `hb.Method` to
`http.MethodGet`, since the method is fixed by the endpoint rather than
caller-configurable.

### Request body

Same shape as Create VA's request, except the identity triple is
Mandatory here (research §5.3: "same request shape as Create VA but
identity triple back to Mandatory").

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerServiceId | String(8) | M | `string` |
| customerNo | String(20) | M | `string` |
| virtualAccountNo | String(28) | M | `string` |
| virtualAccountName | String | M | `string` |
| trxId | String(64) | M | `string` |
| totalAmount | Object | O | `*Money` |
| billDetails | Array of Object | O | `[]BillDetail` |
| freeTexts | Array of Object | O | `[]LocalizedText` |
| virtualAccountTrxType | String(1) | O | `string` |
| feeAmount | Object | O | `*Money` |
| expiredDate | String(25) | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body (`UpdateVAData`, wrapped as `virtualAccountData`)

Create VA's response fields plus `lastUpdateDate` and `paymentDate`
(both unmarked in research §5.3, so `string` with `omitempty`).

## Endpoint 3: VA - Update Status VA (Service Code 29, PUT)

Path `.../{version}/transfer-va/update-status`. PUT — `hb.Method` set
to `http.MethodPut` by the function itself.

### Request body

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerServiceId | String(8) | M | `string` |
| customerNo | String(20) | M | `string` |
| virtualAccountNo | String(28) | M | `string` |
| trxId | String(64) | M | `string` |
| paidStatus | String(1) | M | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body (`UpdateStatusVAData`, wrapped as `virtualAccountData`)

Research §5.3: "Response is the full VA data object" — same field set
as `UpdateVAData`, under its own type name per this phase's stated
per-endpoint-type convention.

## Endpoint 4: VA - Inquiry VA (Service Code 30)

Path `.../{version}/transfer-va/inquiry-va`. POST.

### Request body

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerServiceId | String(8) | M | `string` |
| customerNo | String(20) | M | `string` |
| virtualAccountNo | String(28) | M | `string` |
| trxId | String(64) | M | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body (`InquiryVAData`, wrapped as `virtualAccountData`)

Research §5.3: "Response is full VA data object (same shape as Update
VA response)" — same field set as `UpdateVAData`, under its own type
name.

## Endpoint 5: VA - Delete VA (Service Code 31, DELETE)

Path `.../{version}/transfer-va/delete-va`. DELETE — the package's
first DELETE binding. `hb.Method` is set to `http.MethodDelete` by the
function itself. Research §5.3 confirms this DELETE ships a JSON
request body, no path parameters, so `hb.Body` is still set the same
way as every POST binding.

### Request body

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerServiceId | String(8) | M | `string` |
| customerNo | String(20) | M | `string` |
| virtualAccountNo | String(28) | M | `string` |
| trxId | String(64) | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body (`DeleteVAData`, wrapped as `virtualAccountData`)

Research §5.3: "identity triple + `trxId String(12) O` (length shrinks
64→12 between req/resp in the table — likely a doc typo) + `additionalInfo`."
Recorded as `string` regardless of the length discrepancy, since Go
string fields carry no length constraint either way.

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerServiceId | String(8) | O | `string` |
| customerNo | String(20) | O | `string` |
| virtualAccountNo | String(28) | O | `string` |
| trxId | String(12) | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

## Design

New files:

- `transfer_shared_types.go` (extended): `LocalizedText`, `BillDetail`.
- `virtual_account_create.go`: `CreateVARequest`, `CreateVAData`,
  `CreateVAResponse`, `CreateVA(ctx, t, hb, req)`.
- `virtual_account_update.go`: `UpdateVARequest`, `UpdateVAData`,
  `UpdateVAResponse`, `UpdateVA(ctx, t, hb, req)`.
- `virtual_account_update_status.go`: `UpdateStatusVARequest`,
  `UpdateStatusVAData`, `UpdateStatusVAResponse`,
  `UpdateStatusVA(ctx, t, hb, req)`.
- `virtual_account_inquiry_va.go`: `InquiryVARequest`, `InquiryVAData`,
  `InquiryVAResponse`, `InquiryVA(ctx, t, hb, req)`.
- `virtual_account_delete.go`: `DeleteVARequest`, `DeleteVAData`,
  `DeleteVAResponse`, `DeleteVA(ctx, t, hb, req)`.

All five response types wrap their data object in a field tagged
`virtualAccountData` (matching the capital-D group), typed as a pointer
(`*CreateVAData` etc.) since the wrapper is not documented Mandatory
and `omitempty` requires a pointer to take effect on a struct field.

`CreateVA` and `UpdateVA` mint or replace VA configuration; `DeleteVA`
removes it. All three, plus `UpdateStatusVA` (changes payment-eligibility
state), are mutating calls and get the package's standard
non-idempotency doc comment. `InquiryVA` is a pure read and gets none,
matching the package's existing inquiry-endpoint precedent.

## Testing (mechanical precedent checks)

Each endpoint gets the standard 5-test core pattern (full-struct
response DeepEqual, request wire round-trip, non-2xx-responseCode,
non-2xx-status-with-2xx-body, 2xx-status-with-no-responseCode) plus one
`MandatoryFieldsAlwaysSerialized` test asserting every mandatory
no-`omitempty` field's zero-value shape. `UpdateVA`'s and `DeleteVA`'s
tests additionally assert `hb.Method` reaches the wire request as PUT
and DELETE respectively (via the test server's `r.Method`), since this
is the package's first phase where a binding overrides the caller's
configured method. `CreateVA` has no mandatory-field test beyond the
two string fields (`VirtualAccountName`, `TrxID`) since the identity
triple is Optional here, unlike every other VA endpoint.
