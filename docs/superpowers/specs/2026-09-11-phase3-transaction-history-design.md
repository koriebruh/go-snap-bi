# go-snap-bi: Phase 3 — Transaction History List (Riwayat Transaksi)

Status: draft, following the pattern established by Phase 2's Balance
Inquiry binding. Scope: one endpoint — API Transaction History List.
"Transaction History Detail" and "API Bank Statement" (siblings in the same
service group) are separate future phases, per the per-endpoint
decomposition already established.

## Source of truth

- Path: `.../{version}/transaction-history-list`
- HTTP Method: POST
- Service Code: 12, Version: 1.0

### Request body

| Field | Type | Mandatory | Length | Notes |
|---|---|---|---|---|
| partnerReferenceNo | string | O | 64 | |
| fromDateTime | string | O | 25 | ISO 8601. Default: now (DESC) or now-1mo (ASC) |
| toDateTime | string | O | 25 | ISO 8601. Default: now-1mo (DESC) or now (ASC) |
| pageSize | string | O | 2 | Default: 10. **Guides table types this `Integer`, but the standard's own worked example sends it as a quoted JSON string (`"10"`)** — modeled as `string` to match the actual wire example, not the table. |
| pageNumber | string | O | 2 | Default: 0. Same string-vs-Integer discrepancy as pageSize. |
| additionalInfo | object | O | | |

### Response body

| Field | Type | Mandatory | Notes |
|---|---|---|---|
| responseCode | string | M | |
| responseMessage | string | M | |
| referenceNo | string | O | |
| partnerReferenceNo | string | O | |
| detailData | []TransactionDetail | O | see below |
| additionalInfo | object | O | |

`TransactionDetail`: `dateTime` (ISO 8601 string), `amount` (`Money`, shared
type from Phase 2), `remark` (string), `sourceOfFunds` (`[]SourceOfFund`),
`status` (`INIT`/`SUCCESS`/`CLOSED`/`CANCELLED`), `type` (`PAYMENT`/`REFUND`/
`TOP_UP`/`SEND_MONEY`/`RECEIVE_MONEY`/`DISBURSMENT`/etc — open string, not a
closed enum per the standard's own "etc"), `additionalInfo`.

`SourceOfFund` (new shared type, per the standard's "Definisi Tipe" page):
`source` (string, e.g. `"BALANCE"`), `amount` (`Money`).

## Design

- New file `transaction_history.go` in package `snap`.
- Types: `TransactionHistoryListRequest`, `TransactionHistoryListResponse`,
  `TransactionDetail`, `SourceOfFund`. Reuses `Money` from Phase 2
  (`balance_inquiry.go`) rather than redefining it.
- One function: `TransactionHistoryList(ctx context.Context, t *Transport, hb HeaderBuilder, req TransactionHistoryListRequest) (TransactionHistoryListResponse, error)`.
  Same shape as Phase 2's `BalanceInquiry`: marshal once, set `hb.Body`,
  call `t.Do`, use `checkResponseStatus` (Phase 2's fix, already handles the
  transport-status-authoritative case) instead of re-deriving that logic,
  decode into the typed response, reject an empty `responseCode`.

## Testing

Same three-plus-one pattern as Phase 2:
- Worked-example response test (the standard's own sample, hardcoded
  independently), asserting every field via `reflect.DeepEqual` on the full
  decoded struct (not a hand-picked subset).
- Request wire-body test decoding into `map[string]any` (not the same
  request type) to catch a wrong json tag non-tautologically.
- Non-2xx `responseCode` test asserting `errors.Is` against the right
  sentinel.
- Reuses `checkResponseStatus`, so the HTTP-500-with-2xx-body and
  non-JSON-body-on-non-2xx-status cases are already covered by Phase 2's
  fix and don't need re-testing at this layer — only a build-time
  confirmation this binding actually calls it.
