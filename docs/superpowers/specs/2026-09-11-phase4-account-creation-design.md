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
| apiKey | json.RawMessage | O | standard types this "Numeric" with no worked example to confirm the wire shape. **Phase 3's `PageSize`/`PageNumber`-as-string precedent does NOT apply here** — that precedent rests on an observed worked example showing quoted strings; this endpoint has none. Since a fixed type (string or numeric) dominates in the wrong direction on the response side — a plain `string` field hard-fails the *entire* decode if the server sends an unquoted number, discarding `referenceNo`/`accountId`/`authCode` too, on a non-idempotent create where the account may already exist — `apiKey` is modeled as `json.RawMessage` to tolerate either shape losslessly (santa-loop round-1 finding, both reviewers independently) |
| accountId | string | O | |
| state | string | O | opaque CSRF-protection nonce echoed from the request (this is an OAuth-style flow); the caller, not this package, is responsible for comparing it against the value it sent |
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

Same pattern as Phase 2/3, adapted for the missing worked example, plus two
standing requirements every binding now carries (established across Phase
2/3's santa-loop rounds, not just this one):
- A response test using a hand-built fixture (not a captured worked
  example, since none exists) covering every response field via
  `reflect.DeepEqual` on the full struct, including a populated
  `AdditionalInfo`.
- A request wire-body test decoding into `map[string]any`, covering a
  representative subset of fields including the nested `deviceInfo` object,
  not the full 19-field list (matching Phase 2/3's precedent of not
  asserting every single optional field on the request side).
- Non-2xx `responseCode` test.
- **Standing requirement**: non-2xx HTTP status with a 2xx-shaped body must
  error (Phase 2 santa-loop round-3 finding — `checkResponseStatus` handles
  this at the shared-helper level, but each binding needs its own
  regression test proving it actually calls that helper).
- **Standing requirement**: HTTP 200 with no `responseCode` field must
  error (Phase 3 santa-loop round-1 finding).
- `APIKey`-specific: a quoted-string, an unquoted-number, and a JSON null
  wire shape (which decodes to a non-nil `RawMessage("null")`, distinct
  from an absent key) must all decode without losing the rest of the
  response (this phase's own santa-loop findings, since `apiKey` is the
  first field in the package with a genuinely ambiguous wire type on the
  response side).
