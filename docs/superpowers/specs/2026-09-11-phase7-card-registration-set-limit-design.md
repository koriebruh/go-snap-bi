# go-snap-bi: Phase 7 — Card Registration & Card Registration Set Limit (Registrasi)

Status: implemented. Scope: two endpoints, both POST, both sharing the
`limit` field.

## Ambiguous-type rule for this phase

One rule, applied to every field below: **can the documented type have a
non-string JSON representation?** If yes, `json.RawMessage` (a bare JSON
number/object is a live possibility the standard's own type label permits).
If no (no JSON primitive matches the label — e.g. a datetime), `string`.

| Field | Endpoint | Guides tab type | Worked example (literal) | JSON type possible? | Go type |
|---|---|---|---|---|---|
| `cardData` | Card Registration | "Encrypted Object" | `"UIdFgZi9BhWx9Scbz/YK+...="` (quoted string) | Yes — "Object" permits a bare `{...}` | `json.RawMessage` |
| `limit` | Card Registration, Set Limit | "decimal" | `"1000000"` (quoted string) | Yes — "decimal" permits a bare number | `json.RawMessage` |

## Endpoint 1: API Card Registration (Service Code 01)

Path `.../{version}/registration-card-bind`. POST. Confirmed via worked
`responseCode` `2000100`.

### Request body

| Field | Guides tab type | M/O | Go type |
|---|---|---|---|
| partnerReferenceNo | String | O | `string` |
| accountName | String | O | `string` |
| cardData | Encrypted Object | O | `json.RawMessage` |
| bankAccountNo | String | O | `string` |
| bankCardNo | String | **M** | `string` |
| bankCardType | String | O | `string` |
| dateOfBirth | String | O | `string` |
| email | String | O | `string` |
| expiredDatetime | String | O | `string` |
| expiryDate | String | O | `string` |
| identificationNo | String | O | `string` |
| identificationType | String | O | `string` |
| custIdMerchant | String | **M** | `string` |
| isBindAndPay | String | O | `string` |
| merchantId | String | O | `string` |
| terminalId | String | O | `string` |
| journeyId | String | O | `string` |
| subMerchantId | String | O | `string` |
| externalStoreId | String | O | `string` |
| limit | decimal | O | `json.RawMessage` |
| merchantLogoUrl | String | O | `string` |
| phoneNo | String | O | `string` |
| sendOtpFlag | String | O | `string` |
| type | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body

| Field | Guides tab type | M/O | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |
| referenceNo | String | O | `string` |
| partnerReferenceNo | String | O | `string` |
| bankCardToken | String | **M** | `string` |
| chargeToken | String | O | `string` |
| randomString | String | O | `string` |
| tokenExpiryTime | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

## Endpoint 2: API Card Registration – Set Limit (Service Code 02)

Path `.../{version}/registration/card-bind-limit` (note the `/` where
sibling endpoints use `-`; this is the standard's own path, not a typo).
POST. Confirmed via worked `responseCode` `2000200`.

Source artifact, not reconciled: the portal's own "Sample Request" line for
this endpoint reads `POST …/v1.0/registration-card-inquiry HTTP/1.2` — a
copy/paste leftover from the adjacent Inquiry section. The declared path in
the endpoint's own "Informasi Umum" table (`.../{version}/registration/card-bind-limit`)
is used here; the sample request/response bodies below are Set Limit's own
content, not Inquiry's, so only the one header line is wrong on the portal.

### Request body

| Field | Guides tab type | M/O | Go type |
|---|---|---|---|
| partnerReferenceNo | String | O | `string` |
| bankAccountNo | String | O | `string` |
| bankCardNo | String | O | `string` |
| limit | decimal | O | `json.RawMessage` |
| bankCardToken | String | **M** | `string` |
| otp | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body

| Field | Guides tab type | M/O | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |
| referenceNo | String | O | `string` |
| partnerReferenceNo | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

## Design

Two new files, same shape as Phase 2-6 (marshal once into `hb.Body`,
`t.Do`, `checkResponseStatus`, decode, reject empty `responseCode`):

- `card_registration.go`: `CardRegistrationRequest`, `CardRegistrationResponse`,
  `CardRegistration(ctx, t, hb, req)`.
- `card_registration_set_limit.go`: `CardRegistrationSetLimitRequest`,
  `CardRegistrationSetLimitResponse`, `CardRegistrationSetLimit(ctx, t, hb, req)`.

Neither endpoint has an idempotency note in the source (searched the whole
Registrasi page text for "idempot*": no matches for any endpoint in this
group). The only related documented behavior is generic, not per-endpoint:
the shared response-code table lists HTTP 409/case 00 ("Cannot use same
X-EXTERNAL-ID in same day") and HTTP 409/case 01 ("Duplicate
partnerReferenceNo"). Neither `CardRegistration` nor
`CardRegistrationSetLimit` carries an X-EXTERNAL-ID retry note, since no
source text singles out these two as non-idempotent any more than any
other POST in the standard.

## Testing

6 tests per endpoint (12 total): the 5 core tests established since
Phase 2, plus one wire-shape test per endpoint for its `json.RawMessage`
field(s) (`cardData`/`limit` for Card Registration, `limit` for Set Limit),
following `TestAccountCreation_APIKeyAcceptsEitherWireShape`'s
table-driven pattern (quoted, unquoted/object, null).
