# go-snap-bi: Phase 2 — Balance Inquiry (Informasi Saldo)

Status: draft, following the pattern established by the Phase 1 core design
(`docs/superpowers/specs/2026-09-10-go-snap-bi-core-design.md`). This is the
first per-service binding built on top of that core package.

Scope: one service group, one endpoint — API Balance Inquiry (Informasi
Saldo). Service Code 11, API version 1.0, per the ASPI SNAP Developer Site
("Informasi Saldo" service page) and the technical standard document.

## Source of truth

- Path: `.../{version}/balance-inquiry`
- HTTP Method: POST
- Service Code: 11, Version: 1.0

### Request body

| Field | Type | Mandatory | Length | Notes |
|---|---|---|---|---|
| partnerReferenceNo | string | O | 64 | Transaction identifier on service consumer system |
| bankCardToken | string | C | 128 | Card token; must be filled if accountNo and customer token are both null |
| accountNo | string | C | 16 | Bank account number; must be filled if bankCardToken and customer token are both null |
| balanceTypes | []string | O | — | Absent means "inquiry all balance types" |
| additionalInfo | object | O | — | Free-form |

### Response body

| Field | Type | Mandatory | Notes |
|---|---|---|---|
| responseCode | string | M | Slice 2's 7-char structured code |
| responseMessage | string | M | |
| referenceNo | string | O | |
| partnerReferenceNo | string | O | |
| accountNo | string | O | |
| name | string | O | |
| accountInfos | []AccountInfo | O | see below |
| additionalInfo | object | O | |

`AccountInfo`: `balanceType`, `amount{value,currency}`, `floatAmount{value,currency}`,
`holdAmount{value,currency}`, `availableBalance{value,currency}`,
`ledgerBalance{value,currency}`, `currentMultilateralLimit{value,currency}`,
`registrationStatusCode`, `status` (`0001`=Active, `0002`=Closed, `0004`=New,
`0006`=Restricted, `0007`=Frozen, `0009`=Dormant).

`amount`-shaped fields all share the same `{value string, currency string}`
shape (value is a decimal string, e.g. `"200000.00"`; currency is ISO 4217,
e.g. `"IDR"`) — modeled as one shared `Money` type reused across every field.

## Design

- New file `balance_inquiry.go` in package `snap` (flat package, same as
  Phase 1 — no subpackage split, consistent with that decision).
- Types: `BalanceInquiryRequest`, `BalanceInquiryResponse`, `AccountInfo`,
  `Money` (the shared amount-with-currency shape — first per-service type
  reused across future services, so it belongs at this level, not duplicated
  per-endpoint).
- One function: `BalanceInquiry(ctx context.Context, t *Transport, hb HeaderBuilder, req BalanceInquiryRequest) (BalanceInquiryResponse, error)`.
  Marshals `req` to JSON, sets it as `hb.Body`, calls `t.Do(ctx, hb)` (Slice 3's
  `Transport`), and on success unmarshals `Envelope.Raw` into
  `BalanceInquiryResponse`. On a non-2xx `Envelope.ResponseCode`, returns the
  error from `ResponseCodeError` (Slice 2) rather than a "successful" empty
  response.
- `hb.Body` must be set to the exact marshaled bytes before `Build()` signs
  it (the same single-marshal constraint documented in Phase 1's design doc)
  — `BalanceInquiry` marshals once and reuses those bytes for both signing
  and the wire body via `hb.Body = body`.

## Testing

- Table-driven test using `httptest.Server` returning the standard's own
  worked example response (captured during research), asserting every field
  unmarshals correctly, including nested `AccountInfo`/`Money`.
- A non-2xx `responseCode` test asserting the returned error matches via
  `errors.Is` against the correct Slice 2 sentinel.
- A test proving `partnerReferenceNo`/`accountNo`/`balanceTypes` round-trip
  correctly into the marshaled request body sent to the server.
