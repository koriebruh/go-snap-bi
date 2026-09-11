# Transfer Kredit — ASPI SNAP Developer Portal Research

Source: https://apidevportal.aspi-indonesia.or.id/api-services/transfer-kredit and its 10 sub-group pages.
Retrieved: 2026-09-11, via browser-harness (DOM `document.body.textContent` extraction; Guides + Code Snippets tabs are both present in the DOM regardless of active tab).

This is a raw research dump, not implementation-ready prose — later phases cite it as the source of record for field tables, worked examples, and confirmed service codes. Where the portal itself contradicts across tables/examples, both readings are recorded rather than resolved here.

## 1. Full Scope: 10 Sub-Groups, 43 Endpoints

Service codes are non-contiguous: 15-53 run sequentially by sub-group, plus four later additions (75, 76, 77, 78) slotted into Trigger Transfer and MPM as notify/cancel/refund endpoints.

| # | Sub-group | Endpoint | Svc Code | Method | Path (`.../{version}/...`) |
|---|---|---|---|---|---|
|1| Account Inquiry | Internal Account Inquiry | 15 | POST | `account-inquiry-internal` |
|2| Account Inquiry | External Account Inquiry | 16 | POST | `account-inquiry-external` |
|3| Trigger Transfer | Intrabank Transfer | 17 | POST | `transfer-intrabank` |
|4| Trigger Transfer | Interbank Transfer | 18 | POST | `transfer-interbank` |
|5| Trigger Transfer | Request for Payment | 19 | POST | `transfer-request-for-payment` |
|6| Trigger Transfer | Interbank Bulk Transfer | 20 | POST | `transfer-interbank-bulk` |
|7| Trigger Transfer | Interbank Bulk Transfer - Notification | 21 | POST | `transfer-interbank-bulk/notify` |
|8| Trigger Transfer | Transfer RTGS | 22 | POST | `transfer-rtgs` |
|9| Trigger Transfer | RTGS - Notification | 76 | POST | `transfer-rtgs/notify` |
|10| Trigger Transfer | Transfer SKNBI | 23 | POST | `transfer-skn` |
|11| Trigger Transfer | SKNBI - Notification | 75 | POST | `transfer-skn/notify` |
|12| Virtual Account | VA - Inquiry | 24 | POST | `transfer-va/inquiry` |
|13| Virtual Account | VA - Payment | 25 | POST | `transfer-va/payment` |
|14| Virtual Account | VA - Inquiry Status | 26 | POST | `transfer-va/status` |
|15| Virtual Account | VA - Create VA | 27 | POST | `transfer-va/create-va` |
|16| Virtual Account | VA - Update VA | 28 | PUT | `transfer-va/update-va` |
|17| Virtual Account | VA - Update Status VA | 29 | PUT | `transfer-va/update-status` |
|18| Virtual Account | VA - Inquiry VA | 30 | POST | `transfer-va/inquiry-va` |
|19| Virtual Account | VA - Delete VA | 31 | DELETE | `transfer-va/delete-va` |
|20| Virtual Account | VA - Inquiry Payment to VA (Intrabank) | 32 | POST | `transfer-va/inquiry-intrabank` |
|21| Virtual Account | VA - Payment to VA (Intrabank) | 33 | POST | `transfer-va/payment-intrabank` |
|22| Virtual Account | VA - Notification for Payment (Intrabank) | 34 | POST | `transfer-va/notify-payment-intrabank` |
|23| Virtual Account | VA - Get Report | 35 | GET (doc) / POST (code snippet) | `transfer-va/report` |
|24| Transaction Status Inquiry Bank | Transaction Status Inquiry | 36 | POST | `transfer/status` |
|25| Customer Top Up | Account Inquiry - Customer Top Up | 37 | POST | `emoney/account-inquiry` |
|26| Customer Top Up | Customer Top Up | 38 | POST | `emoney/topup` |
|27| Customer Top Up | Customer Top Up Inquiry Status | 39 | POST | `emoney/topup-status` |
|28| Bulk Cashin | Submit Bulk Cash In | 40 | POST | `emoney/bulk-cashin-payment` |
|29| Bulk Cashin | Notify Bulk Cash In | 41 | POST | `emoney/bulk-cashin-notify` |
|30| Transfer To Bank | Account Inquiry | 42 | POST | `emoney/bank-account-inquiry` |
|31| Transfer To Bank | Payment Transaction | 43 | POST | `emoney/transfer-bank` |
|32| Transfer To OTC | Create Payment | 44 | POST | `emoney/otc-cashout` |
|33| Transfer To OTC | Transfer Status | 45 | POST | `emoney/otc-status` |
|34| Transfer To OTC | Cancel Payment | 46 | POST | `emoney/otc-cancel` (Overview) vs `otc/cashout/cancel` (Code Snippet — contradiction, unresolved) |
|35| MPM | Generate QR MPM | 47 | POST | `qr/qr-mpm-generate` |
|36| MPM | Decode QR MPM | 48 | POST | `qr/qr-mpm-decode` |
|37| MPM | Payment Redirect - Apply OTT | 49 | POST | `qr/apply-ott` |
|38| MPM | Payment - Host to Host | 50 | POST | `qr/qr-mpm-payment` |
|39| MPM | Query Payment | 51 | POST | `qr/qr-mpm-query` |
|40| MPM | Payment Notification | 52 | POST | `qr/qr-mpm-notify` |
|41| MPM | Cancel Payment | 77 | POST | `qr/qr-mpm-cancel` |
|42| MPM | Refund Payment | 78 | POST | `qr/qr-mpm-refund` |
|43| Transaction Status Inquiry (non-bank) | Transaction Status Inquiry | 53 | POST | `qr/qr-mpm-status` |

Architecturally important: every endpoint in the whole group sends identifiers in a JSON body — none uses a URL path parameter, including PUT (28, 29) and DELETE (31).

## 2. Standard Header Block (all 43 endpoints)

```
Content-type: application/json
Authorization: Bearer <token>              (B2B token)
Authorization-Customer: Bearer <token>     (appears even on server-to-server notify/callback samples — likely doc boilerplate, not verified against sandbox)
X-TIMESTAMP, X-SIGNATURE, X-PARTNER-ID, X-EXTERNAL-ID, X-IP-ADDRESS, X-DEVICE-ID, X-LATITUDE, X-LONGITUDE, CHANNEL-ID
ORIGIN / X-ORIGIN: hostname (Virtual Account + Transaction Status Inquiry Bank sub-groups use "X-ORIGIN"; all other 8 sub-groups use bare "ORIGIN" — same portal, inconsistent header name)
```

No endpoint-specific extra header beyond this set was found anywhere in the group.

## 3. Shared Field Shapes

- `Amount` object: `{value: String(16,2), currency: String(3, ISO4217)}` — used for `amount`, `feeAmount`, `paidAmount`, `totalAmount`, `refundAmount`, `minAmount`, `maxAmount`, `billAmount`, `cumulativePaymentAmount`.
- `LocalizedText` object: `{english: String, indonesia: String}` — used for `inquiryReason`, `paymentFlagReason`, `billDescription`, per-bill `reason`, `freeTexts[]` entries.
- `originatorInfos[]` (Trigger Transfer only): `{originatorCustomerNo String(34) M, originatorCustomerName String(100) M, originatorBankCode String(11) M}`, Conditional array.
- `billDetails[]` (Virtual Account family): `{billCode, billNo, billName, billShortName, billDescription{LocalizedText}, billSubCompany, billAmount{Amount}, additionalInfo, billAmountLabel, billAmountValue, billReferenceNo, status, reason{LocalizedText}}`, max 24 objects, fields present vary by endpoint.
- `additionalInfo`: Object, Optional, present on every request/response in the group.
- `transactionStatus` / `latestTransactionStatus` enum, String(2): `00 Success, 01 Initiated, 02 Paying, 03 Pending, 04 Refunded, 05 Canceled, 06 Failed, 07 Not found`.
- `virtualAccountTrxType` enum, String(1): `C=Closed, O=Open, I=Partial, M=Minimum(once), L=Maximum, N=Open Minimum(multi), X=Open Maximum(multi)`.
- Response envelope: `responseCode String(7) M`, `responseMessage String(150) M`, `responseCode = HTTPStatus(3) + ServiceCode(2) + CaseCode(2)` — confirmed across every worked example (e.g. Internal Account Inquiry → `2001500` = 200+15+00).

## 4. Group-Level Response-Code / Idempotency Notes (identical table across all 10 sub-group pages)

- `409/xx/01 Duplicate partnerReferenceNo` — dedup key is `partnerReferenceNo` (or `partnerBulkId`/`partnerRefundNo` variants per endpoint).
- `409/xx/00 Conflict` — "Cannot use same X-EXTERNAL-ID in same day" (daily-scoped dedup dimension).
- `404/xx/18 Inconsistent Request` retry semantics are documented as asymmetric per endpoint class: considered **success** for Intrabank/Interbank/RTGS/SKNBI transfer and Payment VA / Payment to VA; considered **failed** for Transfer to OTC (and, per the same table, Direct Debit payment / QR CPM payment / Auth payment / Capture, which are outside this group).
- Every sub-group with a mutating transfer has a matching `.../status` or `.../inquiry-status` companion endpoint for timeout/uncertain-outcome reconciliation.

## 5. Per-Sub-Group Field Tables

### 5.1 Account Inquiry (2 endpoints, POST)

**Internal Account Inquiry (15)**
Req: `partnerReferenceNo String(64) O`, `beneficiaryAccountNo String(34) M`, `additionalInfo O`.
Resp: `responseCode/responseMessage`, `referenceNo String(64) O`, `partnerReferenceNo O`, `beneficiaryAccountName String(100) M`, `beneficiaryAccountNo M`, `beneficiaryAccountStatus String(16) O`, `beneficiaryAccountType String(1) O` ("D"/"S"), `currency String(3) O`, `additionalInfo`.
Worked example: all types match table (all String), no ambiguity.

**External Account Inquiry (16)**
Req adds `beneficiaryBankCode String(11) M`. Resp adds `beneficiaryBankName String(50) O`, drops `beneficiaryAccountType`/`beneficiaryAccountStatus`.

### 5.2 Trigger Transfer (9 endpoints, POST)

Shared fields: `Amount`, `originatorInfos[]`, `transactionDate String(25) ISO-8601`.

**Intrabank Transfer (17)**
Req: `partnerReferenceNo M`, `amount M`, `beneficiaryAccountNo String(34) M`, `beneficiaryEmail O`, `currency O`, `customerReference String(30) O`, `feeType String(25) O` (OUR/BEN/SHA-1000), `remark O`, `sourceAccountNo String(19) M`, `transactionDate M`, `originatorInfos C`, `additionalInfo`.
Resp: `referenceNo String(64) C`, `partnerReferenceNo O`, `amount`, `beneficiaryAccountNo`, `currency`, `customerReference`, `sourceAccountNo`, `transactionDate`, `originatorInfos`, `additionalInfo`.

**Interbank Transfer (18)**: adds `beneficiaryAccountName M`, `beneficiaryAddress O`, `beneficiaryBankCode M`, `beneficiaryBankName O`. Resp adds `traceNo String(16) O`.

**Request for Payment (19)**: Req: `partnerReferenceNo M`, `bankCode String(11) M`, `beneficiaryAccountNo/Name M`, `remark O`, `expiredDatetime String(25) M`, `sourceAccountNo/Name M`, `currency O`, `amount O`, `feeType O`, `additionalInfo`. Resp: envelope + `referenceNo C` + `partnerReferenceNo O` + `additionalInfo` only.

**Interbank Bulk Transfer (20)**: Req: `partnerBulkId String(64) O`, `currency O`, `customerReference M`, `feeType O`, `remark O`, `sourceAccountNo M`, `transactionDate M`, `bulkObject[]` (each: `partnerReferenceNo M`, `bankCode M`, `beneficiaryAccountNo/Name M`, `amount M`, `originatorInfos C`, `additionalInfo`). Resp: `bulkId String(64) C`, `partnerBulkId O`, `additionalInfo`.

**Interbank Bulk Transfer - Notification (21)**: Req: `bulkId M`, `partnerBulkId M`, `bulkObject[]` (each carries per-item `responseCode`/`responseMessage` — settlement callback shape). Resp: envelope + `bulkId`/`partnerBulkId`/`additionalInfo`.

**Transfer RTGS (22)**: adds `beneficiaryCustomerResidence String(1) M` (1=Indonesia/2=Non), `beneficiaryCustomerType String(1) M` (1=Individual/2=corporation/3=Government), `kodepos String(10) O`, `receiverPhone String(20) O`, `senderCustomerResidence/Type O`, `senderPhone O`. Resp adds `transactionStatus`, `transactionStatusDesc`, `beneficiaryAccountType`.

**RTGS - Notification (76)**: Req: `originalPartnerReferenceNo/originalReferenceNo/originalExternalId O`, `latestTransactionStatus M`, `amount O`, `beneficiaryAccountName/No M`, `beneficiaryBankCode M`, `sourceAccountNo M`, `transactionDate M`, `additionalInfo`. Resp: envelope-only, no other fields.

**Transfer SKNBI (23)**: field-for-field identical shape to Transfer RTGS.

**SKNBI - Notification (75)**: identical shape to RTGS - Notification.

### 5.3 Virtual Account (12 endpoints — largest sub-group)

Shared identity triple (present on almost every endpoint): `partnerServiceId String(8) M` (derived from X-PARTNER-ID, left-padded), `customerNo String(20) M`, `virtualAccountNo String(28) M` (= partnerServiceId + customerNo).

**Response envelope top-level field name split** (confirmed in field tables and worked JSON, not a table typo):
- `virtualAccountData` (capital D): endpoints 24 (Inquiry), 25 (Payment), 26 (Inquiry Status), 27 (Create VA), 28 (Update VA), 29 (Update Status VA), 30 (Inquiry VA), 31 (Delete VA).
- `virtualAccountdata` (lowercase d): endpoints 32 (Inquiry Payment Intrabank), 33 (Payment Intrabank), 34 (Notify Payment Intrabank), 35 (Get Report).

**Type ambiguities confirmed by worked examples:**
- `customerNo` (String(20) per table) appears unquoted as a bare JSON number in worked request bodies of endpoints 26, 32, 33, 34 (e.g. `"customerNo":12345678901234567890`, 20 digits, exceeds int64 range); quoted string in endpoints 24, 25, 27, 28, 29, 30, 31.
- `channelCode` (documented `Number`): confirmed bare number on the wire (`"channelCode":6011`), consistent.
- `referenceNo` (documented String): one bare-number occurrence (`"referenceNo":123456789012345`) in the Payment-to-VA-Intrabank request; quoted string everywhere else including that endpoint's own response.
- `billReferenceNo` (documented `Number`): always quoted string in every worked example that includes it.
- `paymentType` (documented `String(1)`): bare number (`"paymentType":1`) in every worked example that includes it (VA Payment, VA Inquiry Status, Payment Intrabank).
- `partnerServiceId` documented as `"StringNumber"` (literal typo in source table) only for endpoint 34; worked example shows it quoted as a string.
- `responseCode` (String(7) everywhere else, 43/43 other cases quoted) appears unquoted/bare-number in two worked responses only: Inquiry VA (30) `"responseCode":2003000,` and Get Report (35) `"responseCode":2003500,`.

**Per-endpoint field summary (beyond identity triple / Amount / billDetails):**
- **VA Inquiry (24)**: Req adds `trxDateInit Date O`, `channelCode Number O`, `language String(2) O`, `hashedSourceAccountNo String(32) C`, `sourceBankCode String(11) C`, `passApp String(64) O`, `inquiryRequestId String(128) M`. Resp: `inquiryStatus`, `inquiryReason{LocalizedText}`, `virtualAccountName/Email/Phone`, `inquiryRequestId`, `totalAmount`, `subCompany String(5) O`, `billDetails[]`, `freeTexts[]`, `virtualAccountTrxType`, `feeAmount`.
- **VA Payment (25)**: Req adds `trxId String(64) C` (mandatory if from Create VA), `paymentRequestId String(128) M`, `channelCode`, `hashedSourceAccountNo/sourceBankCode C`, `paidAmount M`, `cumulativePaymentAmount O`, `paidBills String(6) O` (hex bitmask), `totalAmount`, `trxDateTime`, `referenceNo`, `journalNum String(6) O`, `paymentType`, `flagAdvise String(1) O`, `subCompany`, `billDetails[]`, `freeTexts[]`. Resp adds `paymentFlagReason{LocalizedText}`, `paymentFlagStatus String(2) O`, per-bill `status`/`reason`.
- **VA Inquiry Status (26)**: Req: `inquiryRequestId String(128) C` (portal text: "if not sent, returns array based on virtualAccountNo" — but response is documented as a single Object, not an array; unresolved), `paymentRequestId O`. Resp mirrors VA Payment's response shape plus `transactionDate Date O`.
- **VA Create VA (27)**: `partnerServiceId/customerNo/virtualAccountNo` are Optional here (Mandatory on every other VA endpoint). `virtualAccountName M`, `trxId String(64) M`, `totalAmount O`, `billDetails[]`, `freeTexts[]`, `virtualAccountTrxType`, `feeAmount`, `expiredDate String(25) O`.
- **VA Update VA (28, PUT)**: same request shape as Create VA but identity triple back to Mandatory. Response adds `lastUpdateDate`, `paymentDate`.
- **VA Update Status VA (29, PUT)**: `partnerServiceId/customerNo/virtualAccountNo/trxId M`, `paidStatus String(1) M` ("Y"/"N"). Response is the full VA data object.
- **VA Inquiry VA (30)**: Req is identity triple + `trxId M`. Response is full VA data object (same shape as Update VA response).
- **VA Delete VA (31, DELETE)**: Req: identity triple + `trxId String(64) O` + `additionalInfo`. Resp: identity triple + `trxId String(12) O` (length shrinks 64→12 between req/resp in the table — likely a doc typo) + `additionalInfo`. Confirms DELETE ships a JSON body, no path params.
- **VA Inquiry Payment Intrabank (32)**: adds `partnerReferenceNo String(128) O`, `sourceAccountNo/Type O` ("D"/"S"). Resp adds `productName String(30) O`, `billAmountLabel/Value`.
- **VA Payment Intrabank (33)**: adds `sourceAccountNo/Type`, `inquiryRequestId O`, `partnerReferenceNo M`, `paidAmount M`, `cumulativePaymentAmount O`, `paidBills`, `paymentStatus String(20) O` (free-text status, not the 2-digit enum).
- **VA Notify Payment Intrabank (34)**: `inquiryRequestId/paymentRequestId/partnerReferenceNo O`, `trxDateTime O`, `paymentStatus O`, `paymentFlagReason{LocalizedText}`. Callback the PJP sends outward; response mirrors most request fields plus envelope.
- **VA Get Report (35, GET per doc, POST+body per code snippet)**: Req: `partnerServiceId Number M` (flips to `Number` here vs `String` everywhere else), `startDate String(10) O` (yyyy-MM-dd), `startTime String(14) O` (HH:mm, default 00:00), `endDate/endTime` (default 23:59). Resp: `virtualAccountdata` is an Array of Objects (only VA endpoint whose top-level data is an array), each item shaped like the Payment/Inquiry-Status response object.

### 5.4 Transaction Status Inquiry Bank (1 endpoint, POST, service code 36)

Req: `originalPartnerReferenceNo/originalReferenceNo/originalExternalId O`, `serviceCode String(2) M` (points at the original transaction's service code, e.g. "17" for Intrabank), `transactionDate O`, `amount O`, `additionalInfo`.
Resp adds: `beneficiaryAccountNo String(34) M`, `beneficiaryBankCode O`, `previousResponseCode String(7) O`, `referenceNumber String(30) M`, `sourceAccountNo String(19) M`, `transactionId String(8) O` ("unique per 90 days, UTC+07, must be 8 in length"), `latestTransactionStatus M`, `transactionStatusDesc O`.

### 5.5 Customer Top Up (3 endpoints, POST)

**Account Inquiry - Customer Top Up (37)**: Req: `partnerReferenceNo O`, `customerNumber String(32) C` (mandatory if B2B2C token null), `amount M`, `transactionDate O`. Resp: `referenceNo/partnerReferenceNo O`, `sessionId String(25) O`, `customerNumber String(64) C` (masked, e.g. "XXXXXXXXX1857" — response length 64 vs request's 32), `customerName string M`, `customerMonthlyInLimit numeric O` (quoted string on wire), `minAmount/maxAmount/amount/feeAmount O`, `feeType string O`.

**Customer Top Up (38)**: Req: `partnerReferenceNo M`, `customerNumber/customerName O`, `amount/feeAmount O`, `transactionDate O`, `sessionId O`, `categoryId numeric O` (quoted string on wire), `notes string O`. Resp: `referenceNo C`, `partnerReferenceNo/sessionId/customerNumber O`, `amount O`. Worked response also includes an undocumented `referenceNumber` field ("REF993883") absent from the field table.

**Customer Top Up Inquiry Status (39)**: same originalX/serviceCode/status shape as Transaction Status Inquiry Bank.

### 5.6 Bulk Cashin (2 endpoints, POST)

**Submit Bulk Cash In (40)**: Req: `partnerBulkId O`, `transactionDate M`, `currency string O`, `bulkObject[]` (`accountNumber String(64) M`, `accountName O`, `amount O`, `partnerReferenceNo M`, `additionalInfo`), `feeType O`, `additionalInfo`. Resp: `bulkid String(64) M` (lowercase "bulkid" in this one table row, likely a typo for `bulkId` — unresolved), `partnerBulkId O`.

**Notify Bulk Cash In (41)**: Req: `bulkId/partnerBulkId M`, `bulkObject[]` (`customerNumber M`, `customerName O`, `amount O`, `referenceNo/partnerReferenceNo M`, per-item `responseCode/responseMessage M`, `additionalInfo`). Resp: envelope + `bulkId/partnerBulkId M`.

### 5.7 Transfer To Bank (2 endpoints, POST)

**Account Inquiry (42)**: Req: `partnerReferenceNo O`, `CustomerNumber String(32) M` (capitalized in this one table; every other endpoint's field is lowercase-first "customerNumber" — unresolved), `amount M`, `beneficiaryAccountNumber string O`. Resp: `accountType String(25) O`, `beneficiaryAccountNumber/Name M`, `beneficiaryBankCode/ShortName/Name O`, `amount M`, `sessionId string O`.

**Payment Transaction (43)**: Req: `partnerReferenceNo M`, `customerNumber M`, `accountType O`, `beneficiaryAccountNumber M`, `beneficiaryBankCode O`, `amount M`, `sessionId O`, `feeType O`. Resp: `referenceNo C`, `transactionDate O`, `referenceNumber string M` (distinct field from `referenceNo`, both present).

### 5.8 Transfer To OTC (3 endpoints, POST)

**Create Payment (44)**: Req: `partnerReferenceNo M`, `customerNumber M`, `otp string M` (8 chars), `amount M`, `feeType O`. Resp: `referenceNo C`, `transactionDate O`.

**Transfer Status (45)**: originalX/serviceCode/status pattern, adds `customerNumber M`, `amount M` on request.

**Cancel Payment (46)**: Req: `originalReferenceNo String(64) C`, `originalPartnerReferenceNo M`, `originalExternalId O`, `customerNumber M`, `reason String(512) M`. Resp: `originalReferenceNo M` (Mandatory in response vs Conditional in request), `cancelTime String(25) C` ("must be filled if cancelled transaction success"), `transactionDate O`. Path contradiction: Overview says `.../emoney/otc-cancel`; Code Snippet's request line reads `POST .../v1.0/otc/cashout/cancel` — unresolved, needs sandbox/Postman verification before hardcoding either path.

### 5.9 MPM / QR (8 endpoints, POST)

**Generate QR MPM (47)**: Req: `partnerReferenceNo O`, `amount/feeAmount O`, `merchantId String(64) O`, `subMerchantId String(32) O`, `storeId String(64) O`, `terminalId String(16) O`, `validityPeriod O`. Resp: `qrContent String(512) C` (conditional — "if null, qrUrl or qrImage must be filled"), `qrUrl String(256) O`, `qrImage String(unlimited) O` (base64), `redirectUrl String(512) O`, `merchantName/storeId/terminalId O`.

**Decode QR MPM (48)**: Req: `partnerReferenceNo O`, `qrContent M`, `amount O`, `merchantId/subMerchantId O`, `scanTime String(25) M`. Resp: `referenceNo String(64) C` ("Mandatory if redirect"), `redirectUrl C` ("Mandatory if H2H mode" — the two conditional labels read as describing opposite branches, unresolved), `merchantName/Category/Location C`, `merchantInfos[] M` (`merchantPAN Numeric(19) M`, quoted string on wire; `acquirerName String(50) M`), `transactionAmount/feeAmount O`.

**Payment Redirect - Apply OTT (49)**: Req: `userResources Array of String(64) M`. Resp: `userResources[] M` (`resourceType String(32) M`, `value String(64) M`). Worked example request body is `["OTT"]`.

**Payment - Host to Host (50)**: Req: `partnerReferenceNo M`, `merchantId/subMerchantId O`, `amount/feeAmount O`, `otp String(8) O`, `verificationId String(32) O`. Resp adds `verificationId String(64) O`.

**Query Payment (51)**: originalX/serviceCode pattern + `merchantId/subMerchantId/externalStoreId O`. Resp adds `paidTime String(25) C`, `terminalId O`.

**Payment Notification (52)**: Req: `originalReferenceNo M`, `originalPartnerReferenceNo O`, `latestTransactionStatus M`, `customerNumber/accountType/destinationNumber/destinationAccountName O`, `amount O`, `sessionId/bankCode/externalStoreId O`. Resp: envelope-only.

**Cancel Payment (77)**: Req: `originalX O`, `merchantId M`, `subMerchantId/externalStoreId O`, `reason string M`, `amount O`. Resp: `cancelTime C`, `transactionDate O`.

**Refund Payment (78)**: Req: `merchantId/subMerchantId/externalStoreId O`, `originalPartnerReferenceNo M`, `originalReferenceNo O`, `originalExternalId O`, `partnerRefundNo String(64) M`, `refundAmount O`, `reason String(256) O`. Resp: `refundNo String(64) M`, `partnerRefundNo O`, `refundAmount O`, `refundTime String(25) M`.

### 5.10 Transaction Status Inquiry — non-bank (1 endpoint, POST, service code 53)

Same shape as Transaction Status Inquiry Bank (36), plus `originalResponseCode String(7) O`, `originalResponseMessage String(150) O`, `sessionId String(25) O`, `requestId String(25) O`.

## 6. Open Contradictions to Resolve Per-Phase (not resolved in this document)

1. VA Get Report: GET (doc) vs POST+body (code snippet).
2. VA response envelope field name: `virtualAccountData` vs `virtualAccountdata` split across endpoints.
3. `customerNo` bare-number vs quoted-string split across VA endpoints.
4. `channelCode`/`paymentType` documented type vs wire shape.
5. Transfer to OTC - Cancel Payment: two different paths given by the same page.
6. Bulk Cashin Submit: `bulkid` vs `bulkId` field-name casing.
7. Transfer To Bank - Account Inquiry: `CustomerNumber` capitalized vs `customerNumber` elsewhere.
8. Decode QR MPM: `referenceNo`/`redirectUrl` conditional-mandatory descriptions appear to describe opposite branches.
