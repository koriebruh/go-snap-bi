# go-snap-bi: Phase 5 — Account Binding (Registrasi)

Status: draft, same pattern as Phase 2/3/4. Scope: one endpoint — API
Account Binding. Service Code 07, Path `.../{version}/registration-account-binding`,
POST. No worked example available; fields from the Guides tab.

## Request body (all optional except merchantId)

partnerReferenceNo, action, additionalData (object, thin spec — modeled as
`json.RawMessage`), userId, email, postalAddress, authCode, grantType,
isBindAndPay, lang, locale, merchantId (**M**), subMerchantId, msisdn, otp,
phoneNo, platformType, redirectUrl, referenceId, refreshToken,
successParams (object with `accountId`, `terminalId`, `tokenRequestorId` —
typed as a proper nested struct `BindingSuccessParams`, unlike `additionalData`
which has no field breakdown at all), additionalInfo.

### Ambiguous-type field

One request field's name reads as a boolean but the Guides tab types it
string, no worked example given — record the source type, don't silently
guess:

| Field | Guides tab type | Chosen Go type | Rationale |
|---|---|---|---|
| `isBindAndPay` | String | `string` | Guides tab lists type "String", not "Boolean" — likely wire values are the literal strings `"Y"`/`"N"` rather than a JSON boolean. Unlike Phase 4's `APIKey` (a response field, where a wrong type hard-fails decode of the whole response), `isBindAndPay` is request-only: this package only marshals it, so a wrong Go type here means a rejected request, not a decode failure — lower risk, but still worth `string` since that's what the Guides tab says. |

`platformType` is also `string`, but unambiguously — the Guides tab lists
enumerated values (`IOS`/`ANDROID`/`WEB`, not documented exhaustively) with
no boolean-like naming, so it doesn't belong in this table.

## Response body

responseCode (M), responseMessage (M), referenceNo (C), partnerReferenceNo,
accountToken, accessTokenInfo (object: `accessToken`, `expiresIn` and
`reExpiresIn` — Guides tab types these "Datetime of token expiration,
Format: ISO 8601", **not** the `time.Duration` that `Token.ExpiresIn`
(token.go) parses from the seconds-count string the separate B2B/B2B2C
access-token endpoints use; same field name, different endpoint,
different meaning — `refreshToken`,
`tokenStatus` — typed nested struct `BindingAccessTokenInfo`), linkId,
nextAction, linkageToken, params (object, thin spec — `json.RawMessage`),
pinWebViewUrl, redirectToDeeplink, redirectUrl, userInfo (object:
`publicUserId` — typed nested struct `BindingUserInfo`), additionalInfo.

## Design

New file `account_binding.go`. Types: `BindingSuccessParams`, `BindingAccessTokenInfo`,
`BindingUserInfo`, `AccountBindingRequest`, `AccountBindingResponse`. One function
`AccountBinding(ctx, t, hb, req) (AccountBindingResponse, error)`, identical
shape to prior bindings (marshal once, `hb.Body`, `t.Do`,
`checkResponseStatus`, decode, reject empty `responseCode`).

## Testing

6 tests, same standing-requirement pattern as Phase 4 (full-struct
DeepEqual with populated AdditionalInfo, wire-body round-trip,
non-2xx-responseCode, non-2xx-status-with-2xx-body,
2xx-status-with-no-responseCode) plus nested struct coverage for
`BindingAccessTokenInfo`/`BindingUserInfo` in the response test, and
`BindingSuccessParams` (request-side only) in the wire-body round-trip
test. The 6th test pins `MerchantID`'s wire shape: it's the only request
field in the package without `omitempty` on its JSON tag (deliberate,
since it's the one mandatory field), so a regression test asserts
`AccountBindingRequest{}` still marshals `"merchantId":""` rather than
silently dropping the key.

## Naming note

`BindingSuccessParams`/`BindingAccessTokenInfo`/`BindingUserInfo` are
prefixed `Binding` (go-review finding): Account Binding Inquiry and Account
Unbinding (future phases, same service group) will plausibly have their
own differently-shaped `successParams`/`userInfo` objects, and an
unprefixed name would collide in this flat package.
