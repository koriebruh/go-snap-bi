# go-snap-bi: Phase 12 — Trigger Transfer: Remaining 7 Endpoints

Status: implemented. Scope: closes out the Trigger Transfer sub-group
(Service Codes 19, 20, 21, 22, 23, 75, 76). Batched as one phase since
fixed per-phase review overhead is largely independent of endpoint
count, and three of the seven are notification/callback shapes rather
than full new client bindings.
Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.2.

## Architecture decision: notification endpoints are inbound, not outbound

Service Codes 21 (Interbank Bulk Transfer - Notification), 75 (SKNBI -
Notification), and 76 (RTGS - Notification) are settlement callbacks:
the request carries `latestTransactionStatus`/per-item
`responseCode`/`responseMessage` fields describing an outcome, which is
information this package's caller (the PJP) *receives*, not something
it has an outbound reason to POST to the switching network itself.

This package already has generic inbound-request verification
infrastructure (`ServerVerifier.VerifyTransactionRequest`,
`IncomingRequest`) that is endpoint-agnostic — it verifies a signature
over method/URL/body/timestamp regardless of which SNAP endpoint the
inbound call is for. A caller receiving one of these three notification
calls on their own HTTP server already has everything needed to
authenticate it via that existing machinery.

Given that, this phase models these three endpoints as **plain
request/response struct pairs only — no calling function**. A caller
receiving the notification: (1) builds an `IncomingRequest` from their
HTTP framework's request, (2) calls
`ServerVerifier.VerifyTransactionRequest` to authenticate it, (3)
`json.Unmarshal`s the body into the package's typed
`*NotificationRequest`, (4) does their own application logic, (5)
marshals the package's typed `*NotificationResponse` as their HTTP
response body. No new verification code is added by this phase — the
existing generic verifier already covers it.

## RTGS/SKNBI field baseline: an inference, not a stated fact

Research §5.2 phrases Transfer RTGS (22) as "adds" fields relative to
an unstated baseline, following the same "adds" phrasing used for
Interbank Transfer (18) relative to Intrabank Transfer (17). Since RTGS
moves funds between banks (like Interbank Transfer, not Intrabank
Transfer), this design doc infers the baseline is Interbank Transfer's
full field set, not Intrabank's. This is an inference from the
research doc's own "adds" pattern, not a fact the research doc states
outright — flagged here rather than asserted as certain.

Under that inference, Transfer SKNBI (23) is "field-for-field identical
shape to Transfer RTGS" (research §5.2's own words), so its
request/response field lists are identical to RTGS's, with a distinct
service code (23) and endpoint path.

## Endpoint 1: Request for Payment (Service Code 19)

Path `.../{version}/transfer-request-for-payment`. POST.

### Request body

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerReferenceNo | String | M | `string` |
| bankCode | String(11) | M | `string` |
| beneficiaryAccountNo | String | M | `string` |
| beneficiaryAccountName | String | M | `string` |
| remark | String | O | `string` |
| expiredDatetime | String(25) | M | `string` |
| sourceAccountNo | String | M | `string` |
| sourceAccountName | String | M | `string` |
| currency | String | O | `string` |
| amount | Object | O | `*Money` |
| feeType | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body

Research §5.2 states this response is "envelope + `referenceNo C` +
`partnerReferenceNo O` + `additionalInfo` only" — no other fields.

| Field | Type | M/O | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |
| referenceNo | String | C | `string` |
| partnerReferenceNo | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

`Amount` is Optional here (unlike Intrabank/Interbank Transfer's
Mandatory `amount`), so it is `*Money` with `omitempty`.

## Endpoint 2: Interbank Bulk Transfer (Service Code 20)

Path `.../{version}/transfer-interbank-bulk`. POST.

### Shared type: InterbankBulkTransferItem

One entry in the request's `bulkObject[]` array.

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerReferenceNo | String | M | `string` |
| bankCode | String | M | `string` |
| beneficiaryAccountNo | String | M | `string` |
| beneficiaryAccountName | String | M | `string` |
| amount | Object | M | `Money` |
| originatorInfos | Array of Object | C | `[]TransferOriginatorInfo` |
| additionalInfo | Object | O | `json.RawMessage` |

### Request body

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerBulkId | String(64) | O | `string` |
| currency | String | O | `string` |
| customerReference | String | M | `string` |
| feeType | String | O | `string` |
| remark | String | O | `string` |
| sourceAccountNo | String | M | `string` |
| transactionDate | String | M | `string` |
| bulkObject | Array of Object | M | `[]InterbankBulkTransferItem` |
| additionalInfo | Object | O | `json.RawMessage` |

`BulkObject` is Mandatory (the whole point of the call) — no
`omitempty`, and a nil/empty slice still fails at the server rather
than being silently valid, matching every other Mandatory field's
zero-value-is-still-sent behavior in this package.

### Response body

| Field | Type | M/O | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |
| bulkId | String(64) | C | `string` |
| partnerBulkId | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

## Endpoint 3: Interbank Bulk Transfer - Notification (Service Code 21)

Path `.../{version}/transfer-interbank-bulk/notify`. Struct-only per
the architecture decision above — no calling function.

### Shared type: InterbankBulkTransferNotificationItem

One entry in the request's `bulkObject[]` array — distinct from
`InterbankBulkTransferItem` since this is a settlement-result callback
shape, not a transfer-instruction shape.

| Field | Type | M/O | Go type |
|---|---|---|---|
| partnerReferenceNo | String | M | `string` |
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |

### Request body (`InterbankBulkTransferNotificationRequest`)

| Field | Type | M/O | Go type |
|---|---|---|---|
| bulkId | String | M | `string` |
| partnerBulkId | String | M | `string` |
| bulkObject | Array of Object | M | `[]InterbankBulkTransferNotificationItem` |

### Response body (`InterbankBulkTransferNotificationResponse`)

| Field | Type | M/O | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |
| bulkId | String | O | `string` |
| partnerBulkId | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

## Endpoint 4: Transfer RTGS (Service Code 22)

Path `.../{version}/transfer-rtgs`. POST.

### Request body

Interbank Transfer's full field set (see Phase 11) plus:

| Field | Type | M/O | Go type |
|---|---|---|---|
| beneficiaryCustomerResidence | String(1) | M | `string` |
| beneficiaryCustomerType | String(1) | M | `string` |
| kodepos | String(10) | O | `string` |
| receiverPhone | String(20) | O | `string` |
| senderCustomerResidence | String(1) | O | `string` |
| senderCustomerType | String(1) | O | `string` |
| senderPhone | String | O | `string` |

`BeneficiaryCustomerResidence` values are "1=Indonesia/2=Non" and
`BeneficiaryCustomerType` values are "1=Individual/2=corporation/
3=Government" per the research doc — both still `string`, matching the
package's rule that a documented type can only become a non-string Go
type if the type label itself is non-string; enum-coded strings stay
`string`.

### Response body

Interbank Transfer's full response field set plus:

| Field | Type | M/O | Go type |
|---|---|---|---|
| transactionStatus | String(2) | O | `string` |
| transactionStatusDesc | String | O | `string` |
| beneficiaryAccountType | String(1) | O | `string` |

## Endpoint 5: RTGS - Notification (Service Code 76)

Path `.../{version}/transfer-rtgs/notify`. Struct-only per the
architecture decision above.

### Request body (`RTGSNotificationRequest`)

| Field | Type | M/O | Go type |
|---|---|---|---|
| originalPartnerReferenceNo | String | O | `string` |
| originalReferenceNo | String | O | `string` |
| originalExternalId | String | O | `string` |
| latestTransactionStatus | String(2) | M | `string` |
| amount | Object | O | `*Money` |
| beneficiaryAccountName | String | M | `string` |
| beneficiaryAccountNo | String | M | `string` |
| beneficiaryBankCode | String | M | `string` |
| sourceAccountNo | String | M | `string` |
| transactionDate | String | M | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

### Response body (`RTGSNotificationResponse`)

Research §5.2 states this is "envelope-only, no other fields."

| Field | Type | M/O | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |

## Endpoint 6: Transfer SKNBI (Service Code 23)

Path `.../{version}/transfer-skn`. POST. Field-for-field identical
request/response shape to Transfer RTGS (research §5.2's own words),
with a distinct Go type name (`SKNBITransferRequest`/
`SKNBITransferResponse`) and its own service code / path, matching the
package's convention of one named type per endpoint rather than a
shared top-level type across two service codes.

## Endpoint 7: SKNBI - Notification (Service Code 75)

Path `.../{version}/transfer-skn/notify`. Struct-only, field-for-field
identical shape to RTGS - Notification, with distinct Go type names
(`SKNBINotificationRequest`/`SKNBINotificationResponse`).

## Design

New shared types in `transfer_shared_types.go`:
`InterbankBulkTransferItem`, `InterbankBulkTransferNotificationItem`.

New files, one per endpoint:

- `transfer_request_for_payment.go`: `RequestForPaymentRequest`,
  `RequestForPaymentResponse`, `RequestForPayment(ctx, t, hb, req)`.
- `transfer_interbank_bulk.go`: `InterbankBulkTransferRequest`,
  `InterbankBulkTransferResponse`, `InterbankBulkTransfer(ctx, t, hb, req)`.
- `transfer_interbank_bulk_notification.go`:
  `InterbankBulkTransferNotificationRequest`,
  `InterbankBulkTransferNotificationResponse` — struct-only, no function.
- `transfer_rtgs.go`: `RTGSTransferRequest`, `RTGSTransferResponse`,
  `RTGSTransfer(ctx, t, hb, req)`.
- `transfer_rtgs_notification.go`: `RTGSNotificationRequest`,
  `RTGSNotificationResponse` — struct-only, no function.
- `transfer_sknbi.go`: `SKNBITransferRequest`, `SKNBITransferResponse`,
  `SKNBITransfer(ctx, t, hb, req)`.
- `transfer_sknbi_notification.go`: `SKNBINotificationRequest`,
  `SKNBINotificationResponse` — struct-only, no function.

`RequestForPayment`, `InterbankBulkTransfer`, `RTGSTransfer`, and
`SKNBITransfer` are mutating (fund-moving or fund-request-initiating)
POST calls and get the package's standard non-idempotency doc comment.
The three notification types carry no such comment since they are not
functions this package calls.

## Testing (mechanical precedent checks)

For each of the four calling-function endpoints: standard 5-test core
pattern (full-struct response DeepEqual, request wire round-trip,
non-2xx-responseCode, non-2xx-status-with-2xx-body,
2xx-status-with-no-responseCode) plus one
`MandatoryFieldsAlwaysSerialized` test asserting every mandatory
no-`omitempty` field's full zero-value shape, including nested `Money`/
`[]InterbankBulkTransferItem` fields. `InterbankBulkTransfer`'s
mandatory-field test asserts `BulkObject` is present on the wire as
`null` when the slice is nil-but-mandatory — Go's `encoding/json`
marshals a nil slice without `omitempty` as `null`, not `[]`, and this
test pins that actual (not idealized) wire behavior rather than
assuming an empty array.

For the three struct-only notification types: no calling function
exists to test, so tests cover only `json.Marshal`/`json.Unmarshal`
round-trips of the Request and Response types directly — one
`Test<Type>RoundTrips` test per type (6 tests total across the three
notification endpoints), asserting every field survives an unmarshal
of a worked-example-shaped fixture and every field appears on the wire
when marshaling a fully-populated value.
