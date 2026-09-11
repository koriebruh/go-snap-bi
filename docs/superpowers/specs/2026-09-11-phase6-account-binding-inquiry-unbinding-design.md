# go-snap-bi: Phase 6 — Account Binding Inquiry & Account Unbinding (Registrasi)

Status: draft, same pattern as Phase 2-5. Scope: two endpoints, the remaining
two legs of the Account Binding trio (bind → inquire → unbind) that Phase 5
started. Both flat responses, no accessTokenInfo/userInfo nesting like
Account Binding (07) has.

## Endpoint 1: API Account Binding Inquiry (Service Code 08)

Path `.../{version}/registration-account-inquiry` (note: "binding" is
dropped from the URL segment itself — confirmed against the portal, not a
typo). POST.

### Request body

| Field | Guides tab type | Mandatory | Go type |
|---|---|---|---|
| partnerReferenceNo | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` (thin spec, no field breakdown) |

### Response body

| Field | Guides tab type | Mandatory | Go type | Notes |
|---|---|---|---|---|
| responseCode | String | M | `string` | |
| responseMessage | String | M | `string` | |
| referenceNo | String | C | `string` | |
| partnerReferenceNo | String | O | `string` | |
| accountCurrency | String | O | `string` | |
| accountName | String | O | `string` | |
| accountNo | String | O | `string` | |
| accountTransactionLimit | **Numeric** | O | **`json.RawMessage`** | Same fix as Phase 4's `APIKey`, same reasoning: Guides tab types this Numeric, and while this portal's one worked example renders it as a quoted wire value (`"accountTransactionLimit":"1000000"`), SNAP is a multi-PJP standard — one example from one PJP is thin evidence about what every issuer sends. A plain `string` field would fail the entire decode (discarding `ResponseCode`/`AccountNo`/`AccountName`/etc. too) if some other issuer sends it unquoted, for a field this package has no way to retry around. `json.RawMessage` tolerates either shape. |
| endDatePeriod | String (YYYY-MM-DD) | O | `string` | |
| startDatePeriod | String (YYYY-MM-DD) | O | `string` | |
| additionalInfo | Object | O | `json.RawMessage` | |

Worked example confirms `responseCode` prefix `2000800` (HTTP 200 + service
08 + case 00), which is how the Service Code was confirmed (the portal page
groups this under "Account Binding" without restating the code plainly).

## Endpoint 2: API Account Unbinding (Service Code 09)

Path `.../{version}/registration-account-unbinding`. POST.

### Request body

| Field | Guides tab type | Mandatory | Go type |
|---|---|---|---|
| partnerReferenceNo | String | O | `string` |
| linkId | String | O | `string` |
| merchantId | String | **M** | `string` |
| subMerchantId | String | O | `string` |
| tokenId | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

Note on the M/O split: `merchantId` is the only Mandatory field; `linkId`
and `tokenId` — the fields that would actually identify *which* binding to
remove — are both Optional per the Guides tab, and the standard doesn't
explain server-side resolution when only `merchantId` is given. Following
the package's established stance (`balance_inquiry.go`'s precedent: this
package does not enforce business rules beyond the wire shape, the server
validates and rejects), `AccountUnbindingRequest` does not add a Go-level
check requiring `linkId` or `tokenId` — this is documented here so a future
maintainer doesn't miss it, not treated as a Phase 6 defect to fix in Go.

### Response body

| Field | Guides tab type | Mandatory | Go type | Notes |
|---|---|---|---|---|
| responseCode | String | M | `string` | |
| responseMessage | String | M | `string` | |
| referenceNo | String | O | `string` | |
| partnerReferenceNo | String | O | `string` | |
| merchantId | String | O | `string` | |
| subMerchantId | String | O | `string` | |
| linkId | String | O | `string` | |
| unlinkResult | String | O | `string` | Sample value `"success"` reads status-like, but the Guides tab lists no enum — plain `string`, not a bool or typed enum. |
| additionalInfo | Object | O | `json.RawMessage` | |

Worked example confirms `responseCode` prefix `2000900`, confirming
Service Code 09.

## Design

Two new files, each following the Phase 2-5 shape exactly (marshal once
into `hb.Body`, `t.Do`, `checkResponseStatus`, decode, reject empty
`responseCode`):

- `account_binding_inquiry.go`: `AccountBindingInquiryRequest`,
  `AccountBindingInquiryResponse`, `AccountBindingInquiry(ctx, t, hb, req)`.
- `account_unbinding.go`: `AccountUnbindingRequest`,
  `AccountUnbindingResponse`, `AccountUnbinding(ctx, t, hb, req)`.

Neither response needs nested `Binding*`-prefixed structs — both are flat,
unlike Account Binding (07)'s `accessTokenInfo`/`userInfo`. No naming
collision risk, so no prefix is needed on any field or type here.

## Testing

6 tests per endpoint (12 total) — the five core tests Phase 2 established
(full-struct response DeepEqual, request wire round-trip,
non-2xx-responseCode, non-2xx-status-with-2xx-body,
2xx-status-with-no-responseCode) plus one 6th test per endpoint pinning its
own risk, the same way Phase 4 and Phase 5 each added a 6th test for their
respective ambiguous/mandatory field:

- Account Binding Inquiry: a table-driven wire-shape test for
  `accountTransactionLimit` (quoted string, unquoted number, JSON null),
  identical in shape to Phase 4's `TestAccountCreation_APIKeyAcceptsEitherWireShape`,
  confirming `json.RawMessage` tolerates every shape without losing the
  rest of the response.
- Account Unbinding: a `MerchantID`-always-serialized test, identical in
  shape to Phase 5's `TestAccountBinding_MerchantIDAlwaysSerialized`, since
  `MerchantID` is likewise the one request field without `omitempty`.
