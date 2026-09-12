# Transfer Debit — ASPI SNAP Developer Portal Research

Source: https://apidevportal.aspi-indonesia.or.id/api-services/transfer-debit
and its 4 sub-group pages.
Retrieved: 2026-09-12, via WebFetch (rendered page text extraction; the
Auth Payment sub-group required a follow-up fetch to recover fields
truncated by summarization on the first pass — noted where relevant).

This is a raw research dump, not implementation-ready prose — future
phases cite it as the source of record for field tables, worked
examples, and confirmed service codes, mirroring
`docs/research/2026-09-11-transfer-kredit-portal-research.md`'s role
for Transfer Kredit. Where the portal itself contradicts across
tables/examples, both readings are recorded rather than resolved here.

## 1. Full Scope: 4 Sub-Groups, 21 Endpoints

Service codes are non-contiguous: 54-72 run sequentially by sub-group,
plus two later additions (79, 80) slotted into CPM as
notify/refund endpoints — the same pattern Transfer Kredit used for
75-78.

| # | Sub-group | Endpoint | Svc Code | Method | Path (`.../{version}/...`) |
|---|---|---|---|---|---|
|1| Direct Debit | Direct Debit Payment | 54 | POST | `debit/payment-host-to-host` |
|2| Direct Debit | Direct Debit Payment Status | 55 | POST | `debit/status` |
|3| Direct Debit | Direct Debit Payment Notification | 56 | POST | `debit/notify` |
|4| Direct Debit | Direct Debit Payment Cancel | 57 | POST | `debit/cancel` |
|5| Direct Debit | Direct Debit Payment Refund | 58 | POST | `debit/refund` |
|6| CPM | Generate QR CPM | 59 | POST | `qr/qr-cpm-generate` |
|7| CPM | CPM Payment | 60 | POST | `qr/qr-cpm-payment` |
|8| CPM | Query Payment | 61 | POST | `qr/qr-cpm-query` |
|9| CPM | Cancel Payment | 62 | POST | `qr/qr-cpm-cancel` |
|10| CPM | Payment Notification | 79 | POST | `qr/qr-cpm-notify` |
|11| CPM | Refund Payment | 80 | POST | `qr/qr-cpm-refund` |
|12| Auth Payment | Auth Payment | 63 | GET (Overview) / POST (Code Snippet — contradiction, unresolved) | `auth/payment` |
|13| Auth Payment | Payment Query | 64 | GET (Overview) / POST (Code Snippet) | `auth/query` |
|14| Auth Payment | Capture | 65 | GET (Overview) / POST (Code Snippet) | `auth/capture` |
|15| Auth Payment | Capture Query | 66 | GET (Overview) / POST (Code Snippet) | `auth/capture-query` |
|16| Auth Payment | Void | 67 | GET (Overview) / POST (Code Snippet) | `auth/void` |
|17| Auth Payment | Void Query | 68 | GET (Overview) / POST (Code Snippet) | `auth/void-query` |
|18| Auth Payment | Refund | 69 | GET (Overview) / POST (Code Snippet) | `auth/refund` |
|19| Direct Debit BI-FAST | Registrasi e-Mandate | 70 | POST | `debit/fast-emandate` |
|20| Direct Debit BI-FAST | Trigger Direct Debit Transfer (Payment) | 71 | POST | `debit/fast-payment` |
|21| Direct Debit BI-FAST | Notify | 72 | POST | `debit/fast-notify` |

**Auth Payment's method contradiction is sub-group-wide, not
per-endpoint**: all 7 endpoints in this sub-group show GET on the
Overview tab but POST with a JSON body on the Code Snippet tab. Every
worked example is a JSON request/response pair, which only makes sense
for POST — the GET marking looks like a portal-wide copy/paste default
for this sub-group specifically, not per-endpoint variation. Needs
sandbox/Postman verification before implementing; the literal-reading
convention (VA Get Report precedent) suggests POST is authoritative
here given every worked example is POST-shaped, but this is recorded as
unresolved, not decided.

## 2. Standard Header Block (all 21 endpoints)

```
Content-type: application/json
Authorization: Bearer <token>              (B2B token)
Authorization-Customer: Bearer <token>     (Direct Debit BI-FAST's page shows this even on Notify, matching Transfer Kredit's pattern)
X-TIMESTAMP, X-SIGNATURE, X-PARTNER-ID, X-EXTERNAL-ID, X-IP-ADDRESS, X-DEVICE-ID, X-LATITUDE, X-LONGITUDE, CHANNEL-ID
ORIGIN: hostname
```

Unlike Transfer Kredit, no sub-group in Transfer Debit was observed
using `X-ORIGIN` instead of bare `ORIGIN` — all 4 sub-groups' pages show
plain `ORIGIN` consistently. No endpoint-specific extra header beyond
this set was found anywhere in the group.

## 3. Shared Field Shapes

`Money`-equivalent objects (`amount`, `feeAmount`, `captureAmount`,
`voidAmount`, `refundAmount`, `maxAmount`, `transAmount`) all share the
identical two-field shape seen throughout Transfer Kredit: `{value:
String(16,2) M, currency: String(3) M}`. Every occurrence across all 4
sub-groups matches this shape exactly — no variant found.

`transactionStatus`/`latestTransactionStatus` 2-digit enum is identical
to Transfer Kredit's: `00=Success, 01=Initiated, 02=Paying, 03=Pending,
04=Refunded, 05=Canceled, 06=Failed, 07=Not found`. Appears verbatim in
Direct Debit Status/Notification, CPM Query/Notification, and Auth
Payment Query.

A second, distinct enum appears only in Auth Payment: `latestCaptureStatus`
/ `latestVoidStatus` use `INIT, SUCCESS, FAILED` (3-value, not the
8-value transaction enum) — a different status vocabulary scoped to
capture/void sub-resources, not overall transaction status.

## 4. `originalX`/`serviceCode` base pattern — present again, narrower here

Direct Debit Payment Status (55) uses `originalPartnerReferenceNo O`,
`originalReferenceNo O`, `originalExternalId O`, `serviceCode M`,
`transactionDate O`, `amount O`, plus `merchantId/subMerchantId/
externalStoreId O` — structurally the same "originalX + serviceCode"
core as Transfer Kredit's `TransactionStatusInquiryBankRequest`, but
this sub-group's own base additions (`merchantId` etc.) differ from
Transfer Kredit's (`AdditionalInfo` position, no `merchantId` there).
Whether Transfer Debit phases should define their own base type or
reuse/extend Transfer Kredit's is a design decision for implementation
time, not resolved here.

## 5. Per-Sub-Group Field Tables

### 5.1 Direct Debit (5 endpoints, POST)

**Direct Debit Payment (54)**: Req: `partnerReferenceNo String(64) M`,
`bankCardToken String(128) O`, `chargeToken String(40) O`, `otp
String(8) O`, `otpTrxCode String(2) O`, `merchantId String(64) O`,
`terminalId String(64) O`, `journeyId String(64) O`, `subMerchantId
String(32) O`, `amount O` (`amount.value M`, `amount.currency M`),
`urlParams[] O` (`url String(512) M`, `type String(32) M` — PAY_RETURN/PAY_NOTIFY,
`isDeeplink String(1) M` — Y/N), `externalStoreId String(64) O`,
`validUpTo String(25) O`, `pointOfInitiation String(20) O`, `feeType
String(25) O`, `disabledPayMethods String(64) O`, `payOptionDetails[]
O` (`payMethod M`, `payOption M`, `transAmount O`, `feeAmount O`,
`cardToken O`, `merchantToken O`, `additionalInfo O`), `additionalInfo
O`. Resp: `referenceNo String(64) C` (success only), `partnerReferenceNo
O`, `approvalCode String(20) O`, `appRedirectUrl String(2048) O`,
`webRedirectUrl String(2048) O`, `additionalInfo O`.

Note: `amount` itself is documented Optional (`O`) while its own nested
`amount.value`/`amount.currency` are documented Mandatory (`M`) — the
same "container O, members M" pattern research flagged as an
inconsistency for CPM Payment below; the package's established
handling (Optional container → `*Money`, ignore the internal-member
markers) applies without a new decision.

**Direct Debit Payment Status (55)**: see §4 above for the base
pattern. Resp adds (beyond the base): `approvalCode O`,
`latestTransactionStatus M`, `transactionStatusDesc O`,
`originalResponseCode String(7) O`, `originalResponseMessage
String(150) O`, `sessionId String(25) O`, `requestId String(25) O`,
`refundHistory[] O` (`refundNo String(64) C` success only,
`partnerRefundNo String(64) M`, `refundAmount O`, `refundStatus
String(2) M` — 00/03/06 subset of the 8-value enum, `refundDate
String(25) C` date-mandatory/time-optional per the portal's own note,
`reason String(256) O`), `transAmount O`, `feeAmount O`, `paidTime
String(25) C`.

**Direct Debit Payment Notification (56)**: Req: `originalPartnerReferenceNo
O`, `originalReferenceNo M`, `originalExternalId String(32) O`,
`merchantId O`, `subMerchantId O`, `amount O`, `latestTransactionStatus
M`, `transactionStatusDesc O`, `createdTime String(25) O`,
`finishedTime String(25) O`, `externalStoreId O`, `additionalInfo O`.
Resp: envelope + `approvalCode String(20) O` only — the field ordering
in the worked example is `responseCode, approvalCode, responseMessage`
(non-standard order, `approvalCode` between the two envelope fields),
recorded verbatim since JSON field order is not semantically
meaningful but worth noting for anyone diffing against other envelopes.

**Direct Debit Payment Cancel (57)**: Req: `originalPartnerReferenceNo
M`, `originalReferenceNo O`, `approvalCode String(20) O`,
`originalExternalId O`, `merchantId O`, `subMerchantId O`, `reason
String(256) O`, `externalStoreId O`, `amount O`, `additionalInfo O`.
Resp: `originalPartnerReferenceNo O`, `originalReferenceNo C` (success
only), `originalExternalId O`, `cancelTime String(25) C` ("required if
successful"), `transactionDate O`, `additionalInfo O`.

**Direct Debit Payment Refund (58)**: Req: `merchantId O`, `subMerchantId
O`, `originalPartnerReferenceNo M`, `originalReferenceNo O`,
`originalExternalId O`, `partnerRefundNo String(64) M`, `refundAmount
O`, `externalStoreId O`, `reason String(256) O`, `additionalInfo O`.
Resp: `originalPartnerReferenceNo O`, `originalReferenceNo C` (success
only), `originalExternalId O`, `partnerTrxId String(32) O`, `refundNo
String(64) M`, `partnerRefundNo String(64) M`, `refundAmount O`,
`refundTime String(25) M`, `additionalInfo O`.

### 5.2 CPM (6 endpoints, POST)

**Generate QR CPM (59)**: Req: `partnerReferenceNo O`, `userAccessToken
String(64) O`, `merchantId O`, `subMerchantId O`, `partnerTrxDate
String(25) M`, `additionalInfo O`. Resp: `referenceNo O`,
`partnerReferenceNo O`, `qrContent String(512) O`, `qrUrl String(255)
O`, `expiryTime String(25) M`, `additionalInfo O`. Unlike Generate QR
MPM (Transfer Kredit, service 47), `qrContent`/`qrUrl` here carry no
one-of-three conditional rule in the portal text — both are plain
Optional, and there is no `qrImage` field at all in this sub-group.

**CPM Payment (60)**: Req: `partnerReferenceNo M`, `qrContent
String(512) M`, `amount O` (members M — same container/member pattern
as Direct Debit Payment), `feeAmount O`, `merchantId M`, `subMerchantId
O`, `title String(256) O`, `expiryTime String(25) O`, `items O`
(unstructured "Object" per the portal, no item-level field table
given), `externalStoreId O`, `merchantName String(64) O`,
`merchantLocation String(64) O`, `acquirerName String(64) O`,
`terminalId String(32) O`, `scannerInfo O` (`deviceId String(64) O`,
`deviceVersion String(128) O`, `deviceModel String(128) O`, `deviceIp
String(64) O`), `additionalInfo O`. Resp: `referenceNo C` (success
only), `partnerReferenceNo O`, `transactionDate String(25) O`,
`additionalInfo O`.

**Query Payment (61)**: Req: `originalReferenceNo O`,
`originalPartnerReferenceNo O`, `originalExternalId String(32) O`,
`merchantId O`, `subMerchantId O`, `externalStoreId O`, `additionalInfo
O`. Resp: `originalReferenceNo C`, `originalPartnerReferenceNo O`,
`originalExternalId O`, `title String(256) O`, `latestTransactionStatus
M`, `transactionStatusDesc O`, `paidTime String(25) M`, `additionalInfo
O`.

**Cancel Payment (62)**: Req: `originalPartnerReferenceNo M`,
`originalReferenceNo O`, `originalExternalId O`, `merchantId O`,
`subMerchantId O`, `externalStoreId O`, `amount O`, `reason
String(256) O`, `additionalInfo O`. Resp: `originalPartnerReferenceNo
O`, `originalReferenceNo C`, `originalExternalId O`, `cancelTime
String(25) C` ("filled if successful"), `transactionDate O`,
`additionalInfo O`. Structurally identical to Transfer Kredit's
`QRMPMCancelPaymentRequest`/`Response` (Phase 24) — no `serviceCode`
field here either, all three `originalX` fields Optional, matching
that same narrower shape.

**Payment Notification (79)**: Req: `originalPartnerReferenceNo O`,
`originalReferenceNo O`, `merchantId M`, `subMerchantId O`,
`externalStoreId O`, `amount O`, `latestTransactionStatus M`,
`transactionStatusDesc O`, `customerNumber String(64) O`, `accountType
String(25) O`, `destinationNumber String(25) O`,
`destinationAccountName String(25) O`, `sessionId String(25) O`,
`bankCode String(11) O`, `additionalInfo O`. Resp: envelope-only —
matches Transfer Kredit's `QRMPMPaymentNotification` pattern exactly
(inbound settlement callback, no calling function per that
convention).

**Refund Payment (80)**: Req: `merchantId O`, `subMerchantId O`,
`externalStoreId O`, `originalPartnerReferenceNo M`, `originalReferenceNo
O`, `originalExternalId O`, `partnerRefundNo String(64) M`,
`refundAmount O`, `reason String(256) O`, `additionalInfo O`. Resp:
`originalPartnerReferenceNo O`, `originalReferenceNo O` (no C/M marker
given for this one field in this endpoint specifically — recorded as
shown), `originalExternalId O`, `refundNo String(64) M`,
`partnerRefundNo String(64) O`, `refundAmount O`, `refundTime
String(25) M`, `additionalInfo O`.

### 5.3 Auth Payment (7 endpoints, method contradiction — see §1)

A hold/capture/void/refund lifecycle: **Auth Payment** places a hold,
**Capture** charges some or all of the held amount (possibly in
multiple partial captures, `lastCapture` flag marks the final one),
**Void** releases held-but-uncaptured funds, **Refund** reverses an
already-captured amount. Each of Capture/Void has its own `*Query`
companion, mirroring the "every mutating sub-group gets a status
companion" convention already observed in Transfer Kredit.

**Auth Payment (63)**: Req: `partnerReferenceNo M`, `merchantId M`,
`subMerchantId O`, `amount.value M`, `amount.currency M` (amount itself
not separately marked, unlike other endpoints' `amount O` container
convention — recorded as shown), `feeType String(25) O` (OUR/BEN/SHA),
`mcc String(32) O`, `productCode String(64) O`, `title String(256) M`,
`items O` (list of purchased goods, no full item schema given beyond
the worked example's `goodsId`/`price`/`category`/`unit`/`quantity`),
`additionalInfo O`. Resp: `referenceNo C`, `partnerReferenceNo O`,
`amount.value M`, `amount.currency M`, `paidTime String(25) M`,
`additionalInfo O`.

**Payment Query (64)**: Req: `originalPartnerReferenceNo O`,
`originalReferenceNo O`, `merchantId O`, `subMerchantId O`,
`externalStoreId O`, `additionalInfo O`. Resp: `originalPartnerReferenceNo
O`, `originalReferenceNo O`, `amount O` (members M), `paidTime
String(25) M`, `latestTransactionStatus M`, `transactionStatusDesc
String(50) O`, `additionalInfo O`. Worked example response key-cases
`originalpartnerReferenceNo` (lowercase p) against the field table's
`originalPartnerReferenceNo` — an unresolved table-vs-example casing
contradiction, same class as several already recorded in the Transfer
Kredit doc (§6 there).

**Capture (65)**: Req: `originalReferenceNo M`, `originalPartnerReferenceNo
M`, `merchantId M`, `subMerchantId O`, `partnerCaptureNo String(64) M`,
`captureAmount O` (members M), `title String(256) M`, `lastCapture
String(8) O` ("flag to determine whether this is the last capture and
void the rest" — worked example value `"TRUE"`, a string not a JSON
boolean), `additionalInfo O`. Resp: `originalReferenceNo O`,
`originalPartnerReferenceNo O`, `partnerCaptureNo O`, `captureNo
String(64) C`, `captureAmount M` (members M, container itself
unmarked — recorded as shown), `captureTime String(25) C`,
`additionalInfo O`.

**Capture Query (66)**: Req: `originalReferenceNo M`,
`originalPartnerReferenceNo O`, `merchantId M`, `subMerchantId O`,
`captureNo String(64) O`, `partnerCaptureNo String(64) M`,
`additionalInfo O`. Resp: `originalReferenceNo O`,
`originalPartnerReferenceNo O`, `captureNo O`, `captureAmount M`
(members M), `captureTime C`, `latestCaptureStatus String(32) C` — the
3-value enum (§3), `partnerCaptureNo M`, `additionalInfo O`.

**Void (67)**: Req: `originalReferenceNo M`, `originalPartnerReferenceNo
M`, `merchantId M`, `subMerchantId O`, `voidAmount O` (members M),
`partnerVoidNo String(64) M`, `voidRemainingAmount String(8) O` (flag,
string not boolean — "TRUE" in the worked example, same convention as
`lastCapture`), `reason String(256) O`, `additionalInfo O`. Resp:
`originalReferenceNo O`, `originalPartnerReferenceNo O`, `voidNo
String(64) C`, `partnerVoidNo M`, `voidAmount M` (members M),
`voidTime String(25) C`, `additionalInfo O`.

**Void Query (68)**: Req: `originalReferenceNo M`,
`originalPartnerReferenceNo O`, `merchantId M`, `subMerchantId O`,
`voidNo String(64) O`, `partnerVoidNo String(64) M`, `additionalInfo
O`. Resp: `originalReferenceNo O`, `originalPartnerReferenceNo O`,
`voidNo O`, `voidAmount M` (members M), `voidTime C`,
`latestVoidStatus String(32) C` — the 3-value enum, `partnerVoidNo O`,
`additionalInfo O`.

**Refund (69)**: Req: `originalPartnerReferenceNo M`, `originalReferenceNo
O`, `partnerRefundNo String(64) M`, `merchantId O`, `subMerchantId O`,
`originalCaptureNo String(64) C` ("must be filled upon unsuccessful
transaction" — note this is the *opposite* condition-sense from most
Conditional fields in the package, which are typically filled on
*success*; recorded verbatim, not adjusted), `refundAmount O` (members
M), `externalStoreId O`, `reason String(256) O`, `additionalInfo O`.
Resp: `originalCaptureNo C` (same unsuccessful-transaction condition),
`originalReferenceNo C` ("must be filled upon successful transaction"
— so the response's two Conditional identifier fields have opposite
success/failure conditions from each other), `originalPartnerReferenceNo
O`, `partnerRefundNo O`, `refundNo String(64) M`, `refundAmount O`
(members M), `refundTime String(25) M`, `additionalInfo O`.

### 5.4 Direct Debit BI-FAST (3 endpoints, POST)

**Registrasi e-Mandate (70)**: Req: `partnerReferenceNo O`, `bankCode
String(11) M`, `sourceAccountNo String(19) M`, `sourceAccountName
String(100) M`, `maxAmount O` (members M), `billerId String(30) M`,
`billerName String(50) M`, `customerId String(45) M`, `expiredDatetime
String(25) M`, `additionalInfo O`. Resp: `referenceNo C`,
`partnerReferenceNo O`, `eMandateReffId String(30) M`, `additionalInfo
O`.

**Trigger Direct Debit Transfer / Payment (71)**: Req:
`partnerReferenceNo M`, `currency String(3) O`, `customerReference
String(30) M`, `feeType String(25) O`, `remark String(50) O`,
`beneficiaryAccountNo String(19) M`, `beneficiaryAccountName
String(100) M`, `transactionDate String(25) M`, `bankCode String(11)
M`, `sourceAccountNo String(34) M` — note the length discrepancy
below, `sourceAccountName String(100) M`, `amount O` (members M),
`eMandateReffId String(30) M`, `additionalInfo O`. Resp: `referenceNo
C`, `partnerReferenceNo O`, `additionalInfo O`.

**Notify (72)**: Req: `originalReferenceNo M`, `originalPartnerReferenceNo
O`, `originalExternalId String(19) O`, `transactionStatus String(2) M`
— the 8-value enum, `transactionStatusDesc String(50) O`,
`eMandateReffId String(30) M`, `sourceAccountNo String(34) M`,
`sourceAccountName String(100) M`, `amount O` (members M), `traceNo
String(16) O`, `additionalInfo O`. Resp: envelope-only. Field is named
`transactionStatus` here, not `latestTransactionStatus` as everywhere
else in both Transfer Kredit and the rest of Transfer Debit — an
unexplained naming inconsistency, recorded as shown, not silently
normalized.

**`sourceAccountNo` length contradiction within this one sub-group**:
Registrasi e-Mandate documents it as `String(19)`; Trigger Transfer and
Notify both document the same logical field as `String(34)`. No length
enforcement exists anywhere in the package regardless, so this has no
code consequence, but it's worth recording since a future reader might
otherwise assume a copy-paste error in this doc rather than the
source.

## 6. Open Contradictions to Resolve Per-Phase (not resolved in this document)

1. Auth Payment sub-group: GET (Overview, all 7 endpoints) vs POST with JSON body (Code Snippet, all 7 endpoints) — every worked example is POST-shaped; needs sandbox/Postman verification before implementing.
2. Payment Query (64, Auth Payment): worked response example casing `originalpartnerReferenceNo` (lowercase p) vs the field table's `originalPartnerReferenceNo`.
3. Auth Refund (69): the response's `originalCaptureNo` and `originalReferenceNo` are documented with opposite success/failure Conditional triggers ("must be filled upon unsuccessful transaction" vs "upon successful transaction") — genuinely two different conditions, not a copy-paste of the same note; needs verification that isn't just a documentation slip.
4. Direct Debit BI-FAST: `sourceAccountNo` documented `String(19)` (Registrasi e-Mandate) vs `String(34)` (Trigger Transfer, Notify) for what is presumably the same logical field.
5. Notify (72, Direct Debit BI-FAST): field named `transactionStatus`, not `latestTransactionStatus` as used everywhere else in both Transfer Kredit and the rest of Transfer Debit — unexplained naming inconsistency.
6. `lastCapture` (Auth Capture, 65) and `voidRemainingAmount` (Auth Void, 67) are documented `String(8)` but carry worked-example values `"TRUE"` — a boolean-shaped value in a string field, same ambiguous-type family as several Transfer Kredit fields, though here the wire shape (quoted string) is unambiguous, unlike Transfer Kredit's number-vs-string cases.
7. `Amount`-family container fields are inconsistently marked Optional at the container level while their `value`/`currency` members are marked Mandatory, across nearly every endpoint in every sub-group — the same pattern already resolved for Transfer Kredit (Optional container → `*Money`, member markers ignored), not a new decision but flagged per-occurrence above since it appears so pervasively here.
