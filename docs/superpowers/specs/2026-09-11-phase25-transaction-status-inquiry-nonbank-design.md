# Phase 25: Transaction Status Inquiry (non-bank)

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.10
(lines 208-210), Service Code 53 — the single endpoint in this
sub-group. **This completes Transfer Kredit** (all sub-groups from §5
now implemented: Virtual Account, Transaction Status Inquiry Bank,
Customer Top Up, Bulk Cash In, Transfer To Bank, Transfer To OTC,
MPM/QR, Transaction Status Inquiry non-bank).

## Naming: not QRMPM-prefixed, despite the qr/ path

Research §1's inventory table (line 56) gives this endpoint's path as
`qr/qr-mpm-status`, which looks like it belongs to the MPM/QR
sub-group. It does not — §5.10 is its own sub-group ("Transaction
Status Inquiry — non-bank"), distinct from §5.9 (MPM/QR), and the
research document itself separates them into different sections. The
path naming is the portal's own inconsistency, not a signal to treat
this as an MPM/QR endpoint. Named `TransactionStatusInquiryNonBank` —
`TransactionStatusInquiryBank` (Phase 16) already owns the unqualified
name, so this is the paired non-bank counterpart, not a `QRMPM*` type.

## Field placement: the four additions go on the request, not the response

Research line 210: "Same shape as Transaction Status Inquiry Bank (36),
plus `originalResponseCode String(7) O`, `originalResponseMessage
String(150) O`, `sessionId String(25) O`, `requestId String(25) O`."

Unlike every §5.9 row that adds response fields (each of which says so
explicitly — "Resp adds `paidTime`, `terminalId`" line 200, "Resp adds
`verificationId`" line 198, "Resp: `cancelTime C`, `transactionDate O`"
line 204), line 210 has no `Resp:` segment at all. This is inferred,
not stated: a bare "plus" with no `Resp` marker, in a document that
marks response additions every other time it means them, reads as
request-side. This is weaker evidence than Phase 23's named-base-plus-
`Resp:`-segment citation — recorded here as an inference, not a literal
transcription, so a future reader with sandbox access can correct it if
wrong.

Because nothing is added to the response, `TransactionStatusInquiryNonBankResponse`
is field-identical to `TransactionStatusInquiryBankResponse` — the
first time in the MPM/QR-adjacent group of endpoints (after Phase 23's
supersets) that this shape reappears as a genuine identical-shape
match, matching Phase 17/20's precedent (`CustomerTopUpInquiryStatus`,
`TransferToOTCTransferStatus`). Guarded with a full-equality reflection
drift test, not left as prose-only like Phase 23's superset relationship.

## Types

`TransactionStatusInquiryNonBankRequest`: `TransactionStatusInquiryBankRequest`'s
7 base fields verbatim (`OriginalPartnerReferenceNo` O, `OriginalReferenceNo`
O, `OriginalExternalID` O, `ServiceCode` M no omitempty, `TransactionDate`
O, `Amount *Money` O, `AdditionalInfo` O) plus `OriginalResponseCode
string` O (String(7)) + `OriginalResponseMessage string` O (String(150))
+ `SessionID string` O (String(25)) + `RequestID string` O (String(25)).
A superset, not an identical shape (matching Phase 23's precedent for
this same relationship), so documented in prose rather than
reflection-drift-guarded against the base request — plus a field-count
guard (see below) to catch a silently added field, following the
convention established in Phase 24.

`TransactionStatusInquiryNonBankResponse`: field-identical to
`TransactionStatusInquiryBankResponse` under its own name, per the
"distinct types per service code even for identical shapes" convention
— guarded by a full-equality `shape()` reflection test.

## Function behavior

Follows the package's standard pattern: marshal `req`, set `hb.Body`,
call `t.Do`, `checkResponseStatus`, unmarshal into `Response`, error if
`ResponseCode == ""`. POST, no method override. Read-only status query
— no non-idempotency note, matching `TransactionStatusInquiryBank`'s
precedent.

Path from research §1's inventory table: `qr/qr-mpm-status` (line 56,
despite the naming mismatch discussed above).
