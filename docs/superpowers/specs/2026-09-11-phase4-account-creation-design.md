# go-snap-bi: Phase 4 — Account Creation (Registrasi)

Status: draft, following the pattern established by Phase 2/3. Scope: one
endpoint — API Account Creation. Registrasi has 11 sub-endpoints total
(Card Registration + Set Limit + Inquiry + Verify OTP + Unbinding, Account
Creation, Account Binding, Account Binding Inquiry, Account Unbinding, Get
OAuth URL, OTP) — the rest are separate future phases. "Get OAuth URL" is
explicitly excluded from this decomposition: it's a browser-redirect OAuth
authorization-code flow (the merchant constructs a URL and redirects the
user's browser to it), not a signed server-to-server POST like every other
binding in this package — it doesn't fit the `Transport.Do`/`HeaderBuilder`
shape and would need its own design when addressed.

## Source of truth

- Path: `.../{version}/registration-account-creation`
- HTTP Method: POST
- Service Code: 06, Version: 1.0

No worked example was available in the portal's Code Snippets tab for this
endpoint at research time; the request/response shapes below come directly
from the Guides tab's field tables (the same primary source used for every
other binding), not an example capture.

### Request body

Every field is optional (O) per the standard's own table — this endpoint's
entire request body is optional, since it supports several different
onboarding flows (seamless data, OAuth redirect, direct creation) that each
use a different subset of fields.

| Field | Type | Notes |
|---|---|---|
| partnerReferenceNo | string | |
| countryCode | string | ISO country code |
| customerId | string | |
| deviceInfo | object | `{os, osVersion, model, manufacturer}`, all string |
| email | string | |
| lang | string | |
| locale | string | |
| name | string | |
| onboardingPartner | string | |
| phoneNo | string | format `62xxxxxxxxxxxxx` |
| redirectUrl | string | |
| scopes | string | |
| seamlessData | string | URL-encoded |
| seamlessSign | string | URL-encoded signature of seamlessData |
| state | string | |
| merchantId | string | |
| subMerchantId | string | |
| terminalType | object | modeled as `json.RawMessage`: the standard's own table types it "Object" but gives no field breakdown |
| additionalInfo | object | |

### Response body

| Field | Type | Mandatory | Notes |
|---|---|---|---|
| responseCode | string | M | |
| responseMessage | string | M | |
| referenceNo | string | C | must be filled on success |
| partnerReferenceNo | string | O | |
| authCode | string | O | |
| apiKey | string | O | standard types this "Numeric" but gives no length/format constraint; modeled as `string` to avoid a false precision/range assumption, consistent with `PageSize`/`PageNumber`'s established precedent (Phase 3) of trusting the wire shape over a table's type label when there's ambiguity |
| accountId | string | O | |
| state | string | O | |
| additionalInfo | object | O | |

## Design

- New file `account_creation.go` in package `snap`.
- Types: `DeviceInfo` (new shared-shape type: `OS`, `OSVersion`, `Model`,
  `Manufacturer`, all string), `AccountCreationRequest`,
  `AccountCreationResponse`.
- One function: `AccountCreation(ctx context.Context, t *Transport, hb HeaderBuilder, req AccountCreationRequest) (AccountCreationResponse, error)`.
  Identical shape to `BalanceInquiry`/`TransactionHistoryList`: marshal
  once, set `hb.Body`, call `t.Do`, `checkResponseStatus`, decode, reject
  empty `responseCode`.

## Testing

Same pattern as Phase 2/3, adapted for the missing worked example:
- A response test using a hand-built fixture (not a captured worked
  example, since none exists) covering every response field via
  `reflect.DeepEqual` on the full struct, including a populated
  `AdditionalInfo`.
- A request wire-body test decoding into `map[string]any`, covering a
  representative subset of fields including the nested `deviceInfo` object,
  not the full 18-field list (matching Phase 2/3's precedent of not
  asserting every single optional field on the request side).
- Non-2xx `responseCode` test.
- Binding-level HTTP-200-with-no-responseCode test (per the Phase 3
  santa-loop precedent — this is now a required test for every binding,
  not something to skip).
