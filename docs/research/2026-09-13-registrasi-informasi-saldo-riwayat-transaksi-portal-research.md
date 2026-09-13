# Registrasi, Informasi Saldo, Riwayat Transaksi — ASPI SNAP Developer Portal Research

Source: https://apidevportal.aspi-indonesia.or.id/api-services/registrasi,
.../api-services/informasi-saldo, .../api-services/riwayat-transaksi.
Retrieved: 2026-09-13, via WebFetch (rendered page text extraction).

This document exists to close a gap flagged by a pre-publish
spec-fidelity audit: the `registration`, `balanceinfo`, and
`transactionhistory` packages (Service Codes 01-14, 81 — the
Registrasi, Informasi Saldo, and Riwayat Transaksi portal categories)
were implemented in early sessions before this project adopted the
practice of saving a research document per portal category, so their
field tagging had no in-repo primary source to verify against. This
document is that source, retrieved after the fact and checked
line-by-line against the actual `.go` source (not asserted from
memory) before any "matches" claim below was written.

## 1. Full Scope: 3 Categories, 15 Endpoints

| # | Category | Endpoint | Svc Code | Method | Path (`.../{version}/...`) |
|---|---|---|---|---|---|
|1| Registrasi | Card Registration | 01 | POST | `registration-card-bind` |
|2| Registrasi | Card Registration – Set Limit | 02 | POST | `registration/card-bind-limit` |
|3| Registrasi | Card Registration Inquiry | 03 | GET | `registration-card-inquiry/{custIdMerchant}` (path param) |
|4| Registrasi | Verify OTP (Direct Integration) | 04 | POST | `otp-verification` |
|5| Registrasi | Card Registration Unbinding | 05 | POST | `registration-card-unbind` |
|6| Registrasi | Account Creation | 06 | POST | `registration-account-creation` |
|7| Registrasi | Account Binding | 07 | POST | `registration-account-binding` |
|8| Registrasi | Account Binding Inquiry | 08 | POST | `registration-account-inquiry` |
|9| Registrasi | Account Unbinding | 09 | POST | `registration-account-unbinding` |
|10| Registrasi | Get OAuth URL | 10 | GET | `get-auth-code` (query string; already implemented against a live fetch, not re-verified here) |
|11| Registrasi | OTP | 81 | POST | `otp` |
|12| Informasi Saldo | Balance Inquiry | 11 | POST | `balance-inquiry` |
|13| Riwayat Transaksi | Transaction History List | 12 | POST | `transaction-history-list` |
|14| Riwayat Transaksi | Transaction History Detail | 13 | POST | `transaction-history-detail` (already implemented against a live fetch, not re-verified here) |
|15| Riwayat Transaksi | Bank Statement | 14 | POST | `bank-statement` (already implemented against a live fetch, not re-verified here) |

## 2. Per-Endpoint Field Tables and Verification Outcome

### 2.1 Card Registration (01)

Req: `partnerReferenceNo O`, `accountName O`, `cardData M` (Encrypted
Object), `bankAccountNo O`, `bankCardNo M` (String(19)), `bankCardType
O`, `dateOfBirth O`, `email O`, `expiredDatetime O`, `expiryDate O`,
`identificationNo O`, `identificationType O`, `custIdMerchant M`
(String(18)), `isBindAndPay O`, `merchantId O`, `terminalId O`,
`journeyId O`, `subMerchantId O`, `externalStoreId O`, `limit O`
(decimal(17,2)), `merchantLogoUrl O`, `phoneNo O`, `sendOtpFlag O`,
`type O`, `additionalInfo O`.

Resp: `referenceNo O`, `partnerReferenceNo O`, `bankCardToken M`
(String(128)), `chargeToken O`, `randomString O`, `tokenExpiryTime O`,
`additionalInfo O`.

**Verification outcome: 1 discrepancy found and fixed.**
`registration/card_registration.go`'s `CardRegistrationRequest.CardData`
was tagged `json:"cardData,omitempty"` and the type's doc comment
listed only `BankCardNo`/`CustIDMerchant` as mandatory — but this
table marks `cardData` Mandatory too. `omitempty` on a Mandatory
`json.RawMessage` field means a caller who leaves it unset (or a
caller who genuinely has no card data to send, which shouldn't be
possible for this endpoint but the type didn't prevent it) silently
drops the field from the wire body instead of sending it as an
explicit `null` the server can reject with a clear error. Fixed:
`omitempty` removed from the tag, doc comment corrected to list all
three Mandatory fields, and a regression test added
(`TestCardRegistration_MandatoryFieldsAlwaysSerialized` now also
checks `cardData` is present as `null` in a zero-value request).

### 2.2 Card Registration – Set Limit (02)

Req: `partnerReferenceNo O`, `bankAccountNo O`, `bankCardNo O`, `limit
O` (decimal(17,2)), `bankCardToken M` (String(128)), `otp O`,
`additionalInfo O`.

Resp: `referenceNo O`, `partnerReferenceNo O`, `additionalInfo O` — no
Mandatory field beyond the envelope.

**Verification outcome: matches.** `registration/card_registration_set_limit.go`
tags `BankCardToken` with no `omitempty` and every other field with
`omitempty`, exactly matching this table.

### 2.3 Card Registration Inquiry (03)

GET, `custIdMerchant` as a path parameter (String(18), Mandatory).

Resp: `accountList` (array, unmarked cardinality), each entry
wrapping `accountData` (`accountId O`, `createdDate O` (String(26)),
`credentialNo O`, `credentialType O`, `maxLimit O` (String(6)),
`status O`), `additionalInfo O` — no Mandatory field beyond the
envelope.

**Verification outcome: matches.** `registration/card_registration_inquiry.go`
takes `custIDMerchant` as a plain `string` parameter (not a JSON body
field) and validates it via an allowlist regex before joining it onto
the URL path — consistent with this table's path-parameter modeling.
`CardRegistrationInquiryAccountData`'s fields are all tagged
`omitempty`, matching every field being Optional here. The
`accountData` wrapper nesting matches the table's own extra nesting
level.

### 2.4 Verify OTP — Direct Integration (04)

Req: `originalPartnerReferenceNo O`, `originalReferenceNo O`, `action
O`, `merchantId O`, `otp O`, `chargeToken O`, `type O`,
`additionalInfo O` — no Mandatory field.

Resp: `originalReferenceNo O`, `originalPartnerReferenceNo O`,
`accountNo O`, `bankCardToken O`, `cardPan O`, `customerId O`, `email
O`, `expiredDatetime O`, `expiryDate O`, `identificationNo O`,
`linkageToken O`, `phoneNo O`, `qParamsURL O`, `qParams O` (Object —
untyped), `action O`, `sendOtpFlag O`, `subscribeDatetime O`,
`tokenExpiryTime O`, `transactionTimestamp O`, `additionalInfo O` — no
Mandatory field beyond the envelope.

**Verification outcome: matches.** `registration/verify_otp.go` tags
every field `omitempty` on both request and response, matching every
field being Optional here, including `QParams` as `json.RawMessage`
for the untyped Object field.

### 2.5 Card Registration Unbinding (05)

Req: `partnerReferenceNo O`, `token M` (String(128)), `bankCardNo O`,
`type O`, `part O`, `merchantId O`, `subMerchantId O`, `terminalId O`,
`tokenRequestorId O`, `journeyId O`, `transactionDate O`,
`additionalInfo O`.

Resp: `referenceNo O`, `partnerReferenceNo O`, `message O`, `customerId
O`, `unsubscribeDate O` (Datetime), `additionalInfo O` — no Mandatory
field beyond the envelope.

**Verification outcome: matches.** `registration/card_registration_unbinding.go`
tags `Token` with no `omitempty`, every other field with `omitempty`.

### 2.6 Account Creation (06)

Req: `partnerReferenceNo O`, `countryCode O`, `customerId O`,
`deviceInfo O` (Object: `os O`, `osVersion O`, `model O`, `manufacturer
O`), `email O`, `lang O`, `locale O`, `name O`, `onboardingPartner O`,
`phoneNo O`, `redirectUrl O` (String(2048)), `scopes O`, `seamlessData
O`, `seamlessSign O`, `state O`, `merchantId O`, `subMerchantId O`,
`terminalType O` (documented "Object", length 32 — an
internally-inconsistent row in the portal's own table, recorded as
shown), `additionalInfo O` — no Mandatory field on the request.

Resp: `referenceNo C` (success only), `partnerReferenceNo O`,
`authCode O`, `apiKey O` (Numeric — ambiguous-type), `accountId O`,
`state O`, `additionalInfo O`.

**Verification outcome: matches.** `registration/account_creation.go`
tags every request field `omitempty` (all Optional), `ReferenceNo`
with `omitempty` (Conditional → omitempty, correct package convention),
and models `APIKey` as `json.RawMessage` for the ambiguous Numeric
type — consistent with this package's established precedent for other
Numeric-typed fields elsewhere.

### 2.7 Account Binding (07)

Req: `partnerReferenceNo O`, `action O`, `additionalData O` (Object:
`userId O`, `email O`, `postalAddress O`), `authCode O`, `grantType
O`, `isBindAndPay O`, `lang O`, `locale O`, `merchantId M`
(String(64)), `subMerchantId O`, `msisdn O`, `otp O`, `phoneNo O`,
`platformType O`, `redirectUrl O`, `referenceId O`, `refreshToken O`,
`successParams O` (Object: `accountId O`, plus `terminalId`/
`tokenRequestorId` per the Guides tab, not separately broken out by
this fetch), `terminalId O`, `tokenRequestorId O`, `additionalInfo O`.

Resp: `referenceNo C` (success only), `partnerReferenceNo O`,
`accountToken O`, `accessTokenInfo O` (Object: `accessToken O`,
`expiresIn O`, `refreshToken O`, `reExpiresIn O`, `tokenStatus O`),
`linkId O`, `nextAction O`, `linkageToken O`, `params O` (Object:
`action O`), `pinWebViewUrl O`, `redirectToDeeplink O`, `redirectUrl
O`, `userInfo O` (Object: `publicUserId O`), `additionalInfo O`.

**Verification outcome: matches.** `registration/account_binding.go`
tags `MerchantID` with no `omitempty` — the sole Mandatory request
field — and every other field `omitempty`. The nested
`BindingSuccessParams`/`BindingAccessTokenInfo`/`BindingUserInfo`
types and `AdditionalData`/`Params` as `json.RawMessage` all match
this table's shape.

### 2.8 Account Binding Inquiry (08)

Req: `partnerReferenceNo O`, `additionalInfo O` — no Mandatory field.

Resp: `referenceNo C` (success only), `partnerReferenceNo O`,
`accountCurrency O`, `accountName O`, `accountNo O`,
`accountTransactionLimit O` (Numeric(19,2) — ambiguous-type),
`endDatePeriod O`, `startDatePeriod O`, `additionalInfo O`.

**Verification outcome: matches.** `registration/account_binding_inquiry.go`
tags every request field `omitempty`, `ReferenceNo` with `omitempty`
(Conditional), and models `AccountTransactionLimit` as
`json.RawMessage` for the ambiguous Numeric type.

### 2.9 Account Unbinding (09)

Req: `partnerReferenceNo O`, `linkId O`, `merchantId M` (String(64)),
`subMerchantId O`, `tokenId O`, `additionalInfo O`.

Resp: `referenceNo O`, `partnerReferenceNo O`, `merchantId O`,
`subMerchantId O`, `linkId O`, `unlinkResult O`, `additionalInfo O` —
no Mandatory field beyond the envelope.

**Verification outcome: matches.** `registration/account_unbinding.go`
tags `MerchantID` with no `omitempty` — the sole Mandatory request
field — and every other field `omitempty` on both request and
response.

### 2.10 Get OAuth URL (10)

Already implemented and verified against a direct portal fetch in an
earlier phase — see `registration/get_oauth_url.go`'s own doc
comments. Not re-verified here.

### 2.11 OTP (81)

Req: `partnerReferenceNo O`, `journeyId M` (String(32), "On the first
request of the journey, this must be equal to the X-EXTERNAL-ID"),
`merchantId O`, `subMerchant O` (note: singular "subMerchant", not
"subMerchantId" — an inconsistency with every other endpoint's field
name in this document), `externalStoreId O`, `trxDateTime O` (Date,
String(25)), `bankCardToken C` (String(128)), `otpTrxCode O`,
`otpReasonCode O`, `otpReasonMessage O`, `additionalInfo O`.

Resp: `referenceNo O`, `partnerReferenceNo O`, `chargeToken M`
(String(40)), `additionalInfo O`.

**Verification outcome: matches, including the naming
inconsistency.** `registration/otp.go` already models the field as
`SubMerchant string \`json:"subMerchant,omitempty"\`` (not
`SubMerchantID`), with its own doc comment recording the standard's
inconsistency explicitly rather than silently normalizing it —
consistent with this table and with this project's established
"record as shown" convention. `JourneyID` and `ChargeToken` both carry
no `omitempty` (the two Mandatory fields), and `BankCardToken` carries
`omitempty` (Conditional → omitempty, correct).

### 2.12 Balance Inquiry (11)

Req: `partnerReferenceNo O`, `bankCardToken C` (String(128), XOR with
`accountNo` and a B2B2C customer token), `accountNo C` (String(16),
same condition), `balanceTypes O` (Array of String), `additionalInfo
O` — no Mandatory field.

Resp: `referenceNo O`, `partnerReferenceNo O`, `accountNo O`, `name
O`, `accountInfos` (array, unmarked cardinality): each entry
`balanceType O`, `amount O` (members M), `floatAmount O` (members M),
`holdAmount O` (members M), `availableBalance O` (members M),
`ledgerBalance O` (members M), `currentMultilateralLimit O` (members
M), `registrationStatusCode O`, `status O`; `additionalInfo O` — no
Mandatory field beyond the envelope.

**Verification outcome: 1 discrepancy found and fixed, the most
significant one in this document.**
`balanceinfo/balance_inquiry.go`'s `AccountInfo` type had all nine of
its own fields (`BalanceType`, all six Money-family containers,
`RegistrationStatusCode`, `Status`) tagged with no `omitempty` — as if
every one of them were Mandatory — and all six Money-family fields
were plain `snap.Money`, not `*snap.Money`. This table marks every one
of those nine fields Optional (`O`) at their own row; only the
`value`/`currency` members nested inside each Money object are
Mandatory. Per this package's established rule (an Optional container
→ `*snap.Money`, applied consistently across all 78 other endpoints in
this module), this was a genuine bug, not a judgment call — likely
predating that rule's establishment, since `BalanceInquiry` (Phase 2)
was one of the very first endpoints implemented. Fixed: all nine
fields now carry `omitempty`, and the six Money-family fields are now
`*snap.Money`. This is a breaking change to `AccountInfo`'s public
shape, made before the first tag.

### 2.13 Transaction History List (12)

Req: `partnerReferenceNo O`, `fromDateTime O`, `toDateTime O`,
`pageSize O` (documented Integer(2), "Default: 10"), `pageNumber O`
(documented Integer(2), "Default: 0"), `additionalInfo O` — no
Mandatory field.

Resp: `referenceNo O`, `partnerReferenceNo O`, `detailData` (array,
unmarked cardinality): each entry `dateTime O`, `amount O` (members
M), `remark O`, `sourceOfFunds O` (`List<SourceOfFund>`), `status M`,
`type M`, `additionalInfo O`; `additionalInfo O` — no Mandatory field
beyond the envelope.

**Verification outcome: 1 discrepancy found and fixed.**
`transactionhistory/transaction_history.go`'s `TransactionDetail.Amount`
was a plain `snap.Money` with no `omitempty`, but this table marks the
`amount` container Optional (only its members are Mandatory) — the
same class of bug as Balance Inquiry's `AccountInfo`, on a single
field this time. Fixed: `Amount` is now `*snap.Money` with
`omitempty`. Note this is genuinely different from the
otherwise-similar Transaction History **Detail** (Service Code 13)
endpoint, where the equivalent `amount` container is documented
Mandatory — the two endpoints are not harmonized, each modeled per its
own table. `Status` and `Type` (both Mandatory, no `omitempty`) and
every other field on `TransactionDetail` already matched this table
and needed no change; `SourceOfFund`'s own internal `Amount` field
(plain `snap.Money`) was not touched, since neither this fetch nor the
earlier Transaction History Detail fetch gives a field-level
breakdown of `SourceOfFund` itself to check it against — recorded here
as an open item, not silently assumed correct.

## 3. Summary

Of 12 endpoints re-verified against a fresh, direct portal fetch (the
3 already built against live data this session — Get OAuth URL,
Transaction History Detail, Bank Statement — were not re-checked),
**3 genuine field-tagging discrepancies were found and fixed**: Card
Registration's `CardData` missing its Mandatory tag, Balance Inquiry's
`AccountInfo` (all 9 fields, 6 of them needing the Optional-container
pointer treatment), and Transaction History List's `TransactionDetail.Amount`
needing the same pointer treatment. All three were pre-existing bugs
from before this package's field-modeling conventions were codified,
not judgment calls — none were previously documented as intentional.
`go build`/`go vet`/`go test` all pass after the fixes; the two
Money-pointer fixes are breaking changes made before the first tag.
