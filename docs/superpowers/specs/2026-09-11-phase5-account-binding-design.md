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
typed as a proper nested struct `SuccessParams`, unlike `additionalData`
which has no field breakdown at all), additionalInfo.

## Response body

responseCode (M), responseMessage (M), referenceNo (C), partnerReferenceNo,
accountToken, accessTokenInfo (object: `accessToken`, `expiresIn`,
`refreshToken`, `reExpiresIn`, `tokenStatus` — typed nested struct
`AccessTokenInfo`), linkId, nextAction, linkageToken, params (object, thin
spec — `json.RawMessage`), pinWebViewUrl, redirectToDeeplink, redirectUrl,
userInfo (object: `publicUserId` — typed nested struct `UserInfo`),
additionalInfo.

## Design

New file `account_binding.go`. Types: `SuccessParams`, `AccessTokenInfo`,
`UserInfo`, `AccountBindingRequest`, `AccountBindingResponse`. One function
`AccountBinding(ctx, t, hb, req) (AccountBindingResponse, error)`, identical
shape to prior bindings (marshal once, `hb.Body`, `t.Do`,
`checkResponseStatus`, decode, reject empty `responseCode`).

## Testing

5 tests, not Phase 4's 6: same standing-requirement pattern (full-struct
DeepEqual with populated AdditionalInfo, wire-body round-trip,
non-2xx-responseCode, non-2xx-status-with-2xx-body,
2xx-status-with-no-responseCode) plus nested struct coverage for
`BindingAccessTokenInfo`/`BindingUserInfo`/`BindingSuccessParams` in the
response test. No 6th test: Phase 4's extra test was `APIKey`-specific
(an ambiguous-wire-type regression); Account Binding has no comparably
ambiguous scalar field.

## Naming note

`BindingSuccessParams`/`BindingAccessTokenInfo`/`BindingUserInfo` are
prefixed `Binding` (go-review finding): Account Binding Inquiry and Account
Unbinding (future phases, same service group) will plausibly have their
own differently-shaped `successParams`/`userInfo` objects, and an
unprefixed name would collide in this flat package.
