# go-snap-bi: Phase 9 — Verify OTP, Card Registration Unbinding, API OTP (Registrasi)

Status: implemented. Scope: three POST endpoints, batched together since
none needs shared-code changes or a new request/response shape — all
follow the Phase 2-7 marshal/hb.Body/checkResponseStatus/decode pattern
exactly. This closes out the Registrasi group except "API Get OAuth URL"
(Service Code 10), which remains explicitly out of scope for this
package's binding shape.

## Ambiguous-type rule applied (same rule as Phase 7)

Can the documented type have a non-string JSON representation? None of
this phase's fields can: `unsubscribeDate`/`trxDateTime` are labeled
"Datetime"/"Date", and JSON has no datetime primitive, so both are
`string`, unambiguous. No field in this phase is labeled "Numeric" or
"decimal".

## Endpoint 1: API Verify OTP (Direct Integration) (Service Code 04)

Path `.../{version}/otp-verification`. POST. Confirmed via worked
`responseCode` `2000400`.

### Request body (all Optional per Guides tab)

| Field | Guides tab type | Go type |
|---|---|---|
| originalPartnerReferenceNo | String | `string` |
| originalReferenceNo | String | `string` |
| action | String | `string` |
| merchantId | String | `string` |
| otp | String | `string` |
| chargeToken | String | `string` |
| type | String | `string` |
| additionalInfo | Object | `json.RawMessage` |

### Response body

| Field | Guides tab type | M/O | Go type | Notes |
|---|---|---|---|---|
| responseCode | String | M | `string` | |
| responseMessage | String | M | `string` | |
| originalReferenceNo | String | O | `string` | |
| originalPartnerReferenceNo | String | O | `string` | |
| accountNo | String | O | `string` | |
| bankCardToken | String | O | `string` | |
| cardPan | String | O | `string` | |
| customerId | String | O | `string` | |
| email | String | O | `string` | |
| expiredDatetime | String | O | `string` | |
| expiryDate | String | O | `string` | MMYY |
| identificationNo | String | O | `string` | |
| linkageToken | String | O | `string` | |
| phoneNo | String | O | `string` | |
| qParamsURL | String | O | `string` | |
| qParams | Object | O | `json.RawMessage` | |
| sendOtpFlag | String | O | `string` | |
| subscribeDatetime | String | O | `string` | |
| tokenExpiryTime | String | O | `string` | |
| transactionTimestamp | String | O | `string` | Source artifact, not reconciled: the Guides tab's description text for this field ("Random String to generate validation for webview") doesn't match its name, and the worked example holds a random-string-looking value (`"g4BoEz43jfjVvAvN"`), not an actual timestamp. Recorded here per the package's convention of noting unreconciled source discrepancies rather than guessing which is right. |
| additionalInfo | Object | O | `json.RawMessage` | |

## Endpoint 2: API Card Registration Unbinding (Service Code 05)

Path `.../{version}/registration-card-unbind`. POST. Confirmed via
worked `responseCode` `2000500`.

### Request body

| Field | Guides tab type | M/O | Go type | Notes |
|---|---|---|---|---|
| partnerReferenceNo | String | O | `string` | |
| token | String | **M** | `string` | |
| bankCardNo | String | O | `string` | |
| type | String | O | `string` | |
| part | String | O | `string` | Source artifact, not reconciled: the Guides tab's description text for `part` is identical to `merchantId`'s description on the portal, and the worked example sets both fields to the same value. Kept as its own field since the Guides tab lists it as a distinct field name. |
| merchantId | String | O | `string` | |
| subMerchantId | String | O | `string` | |
| terminalId | String | O | `string` | |
| tokenRequestorId | String | O | `string` | |
| journeyId | String | O | `string` | |
| transactionDate | String | O | `string` | ISO 8601 |
| additionalInfo | Object | O | `json.RawMessage` | |

### Response body

| Field | Guides tab type | M/O | Go type | Notes |
|---|---|---|---|---|
| responseCode | String | M | `string` | |
| responseMessage | String | M | `string` | |
| referenceNo | String | O | `string` | |
| partnerReferenceNo | String | O | `string` | |
| message | String | O | `string` | |
| customerId | String | O | `string` | |
| unsubscribeDate | **Datetime** | O | `string` | Only field in the whole Registrasi group labeled "Datetime" (every other date-like field is labeled "String") — JSON has no datetime primitive, and the worked example is a plain quoted ISO-8601 string, so `string` regardless of the unique label. |
| additionalInfo | Object | O | `json.RawMessage` | |

## Endpoint 3: API OTP (Service Code 81)

Path `.../v1.0/otp` (note: not versioned via `{version}` template like
every other endpoint in this package — the portal's own path is the
literal `v1.0/otp`). POST. Confirmed via worked `responseCode` `2008100`
— not sequential with Registrasi's other single/double-digit service
codes, confirmed only from the worked example, not from a stated code in
the Guides tab.

This is the package's first endpoint whose own worked example includes
an `Authorization-Customer` header (absent from every other Registrasi
endpoint) — i.e. a B2B2C-shaped call. This package doesn't change: a
caller sets `HeaderBuilder.B2B2C = true` (and the customer token/device
ID it requires) the same way as for any other B2B2C call; `OTP` itself
takes no special parameter for this.

### Request body

| Field | Guides tab type | M/O/C | Go type | Notes |
|---|---|---|---|---|
| partnerReferenceNo | String | O | `string` | |
| journeyId | String | **M** | `string` | |
| merchantId | String | O | `string` | |
| subMerchant | String | O | `string` | Field name is `subMerchant`, not `subMerchantId` as in every other endpoint in this package — the standard's own inconsistency, kept as-is rather than "corrected" to match sibling endpoints. |
| externalStoreId | String | O | `string` | |
| trxDateTime | **Date** | O | `string` | Unique type label ("Date", not "String") in this whole document; JSON has no date primitive, and the worked example is a plain quoted ISO-8601 string, so `string` regardless. |
| bankCardToken | String | **C** | `string` (`omitempty`) | Only Conditional-marked field found across the whole researched Registrasi group. This package's established convention (e.g. `referenceNo` C fields elsewhere) is `omitempty` for Conditional the same as Optional — the field's presence, not the tag, carries the conditional-ness. |
| otpTrxCode | String | O | `string` | |
| otpReasonCode | String | O | `string` | |
| otpReasonMessage | String | O | `string` | |
| additionalInfo | Object | O | `json.RawMessage` | |

### Response body

| Field | Guides tab type | M/O | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |
| referenceNo | String | O | `string` |
| partnerReferenceNo | String | O | `string` |
| chargeToken | String | **M** | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

## Design

Three new files, each following the Phase 2-7 shape exactly:

- `verify_otp.go`: `VerifyOTPRequest`, `VerifyOTPResponse`, `VerifyOTP(ctx, t, hb, req)`.
- `card_registration_unbinding.go`: `CardRegistrationUnbindingRequest`,
  `CardRegistrationUnbindingResponse`, `CardRegistrationUnbinding(ctx, t, hb, req)`.
- `otp.go`: `OTPRequest`, `OTPResponse`, `OTP(ctx, t, hb, req)`.

Idempotency notes, corrected from an earlier draft of this doc (a
go-review finding): the original reasoning here — that none of the
three mints a new long-lived resource, so none needs the note — doesn't
survive `AccountUnbinding`'s own precedent, which carries the note
despite also minting nothing. The note is about server-side duplicate
detection on a mutating call, not specifically about minting.
`CardRegistrationUnbinding` gets the note, matching its exact structural
sibling `AccountUnbinding`. `OTP` gets the note too: it triggers a real
external side effect (an OTP delivery), so a retry under a fresh
X-EXTERNAL-ID risks a duplicate delivery. `VerifyOTP` does not: it
neither mints a resource nor triggers an external side effect, only
checks an already-issued OTP.

## Testing

`VerifyOTP` has no mandatory request/response fields besides
`responseCode`/`responseMessage` (already covered by every core test),
so it gets 5 tests: the core Phase 2 pattern (full-struct response
DeepEqual, request wire round-trip, non-2xx-responseCode,
non-2xx-status-with-2xx-body, 2xx-status-with-no-responseCode).

`CardRegistrationUnbinding` and `OTP` each have exactly one mandatory
request field without `omitempty` (`Token`, `JourneyID`), matching the
pattern Phase 5/6 pinned with a dedicated
`TestAccountBinding_MerchantIDAlwaysSerialized`-style test — so each
gets a 6th test asserting that field is always present on the wire, even
as `""`. 5 + 6 + 6 = 17 tests total.
